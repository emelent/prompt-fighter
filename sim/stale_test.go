package sim

import "testing"

func TestStaleSixthPunchWhiffs(t *testing.T) {
	// No pushback, so p2 stays at distance 7: within punch range 8, but beyond
	// the 6th punch's reduced range of 8×85/100 = 6. Short hitstun keeps the
	// first five from comboing into a knockdown.
	l := run(t, "repeat 6 { punch }", "", startX(30, 37), frames(120), func(c *Config) { c.HitPushback, c.Hitstun = 0, 1 })
	st := events(l, "stale", "p1")
	if len(st) != 1 || st[0].Detail.Attack != "punch" || st[0].Detail.Repeats != 6 || *st[0].Detail.Range != 6 {
		t.Fatalf("stale events = %v, want one for the 6th punch with range 6", st)
	}
	if n := len(events(l, "hit", "p1")); n != 5 {
		t.Errorf("hits = %d, want 5", n)
	}
	w := events(l, "whiff", "p1")
	if len(w) != 1 || w[0].Detail.Reason != "stale" {
		t.Fatalf("whiffs = %v, want the 6th punch to whiff with reason stale", w)
	}
}

func TestStaleRangeSequence(t *testing.T) {
	l := run(t, "repeat 9 { punch }", "", startX(10, 60), frames(200))
	var got []int
	for _, e := range events(l, "stale", "p1") {
		got = append(got, *e.Detail.Range)
	}
	want := []int{6, 5, 4, 3} // 85, 70, 55 and 40% of 8
	if len(got) != len(want) {
		t.Fatalf("stale ranges %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stale ranges %v, want %v", got, want)
		}
	}
	// Staleness changes reach only: every punch still takes 10 frames.
	ws := events(l, "whiff", "p1")
	for i := 1; i < len(ws); i++ {
		if d := ws[i].Frame - ws[i-1].Frame; d != 10 {
			t.Fatalf("punch %d took %d frames, want 10", i+1, d)
		}
	}
}

func TestStaleMixingResets(t *testing.T) {
	l := run(t, "repeat 5 { punch } kick punch punch", "", startX(10, 60), frames(200))
	if st := events(l, "stale", ""); len(st) != 0 {
		t.Fatalf("stale after mixing attacks: %v", st)
	}
}

func TestStaleOverlaysCountSeparately(t *testing.T) {
	// The jump_kick breaks the run of kicks, so the kick after landing is fresh.
	l := run(t, "repeat 5 { kick } jump kick kick", "", startX(10, 60), frames(300))
	if st := events(l, "stale", ""); len(st) != 0 {
		t.Fatalf("stale events %v, want none", st)
	}
	if n := len(events(l, "whiff", "p1")); n != 7 {
		t.Errorf("whiffs = %d, want 7 attacks", n)
	}
}
