package sim

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLogHeader(t *testing.T) {
	a, b := "# turtle\nhigh_block hold 20\n", "kick"
	cfg := DefaultConfig()
	l, err := Simulate(cfg, Entrant{Name: "turtle.pf", Source: a}, Entrant{Name: "rushdown.pf", Source: b})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(l.JSON(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["format_version"] != "1" || doc["config_version"] != cfg.Version || doc["frame_budget"] != float64(1200) {
		t.Errorf("header %v %v %v", doc["format_version"], doc["config_version"], doc["frame_budget"])
	}
	players := doc["players"].([]any)
	p1 := players[0].(map[string]any)
	if p1["id"] != "p1" || p1["name"] != "turtle.pf" || p1["script"] != a {
		t.Errorf("players[0] = %v", p1)
	}
	if players[1].(map[string]any)["id"] != "p2" {
		t.Errorf("players[1] = %v", players[1])
	}
	res := doc["result"].(map[string]any)
	for _, k := range []string{"winner", "reason", "end_frame", "hp"} {
		if _, ok := res[k]; !ok {
			t.Errorf("result missing %q", k)
		}
	}
}

func TestLogDrawWinnerIsNull(t *testing.T) {
	l := run(t, "", "", frames(10))
	if !strings.Contains(string(l.JSON()), `"winner":null`) {
		t.Errorf("draw must encode winner as null")
	}
}

func TestLogCustomFrameBudget(t *testing.T) {
	l := run(t, "loop { move_forward }", "", frames(300))
	if l.FrameBudget != 300 || len(l.Frames) > 300 {
		t.Errorf("budget %d frames %d", l.FrameBudget, len(l.Frames))
	}
}

func TestLogFrameIndexing(t *testing.T) {
	l := run(t, "", "")
	if len(l.Frames) != 600 {
		t.Fatalf("frames %d", len(l.Frames))
	}
	for i, f := range l.Frames {
		if f.F != i {
			t.Fatalf("frames[%d].f = %d", i, f.F)
		}
	}
}

func TestLogFighterFrameFields(t *testing.T) {
	l := run(t, "", "", frames(1))
	var doc struct {
		Frames []struct {
			P1 map[string]any `json:"p1"`
			P2 map[string]any `json:"p2"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(l.JSON(), &doc); err != nil {
		t.Fatal(err)
	}
	p1 := doc.Frames[0].P1
	for _, k := range []string{"x", "y", "facing", "state", "anim", "anim_frame", "phase", "hp", "invulnerable", "combo", "script_line"} {
		if _, ok := p1[k]; !ok {
			t.Errorf("frame entry missing %q", k)
		}
	}
	if p1["facing"] != float64(1) || doc.Frames[0].P2["facing"] != float64(-1) {
		t.Errorf("facing p1 %v p2 %v", p1["facing"], doc.Frames[0].P2["facing"])
	}
}

func TestLogScriptLineDuringCheck(t *testing.T) {
	src := "wait 1\nwait 1\nwait 1\nif enemy.ducking { kick }\n"
	l := run(t, src, "", frames(10))
	// The read on line 4 covers frames 3-4.
	for _, f := range []int{3, 4} {
		if ff := p(l, f, 0); ff.ScriptLine != 4 || ff.Anim != "check" {
			t.Errorf("frame %d: line %d anim %q, want 4/check", f, ff.ScriptLine, ff.Anim)
		}
	}
	if ff := p(l, 1, 0); ff.ScriptLine != 2 || ff.Anim != "wait" {
		t.Errorf("frame 1: %+v", ff)
	}
	// During a stance the stance animation is shown instead.
	l = run(t, "jump\nif enemy.ducking { kick }", "", frames(4))
	if ff := p(l, 2, 0); ff.ScriptLine != 2 || ff.Anim != "jump" {
		t.Errorf("check during jump: %+v", ff)
	}
}

func TestLogAnimFrame(t *testing.T) {
	l := run(t, "kick", "", frames(20))
	for f := 0; f < 18; f++ {
		if got := p(l, f, 0).AnimFrame; got != f {
			t.Errorf("frame %d anim_frame %d", f, got)
		}
	}
	if got := p(l, 18, 0); got.Anim != "idle" || got.AnimFrame != 0 {
		t.Errorf("idle after kick: %+v", got)
	}
}

func TestLogRendererNeedsNoGameLogic(t *testing.T) {
	l := run(t, "jump_forward kick", "", frames(10))
	ff := p(l, 4, 0) // jump_kick startup 1-3, active from frame 4; x moves 1/airborne frame from frame 0
	if ff.State != "airborne" || ff.Anim != "jump_kick" || ff.Phase != "active" || ff.X != 35 || ff.Y != 10 {
		t.Errorf("got %+v", ff)
	}
}

func TestLogCheckFalseSerializes(t *testing.T) {
	l := run(t, "if enemy.airborne { kick }", "", frames(3))
	c := events(l, "check", "p1")
	if len(c) != 1 {
		t.Fatalf("checks %v", c)
	}
	raw := string(encode(c[0], false))
	if !strings.Contains(raw, `"value":false`) {
		t.Errorf("false check value dropped: %s", raw)
	}
	if !strings.Contains(raw, `"sensor":"enemy.airborne"`) || !strings.Contains(raw, `"cost":2`) || !strings.Contains(raw, `"line":1`) {
		t.Errorf("check detail %s", raw)
	}
}

func TestLogZeroHPSerializes(t *testing.T) {
	l := run(t, "punch", "", startX(30, 36), func(c *Config) { c.MaxHP = 5 })
	h := events(l, "hit", "p1")
	if raw := string(encode(h[0], false)); !strings.Contains(raw, `"hp":0`) {
		t.Errorf("final hp 0 dropped: %s", raw)
	}
}

func TestLogEventOrdering(t *testing.T) {
	l := run(t, "punch", "punch", startX(30, 37), frames(20))
	var got []string
	for _, e := range l.Events {
		if e.Frame == 3 {
			got = append(got, e.Actor+":"+e.Type)
		}
	}
	if want := "p1:hit p1:trade p2:hit"; strings.Join(got, " ") != want {
		t.Errorf("frame 3 events %v, want %s", got, want)
	}
	for i := 1; i < len(l.Events); i++ {
		if l.Events[i].Frame < l.Events[i-1].Frame {
			t.Fatalf("events not ordered by frame at %d", i)
		}
	}
}

func TestLogScriptErrorPassthrough(t *testing.T) {
	_, err := Simulate(DefaultConfig(), Entrant{Name: "ok", Source: "kick"}, Entrant{Name: "bad", Source: "kik"})
	se, ok := err.(*ScriptError)
	if !ok || se.Player != "p2" || len(se.Errors) != 1 {
		t.Fatalf("err = %v", err)
	}
}
