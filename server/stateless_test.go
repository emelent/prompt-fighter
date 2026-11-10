package server

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"promptfighter/sim"
)

func TestHealth(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "GET", "/healthz", "")
	var b map[string]any
	decodeJSON(t, rec, &b)
	if rec.Code != 200 || b["status"] != "ok" || b["config_version"] != sim.DefaultConfig().Version || b["format_version"] != sim.FormatVersion || b["store_version"] != float64(1) {
		t.Errorf("%d %v", rec.Code, b)
	}
	if b["config_version"] != "2026.11-2" || b["format_version"] != "1" {
		t.Errorf("versions %v", b)
	}
}

type validateResp struct {
	Valid  bool          `json:"valid"`
	Errors []ScriptError `json:"errors"`
}

func TestValidateValid(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "POST", "/validate", `{"script": "loop { kick }"}`)
	var v validateResp
	decodeJSON(t, rec, &v)
	if rec.Code != 200 || !v.Valid || v.Errors == nil || len(v.Errors) != 0 {
		t.Errorf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestValidateInvalid(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "POST", "/validate", `{"script": `+jsonStr("punch\n\n    kik")+`}`)
	var v validateResp
	decodeJSON(t, rec, &v)
	if rec.Code != 200 || v.Valid || len(v.Errors) != 1 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	e := v.Errors[0]
	if e.Line != 3 || e.Col != 5 || e.Text != `3:5: unknown action "kik" — did you mean "kick"?` || e.Message != `unknown action "kik" — did you mean "kick"?` || e.Player != "" {
		t.Errorf("error %+v", e)
	}
}

func simBody(p1, p2 string, extra string) string {
	return `{"p1": {"name": "alice", "script": ` + jsonStr(p1) + `}, "p2": {"script": ` + jsonStr(p2) + `}` + extra + `}`
}

func TestSimMatchesEngine(t *testing.T) {
	s := newTestServer(t)
	a, b := "loop { move_forward x2 kick }", "loop { if distance < 13 { high_block hold 8 } }"
	rec := do(t, s, "POST", "/sim", simBody(a, b, ""))
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	l, err := sim.Simulate(sim.DefaultConfig(), sim.Entrant{Name: "alice", Source: a}, sim.Entrant{Name: "p2", Source: b})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rec.Body.Bytes(), l.JSON()) {
		t.Errorf("body differs from sim.Simulate output")
	}
}

func TestSimCustomFrames(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "POST", "/sim", simBody("wait 1", "wait 1", `, "frames": 120`))
	var l sim.Log
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil || rec.Code != 200 || l.FrameBudget != 120 || len(l.Frames) != 120 {
		t.Errorf("%d budget %d frames %d", rec.Code, l.FrameBudget, len(l.Frames))
	}
}

func TestSimFramesOutOfRange(t *testing.T) {
	s := newTestServer(t)
	expectError(t, do(t, s, "POST", "/sim", simBody("kick", "kick", `, "frames": 0`)), 400, "bad_request")
	expectError(t, do(t, s, "POST", "/sim", simBody("kick", "kick", `, "frames": 601`)), 400, "bad_request")
	if rec := do(t, s, "POST", "/sim", simBody("kick", "kick", `, "frames": 600`)); rec.Code != 200 {
		t.Errorf("frames 600 should be allowed: %d", rec.Code)
	}
}

func TestSimInvalidScript(t *testing.T) {
	s := newTestServer(t)
	b := expectError(t, do(t, s, "POST", "/sim", simBody("kick", "kik", "")), 422, "invalid_script")
	if len(b.Errors) != 1 || b.Errors[0].Player != "p2" || b.Errors[0].Text != `1:1: unknown action "kik" — did you mean "kick"?` {
		t.Errorf("errors %+v", b.Errors)
	}
}

func TestSimRequestValidation(t *testing.T) {
	s := newTestServer(t)
	expectError(t, do(t, s, "POST", "/sim", `{"p1": {"script": "kick"}}`), 400, "bad_request")
	long := `{"p1": {"name": "` + string(bytes.Repeat([]byte("x"), 41)) + `", "script": "kick"}, "p2": {"script": "kick"}}`
	expectError(t, do(t, s, "POST", "/sim", long), 400, "bad_request")
}

func TestSimBusy(t *testing.T) {
	s := newTestServer(t, func(o *Options) { o.MaxSims, o.SimWait = 1, 50*time.Millisecond })
	s.sims <- struct{}{} // hold the only slot
	defer s.release()
	start := time.Now()
	expectError(t, do(t, s, "POST", "/sim", simBody("kick", "kick", "")), 503, "busy")
	if d := time.Since(start); d < 50*time.Millisecond {
		t.Errorf("returned after %v, before the wait elapsed", d)
	}
}
