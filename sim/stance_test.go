package sim

import (
	"strings"
	"testing"
)

func rep(tok string, n int) string { return strings.TrimSpace(strings.Repeat(tok+" ", n)) }

func TestStanceJumpKickOverlay(t *testing.T) {
	l := run(t, "jump_forward kick", "", frames(25))
	want := strings.Join([]string{"jump", rep("jump_kick/s", 3), rep("jump_kick/a", 5), rep("jump", 11), rep("landing", 2), "idle"}, " ")
	if got := animRow(l, 0, 0, 22); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	for f := 0; f < 20; f++ {
		if p(l, f, 0).State != "airborne" {
			t.Errorf("frame %d: state %s, want airborne (20 airborne frames)", f, p(l, f, 0).State)
		}
	}
}

func TestStanceOneAirAttackPerJump(t *testing.T) {
	l := run(t, "jump kick kick", "", frames(40))
	if got := p(l, 1, 0).Anim; got != "jump_kick" {
		t.Fatalf("frame 1 anim %q, want jump_kick", got)
	}
	// Second kick waits for 20 airborne + 2 landing frames, then is a grounded kick.
	if f := firstFrame(l, 0, 2, anim("kick")); f != 22 {
		t.Errorf("grounded kick starts on frame %d, want 22", f)
	}
	if got := p(l, 22, 0).State; got != "attack" {
		t.Errorf("frame 22 state %q, want attack", got)
	}
}

func TestStanceDivePunchOverlay(t *testing.T) {
	l := run(t, "jump_back punch", "", frames(5))
	if got := p(l, 1, 0).Anim; got != "dive_punch" {
		t.Errorf("frame 1 anim %q, want dive_punch", got)
	}
}

func TestStanceLowJabOverlay(t *testing.T) {
	l := run(t, "duck hold 20 punch", "", frames(30))
	// Duck startup is 2 frames (0-1); the punch waits for the hold phase.
	want := strings.Join([]string{rep("duck/s", 2), rep("low_jab/s", 3), rep("low_jab/a", 2), rep("low_jab/r", 5), rep("duck/a", 10), rep("duck/r", 2), "idle"}, " ")
	if got := animRow(l, 0, 0, 24); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	for f := 2; f <= 11; f++ {
		if p(l, f, 0).State != "duck" {
			t.Errorf("frame %d state %q, want duck while jabbing", f, p(l, f, 0).State)
		}
	}
}

func TestStanceMultipleLowJabs(t *testing.T) {
	l := run(t, "duck hold 30 punch punch", "", frames(30))
	if f := firstFrame(l, 0, 12, anim("low_jab")); f != 12 {
		t.Errorf("second low_jab starts on frame %d, want 12", f)
	}
}

func TestStanceLowJabExtendsHold(t *testing.T) {
	// The hold (3 frames, 2-4) would expire during the jab (frames 2-11).
	l := run(t, "duck hold 3 punch", "", frames(20))
	want := strings.Join([]string{rep("duck/s", 2), rep("low_jab/s", 3), rep("low_jab/a", 2), rep("low_jab/r", 5), rep("duck/r", 2), "idle"}, " ")
	if got := animRow(l, 0, 0, 14); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestStanceNonOverlayWaits(t *testing.T) {
	l := run(t, "jump move_forward x2", "", frames(30))
	if f := firstFrame(l, 0, 0, anim("walk_forward")); f != 22 {
		t.Errorf("movement starts on frame %d, want 22 (after 20 airborne + 2 landing)", f)
	}
	if x := p(l, 23, 0).X; x != 32 {
		t.Errorf("x after walking = %d, want 32", x)
	}
}

func TestStanceChecksRunDuringStance(t *testing.T) {
	// distance starts at 40 and shrinks by 1 per airborne frame of jump_forward.
	l := run(t, "jump_forward if distance < 50 { kick }", "", frames(25))
	c := events(l, "check", "p1")
	if len(c) != 1 || c[0].Frame != 1 {
		t.Fatalf("checks = %v, want one read starting frame 1", c)
	}
	if got := animRow(l, 0, 0, 3); got != "jump jump jump jump_kick/s" {
		t.Errorf("frames 0-3: %s", got)
	}
	if got := p(l, 3, 0).Anim; got != "jump_kick" {
		t.Errorf("frame 3 anim %q, want jump_kick", got)
	}
}

func TestStanceWaitIsConcurrent(t *testing.T) {
	// wait runs during the jump, so the air attack can start mid-jump.
	l := run(t, "jump wait 5 kick", "", frames(25))
	if f := firstFrame(l, 0, 0, anim("jump_kick")); f != 6 {
		t.Errorf("jump_kick starts on frame %d, want 6", f)
	}
}

func TestStanceAirAttackEndsOnLanding(t *testing.T) {
	// kick on airborne frame 16 (frame 16): jump_kick would last to frame 23
	// but the jump lands at frame 20.
	l := run(t, "jump wait 15 kick wait 1", "", frames(30))
	if got := animRow(l, 0, 16, 22); got != "jump_kick/s jump_kick/s jump_kick/s jump_kick/a landing landing idle" {
		t.Errorf("got %s", got)
	}
	// The trailing wait runs concurrently with landing recovery (frame 20).
	if got := events(l, "script_end", "p1"); len(got) != 1 || got[0].Frame != 21 {
		t.Errorf("script_end = %v, want frame 21", got)
	}
}

func TestStanceStartCostsOneFrame(t *testing.T) {
	l := run(t, "duck wait 1", "", frames(5))
	// duck on frame 0, wait (concurrent) on frame 1
	if got := p(l, 1, 0).ScriptLine; got != 1 {
		t.Errorf("script line %d", got)
	}
	if got := events(l, "script_end", "p1"); len(got) != 1 || got[0].Frame != 2 {
		t.Errorf("script_end = %v, want frame 2", got)
	}
}
