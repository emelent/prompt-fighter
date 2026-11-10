package sim

import "testing"

// classicStun pins hitstun and pushback so these interrupt/resume scenarios
// keep fixed frame numbers when the balance values are retuned.
func classicStun(c *Config) { c.Hitstun, c.HitPushback = 5, 3 }

func handlerEnters(l *Log, actor string) []int {
	var fs []int
	for _, e := range events(l, "handler_enter", actor) {
		fs = append(fs, e.Frame)
	}
	return fs
}

func TestHandlerResumeAfterHitstunWithoutHandler(t *testing.T) {
	// p2's punch (active frame 3, range 8) hits p1 during kick startup.
	l := run(t, "kick move_back x2 punch", "punch", startX(30, 37), frames(40), classicStun)
	if h := events(l, "hit", "p2"); len(h) != 1 || h[0].Frame != 3 {
		t.Fatalf("hits = %v", h)
	}
	for f := 3; f <= 8; f++ {
		if p(l, f, 0).State != "hitstun" {
			t.Errorf("frame %d state %q, want hitstun", f, p(l, f, 0).State)
		}
	}
	if got := animRow(l, 0, 9, 11); got != "walk_back walk_back punch/s" {
		t.Errorf("after hitstun: %s, want move_back x2 then punch", got)
	}
	if n := len(events(l, "whiff", "p1")); n != 1 { // only the trailing punch
		t.Errorf("p1 whiffs = %d, the interrupted kick must not whiff", n)
	}
}

func TestHandlerOnHitThenResume(t *testing.T) {
	l := run(t, "wait 2 loop { move_forward x3 kick } on_hit { move_back x2 }", "punch", startX(30, 38), frames(40), classicStun)
	if h := events(l, "hit", "p2"); len(h) != 1 || h[0].Frame != 3 {
		t.Fatalf("hits = %v (want p1 hit during move_forward on frame 3)", h)
	}
	if got := handlerEnters(l, "p1"); len(got) != 1 || got[0] != 9 {
		t.Fatalf("handler_enter frames = %v, want [9]", got)
	}
	if d := events(l, "handler_enter", "p1")[0].Detail.Handler; d != "on_hit" {
		t.Errorf("handler = %q", d)
	}
	if got := animRow(l, 0, 9, 11); got != "walk_back walk_back kick/s" {
		t.Errorf("got %s, want on_hit's move_back x2 then the kick after move_forward", got)
	}
}

func TestHandlerOnBlock(t *testing.T) {
	l := run(t, "kick wait 1 on_block { throw }", "high_block hold 30", startX(30, 40), frames(45), classicStun)
	b := events(l, "block", "p1")
	if len(b) != 1 || b[0].Frame != 6 {
		t.Fatalf("blocks = %v", b)
	}
	// Attacker blockstun = kick recovery (9 frames): frames 6..15.
	for f := 6; f <= 15; f++ {
		if p(l, f, 0).State != "blockstun" {
			t.Errorf("frame %d state %q, want blockstun", f, p(l, f, 0).State)
		}
	}
	if got := handlerEnters(l, "p1"); len(got) != 1 || got[0] != 16 {
		t.Fatalf("handler_enter = %v, want [16]", got)
	}
	if got := p(l, 16, 0).Anim; got != "throw" {
		t.Errorf("frame 16 anim %q, want throw", got)
	}
	// throw lasts 24 frames (16..39), then the statement after the kick.
	if got := p(l, 40, 0).Anim; got != "wait" {
		t.Errorf("frame 40 anim %q, want wait", got)
	}
}

func TestHandlerInterruptDuringCondition(t *testing.T) {
	// The read covers frames 2-3; p2's punch hits on frame 3.
	l := run(t, "wait 2 if enemy.ducking { sweep } else { kick } wait 5", "punch", startX(30, 37), frames(30), classicStun)
	if h := events(l, "hit", "p2"); len(h) != 1 || h[0].Frame != 3 {
		t.Fatalf("hits = %v", h)
	}
	if got := animRow(l, 0, 9, 13); got != "wait wait wait wait wait" {
		t.Errorf("after hitstun: %s, want the whole if chain skipped", got)
	}
}

func TestHandlerInterruptInsideHandler(t *testing.T) {
	// First hit at frame 3; on_hit starts at 9. p2 walks in and hits again at 16.
	l := run(t, "kick wait 1 on_hit { wait 10 }", "punch move_forward x3 punch", startX(30, 37), frames(40), classicStun)
	hits := events(l, "hit", "p2")
	if len(hits) != 2 || hits[0].Frame != 3 || hits[1].Frame != 16 {
		t.Fatalf("hits = %v", hits)
	}
	if got := handlerEnters(l, "p1"); len(got) != 2 || got[0] != 9 || got[1] != 22 {
		t.Fatalf("handler_enter = %v, want [9 22]", got)
	}
	// The restarted handler waits 10 frames (22-31), then main resumes after the kick.
	if got := p(l, 32, 0).Anim; got != "wait" {
		t.Errorf("frame 32 anim %q, want wait (statement after kick)", got)
	}
	if e := events(l, "script_end", "p1"); len(e) != 1 || e[0].Frame != 33 {
		t.Errorf("script_end = %v, want frame 33", e)
	}
}

func TestHandlerScriptEndIdles(t *testing.T) {
	l := run(t, "kick", "", classicStun)
	e := events(l, "script_end", "p1")
	if len(e) != 1 || e[0].Frame != 18 {
		t.Fatalf("script_end = %v, want one at frame 18", e)
	}
	for f := 18; f < 600; f++ {
		if ff := p(l, f, 0); ff.State != "idle" || ff.Anim != "idle" || ff.ScriptLine != 0 {
			t.Fatalf("frame %d: %+v, want idle", f, ff)
		}
	}
}

func TestHandlerRunsAfterScriptEnd(t *testing.T) {
	l := run(t, "wait 1 on_hit { move_back x2 }", "wait 5 punch", startX(30, 37), frames(30), classicStun)
	if h := events(l, "hit", "p2"); len(h) != 1 || h[0].Frame != 8 {
		t.Fatalf("hits = %v", h)
	}
	if got := handlerEnters(l, "p1"); len(got) != 1 || got[0] != 14 {
		t.Fatalf("handler_enter = %v", got)
	}
	if got := animRow(l, 0, 14, 16); got != "walk_back walk_back idle" {
		t.Errorf("got %s", got)
	}
	if e := events(l, "script_end", "p1"); len(e) != 1 {
		t.Errorf("script_end emitted %d times, want once", len(e))
	}
}
