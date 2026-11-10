package sim

import "testing"

// First throw is active on frame 5; the target is knocked down through frame
// 25 and wakes (runs on_hit) on frame 26. The immediate re-throw starts on
// frame 24 and is active on frame 29.

func TestThrowWakeUpAttackBeatsRethrow(t *testing.T) {
	l := run(t, "throw throw", "on_hit { punch }", startX(30, 34), frames(60))
	if th := events(l, "throw", "p1"); len(th) != 1 || th[0].Frame != 5 {
		t.Fatalf("throws = %v, want only the first throw", th)
	}
	h := events(l, "hit", "p2")
	if len(h) != 1 || h[0].Frame != 29 {
		t.Fatalf("p2 hits = %v, want the wake-up punch to hit on frame 29", h)
	}
	if got := handlerEnters(l, "p2"); len(got) < 1 || got[0] != 26 {
		t.Errorf("wake-up handler at %v, want 26", got)
	}
	if len(whiffs(l, "p1")) != 0 {
		t.Errorf("the beaten re-throw must be abandoned, not whiff: %v", whiffs(l, "p1"))
	}
}

func TestThrowRethrowBeatsWakeUpBlock(t *testing.T) {
	l := run(t, "throw throw", "on_hit { high_block hold 10 }", startX(30, 34), frames(60))
	th := events(l, "throw", "p1")
	if len(th) != 2 || th[1].Frame != 29 || th[1].Detail.Outcome != "success" {
		t.Fatalf("throws = %v, want the re-throw to connect on frame 29", th)
	}
	if k := events(l, "knockdown", "p2"); len(k) != 2 {
		t.Errorf("knockdowns = %v, want 2", k)
	}
}

func TestComboLinksAtDefaults(t *testing.T) {
	l := run(t, "punch punch kick", "", startX(30, 36), frames(60))
	hits := events(l, "hit", "p1")
	if len(hits) != 3 {
		t.Fatalf("hits = %v, want 3", hits)
	}
	for i, want := range []struct{ combo, dmg int }{{1, 5}, {2, 5}, {3, 11}} {
		if hits[i].Detail.Combo != want.combo || *hits[i].Detail.Damage != want.dmg {
			t.Errorf("hit %d: %s, want combo %d damage %d", i+1, evStr(hits[i]), want.combo, want.dmg)
		}
	}
	if hp := *hits[2].Detail.HP; hp != 79 {
		t.Errorf("hp %d, want 79", hp)
	}
}

func TestComboPushbackEndsPunchStrings(t *testing.T) {
	l := run(t, "punch punch punch", "", startX(30, 36), frames(60))
	if h := events(l, "hit", "p1"); len(h) != 2 {
		t.Errorf("hits = %v, want 2", h)
	}
	if w := whiffs(l, "p1"); len(w) != 1 || w[0] != `f24 p1 whiff {"attack":"punch","reason":"out_of_range"}` {
		t.Errorf("whiffs = %v", w)
	}
}
