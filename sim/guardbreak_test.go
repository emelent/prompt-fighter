package sim

import "testing"

// noBlockPush keeps the fighters at punch range while one blocks repeatedly.
func noBlockPush(c *Config) { c.BlockPushback = 0 }

func TestGuardBreakOnFourthBlock(t *testing.T) {
	l := run(t, "repeat 4 { punch } kick", "high_block hold 100", startX(30, 36), frames(120), noBlockPush)
	if n := len(events(l, "block", "p1")); n != 3 {
		t.Fatalf("blocks = %d, want 3 before the break", n)
	}
	gb := events(l, "guard_break", "p1")
	if len(gb) != 1 || gb[0].Target != "p2" || gb[0].Detail.Attack != "punch" {
		t.Fatalf("guard breaks = %v", gb)
	}
	f := gb[0].Frame
	for k := f; k <= f+19; k++ {
		if s := p(l, k, 1).State; s != "guard_break" && s != "hitstun" {
			t.Fatalf("frame %d: p2 state %q, want guard_break stun", k, s)
		}
	}
	if p(l, f, 1).HP != 100 {
		t.Errorf("guard break dealt damage")
	}
	// The attacker is not put in blockstun: it finishes its punch normally.
	if s := p(l, f+1, 0); s.State == "blockstun" {
		t.Errorf("attacker in blockstun after a guard break")
	}
	// ... and the follow-up kick punishes the open defender.
	h := events(l, "hit", "p1")
	if len(h) != 1 || h[0].Detail.Attack != "kick" || h[0].Frame > f+20 {
		t.Fatalf("hits = %v, want the punishing kick within the stun", h)
	}
}

func TestGuardBreakStreakLapses(t *testing.T) {
	l := run(t, "repeat 3 { punch } wait 60 punch", "high_block hold 100", startX(30, 36), frames(120), noBlockPush)
	if gb := events(l, "guard_break", ""); len(gb) != 0 {
		t.Fatalf("guard break after a 60-frame gap: %v", gb)
	}
	if n := len(events(l, "block", "p1")); n != 4 {
		t.Errorf("blocks = %d, want 4", n)
	}
}

func TestGuardBreakAttackingResetsStreak(t *testing.T) {
	p1 := "repeat 3 { punch } wait 30 repeat 2 { punch }"
	// p2 guards 3 punches, starts a throw (which whiffs at distance 6, beyond
	// throw range 5) and guards the last 2. Its streak restarts at the throw.
	l := run(t, p1, "high_block hold 26 throw high_block hold 60", startX(30, 36), frames(120), noBlockPush)
	if gb := events(l, "guard_break", ""); len(gb) != 0 {
		t.Fatalf("guard broken although p2 attacked in between: %v", gb)
	}
	if n := len(events(l, "block", "p1")); n != 5 {
		t.Fatalf("blocks = %d, want 5", n)
	}
	// Control: the same timing without the throw breaks the guard.
	c := run(t, p1, "high_block hold 26 wait 24 high_block hold 60", startX(30, 36), frames(120), noBlockPush)
	if gb := events(c, "guard_break", ""); len(gb) != 1 {
		t.Fatalf("control: guard breaks = %v, want 1", gb)
	}
}

func TestGuardBreakRunsOnHit(t *testing.T) {
	l := run(t, "repeat 4 { punch }", "high_block hold 100\non_hit { move_back x3 }", startX(30, 36), frames(80), noBlockPush)
	gb := events(l, "guard_break", "p1")
	if len(gb) != 1 {
		t.Fatalf("guard breaks = %v", gb)
	}
	he := events(l, "handler_enter", "p2")
	if len(he) != 1 || he[0].Detail.Handler != "on_hit" || he[0].Frame != gb[0].Frame+21 {
		t.Fatalf("handler_enter = %v, want on_hit right after the 20-frame stun (frame %d)", he, gb[0].Frame+21)
	}
	if a := p(l, he[0].Frame+1, 1).Anim; a != "walk_back" {
		t.Errorf("after on_hit: anim %q, want walk_back", a)
	}
}

func TestGuardBreakSwappedSidesMirror(t *testing.T) {
	att, def := "repeat 4 { punch } kick", "high_block hold 100"
	a := run(t, att, def, startX(30, 36), frames(80), noBlockPush)
	b := run(t, def, att, startX(64, 70), frames(80), noBlockPush)
	if len(events(a, "guard_break", "p1")) != 1 || len(events(b, "guard_break", "p2")) != 1 {
		t.Fatalf("want one guard break on each side")
	}
	for f := range a.Frames {
		x, y := a.Frames[f], b.Frames[f]
		if x.P1.X != 100-y.P2.X || x.P1.State != y.P2.State || x.P2.State != y.P1.State || x.P2.HP != y.P1.HP {
			t.Fatalf("frame %d not mirrored: %+v %+v vs %+v %+v", f, x.P1, x.P2, y.P1, y.P2)
		}
	}
}
