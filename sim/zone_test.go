package sim

import "testing"

func TestZoneRisingOverheadHitsDucker(t *testing.T) {
	// kick on frame 1 = airborne frame 2 (rising); active frames 4-8.
	l := run(t, "jump_forward kick", "duck hold 30", startX(30, 40), frames(30))
	h := events(l, "hit", "p1")
	if len(h) != 1 || h[0].Detail.Attack != "jump_kick" {
		t.Fatalf("hits = %v, want the jump kick to hit the ducking target", h)
	}
	if len(whiffs(l, "p1")) != 0 {
		t.Errorf("unexpected whiff %v", whiffs(l, "p1"))
	}
	// The attacker's jump continues and lands normally.
	if s := p(l, 19, 0).State; s != "airborne" {
		t.Errorf("frame 19 state %q, want airborne", s)
	}
	if s := p(l, 20, 0).State; s != "landing" {
		t.Errorf("frame 20 state %q, want landing", s)
	}
}

func TestZoneOverheadGuardedByHighBlock(t *testing.T) {
	for name, a := range map[string]string{
		"rising":  "jump_forward kick",        // starts on airborne frame 2
		"falling": "jump wait 13 kick wait 1", // starts on airborne frame 15
	} {
		l := run(t, a, "high_block hold 40", startX(30, 36), frames(40))
		if b := events(l, "block", "p1"); len(b) != 1 {
			t.Errorf("%s: blocks = %v, hits = %v", name, b, events(l, "hit", "p1"))
		}
		l = run(t, a, "low_block hold 40", startX(30, 36), frames(40))
		if h := events(l, "hit", "p1"); len(h) != 1 {
			t.Errorf("%s: low_block must not guard an overhead, hits = %v", name, h)
		}
	}
}

func TestZoneAirToAirWhiffsGrounded(t *testing.T) {
	// kick on frame 8 = airborne frame 9 (peak).
	l := run(t, "jump wait 7 kick", "", startX(30, 36), frames(30))
	w := whiffs(l, "p1")
	if len(w) != 1 || w[0] != `f15 p1 whiff {"attack":"jump_kick","reason":"grounded"}` {
		t.Errorf("whiffs = %v", w)
	}
	if len(events(l, "hit", "p1")) != 0 {
		t.Errorf("air-to-air must not hit a grounded target")
	}
}

func TestZoneAirToAirHitsAirborne(t *testing.T) {
	l := run(t, "jump wait 7 kick", "jump", startX(30, 36), frames(40))
	h := events(l, "hit", "p1")
	if len(h) != 1 || h[0].Frame != 11 {
		t.Fatalf("hits = %v, want one at frame 11", h)
	}
	if ff := p(l, 11, 1); ff.State != "knockdown" || ff.Y != 0 {
		t.Errorf("target %+v, want knockdown at y=0", ff)
	}
}

func TestZoneLandingLowGuardedByLowBlock(t *testing.T) {
	// kick on frame 17 = airborne frame 18 (landing low); active frames 20-24,
	// all after touchdown.
	l := run(t, "jump wait 16 kick", "low_block hold 40", startX(30, 36), frames(40))
	if b := events(l, "block", "p1"); len(b) != 1 || b[0].Frame != 20 {
		t.Errorf("low_block: blocks = %v, hits = %v", b, events(l, "hit", "p1"))
	}
	l = run(t, "jump wait 16 kick", "high_block hold 40", startX(30, 36), frames(40))
	if h := events(l, "hit", "p1"); len(h) != 1 || h[0].Frame != 20 {
		t.Errorf("high_block must not guard a landing low, hits = %v", h)
	}
	// It hits a ducking target too (low attack).
	l = run(t, "jump wait 16 kick", "duck hold 40", startX(30, 36), frames(40))
	if h := events(l, "hit", "p1"); len(h) != 1 {
		t.Errorf("landing low vs ducker: hits = %v", h)
	}
}

func TestZoneFixedAtStart(t *testing.T) {
	// kick on frame 5 = airborne frame 6 (rising overhead); active frames 8-12
	// span airborne frames 9-13 (peak), but it stays an overhead and cannot
	// hit the airborne target.
	l := run(t, "jump wait 4 kick", "jump", startX(30, 36), frames(30))
	if h := events(l, "hit", "p1"); len(h) != 0 {
		t.Errorf("hits = %v, want none", h)
	}
	w := whiffs(l, "p1")
	if len(w) != 1 || w[0] != `f12 p1 whiff {"attack":"jump_kick","reason":"airborne"}` {
		t.Errorf("whiffs = %v", w)
	}
}

func TestZoneGroundedHighStillDucked(t *testing.T) {
	l := run(t, "punch", "duck hold 30", startX(30, 36), frames(20))
	if w := whiffs(l, "p1"); len(w) != 1 || w[0] != `f4 p1 whiff {"attack":"punch","reason":"ducked"}` {
		t.Errorf("whiffs = %v", w)
	}
}
