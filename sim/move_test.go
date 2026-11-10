package sim

import "testing"

func TestMoveEdgeClamp(t *testing.T) {
	l := run(t, "move_back x5", "", startX(3, 70), frames(6))
	if p(l, 0, 0).Facing != 1 {
		t.Fatalf("p1 should face right")
	}
	for f := 0; f < 5; f++ {
		if x := p(l, f, 0).X; x != 2 {
			t.Errorf("frame %d x=%d, want 2", f, x)
		}
	}
	if e := events(l, "script_end", "p1"); len(e) != 1 || e[0].Frame != 5 {
		t.Errorf("all 5 move frames should still be consumed, script_end %v", e)
	}
}

func TestMoveNoOverlapBothMoving(t *testing.T) {
	l := run(t, "move_forward", "move_forward", startX(48, 53), frames(2))
	if a, b := p(l, 0, 0).X, p(l, 0, 1).X; a != 48 || b != 53 {
		t.Errorf("x = %d,%d, want both moves cancelled (48,53)", a, b)
	}
}

func TestMoveBothMovingStopsAtMinSeparation(t *testing.T) {
	// 6 apart: one frame brings them to 4, the next would overlap.
	l := run(t, "move_forward x3", "move_forward x3", startX(47, 53), frames(4))
	if d := p(l, 2, 1).X - p(l, 2, 0).X; d != 4 {
		t.Errorf("distance %d, want 4", d)
	}
}

func TestMoveWalkIntoOpponent(t *testing.T) {
	l := run(t, "move_forward x3 wait 1", "", startX(30, 34), frames(5))
	for f := 0; f < 3; f++ {
		if x := p(l, f, 0).X; x != 30 {
			t.Errorf("frame %d x=%d, want 30", f, x)
		}
	}
	if got := p(l, 3, 0).Anim; got != "wait" {
		t.Errorf("frame 3 anim %q, want wait (3 frames consumed)", got)
	}
}

func TestMoveMoveBackGoesAway(t *testing.T) {
	l := run(t, "", "move_back x3", frames(3))
	if x := p(l, 2, 1).X; x != 73 {
		t.Errorf("p2 x=%d, want 73", x)
	}
}

func TestMoveJumpForwardDistanceAndPeak(t *testing.T) {
	l := run(t, "jump_forward", "", frames(25))
	peak := 0
	for f := 0; f < 20; f++ {
		peak = max(peak, p(l, f, 0).Y)
	}
	if peak != 20 {
		t.Errorf("peak y=%d, want 20", peak)
	}
	if y := p(l, 9, 0).Y; y != 20 {
		t.Errorf("y on airborne frame 10 = %d, want 20", y)
	}
	if ff := p(l, 20, 0); ff.X != 50 || ff.Y != 0 || ff.State != "landing" {
		t.Errorf("after landing: %+v, want x=50 y=0 landing", ff)
	}
}

func TestMoveFacingConstantWhileAirborne(t *testing.T) {
	l := run(t, "jump", "", frames(22))
	for f := 0; f < 20; f++ {
		if p(l, f, 0).Facing != 1 || p(l, f, 1).Facing != -1 {
			t.Fatalf("frame %d facing changed", f)
		}
	}
}

func TestMoveJumpCollidesWithOpponent(t *testing.T) {
	// Fighters never pass over each other (design D4).
	l := run(t, "jump_forward", "", startX(40, 50), frames(25))
	if x := p(l, 19, 0).X; x != 46 {
		t.Errorf("x=%d, want stopped at 46 (min separation 4)", x)
	}
}
