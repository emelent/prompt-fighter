package sim

import (
	"encoding/json"
	"testing"
)

// checkValue returns the value of fighter actor's first read of sensor.
func checkValue(t *testing.T, l *Log, actor, sensor string) int {
	t.Helper()
	for _, e := range events(l, "check", actor) {
		if e.Detail.Sensor == sensor {
			var v int
			if err := json.Unmarshal(e.Detail.Value, &v); err != nil {
				t.Fatalf("check value %s: %v", e.Detail.Value, err)
			}
			return v
		}
	}
	t.Fatalf("no %s check by %s", sensor, actor)
	return 0
}

func TestPassiveBoostedKick(t *testing.T) {
	// p2 idles from frame 0, so its passive counter is h when a hit lands on frame h.
	// p1's kick starts on frame 144 and becomes active on frame 150.
	l := run(t, "wait 100 wait 44 kick", "wait 100 wait 100", startX(30, 40), frames(200))
	h := events(l, "hit", "p1")
	if len(h) != 1 || h[0].Frame != 150 {
		t.Fatalf("hits = %v, want the kick on frame 150", h)
	}
	if d := h[0].Detail; d.Boost != 120 || *d.Damage != 10 {
		t.Fatalf("hit %s, want boost 120 and damage 9×100×120/10000 = 10", evStr(h[0]))
	}
	// The boost is consumed: p2's counter restarts after the hit.
	// The kick interrupts p2's second wait; p2 reads passive right after its
	// hitstun, which does not count, so it reads 0.
	l2 := run(t, "wait 100 wait 44 kick", "wait 100 wait 60\nif passive > 0 { wait 1 }", startX(30, 40), frames(220))
	if v := checkValue(t, l2, "p2", "passive"); v != 0 {
		t.Errorf("passive after the boosted hit = %d, want 0", v)
	}
}

func TestPassiveBoostedThrow(t *testing.T) {
	// The throw becomes active on frame 180, against a counter of 180: ×1.3.
	l := run(t, "wait 100 wait 75 throw", "wait 100 wait 100", startX(30, 35), frames(220))
	th := events(l, "throw", "p1")
	if len(th) != 1 || th[0].Detail.Boost != 130 || *th[0].Detail.Damage != 15 {
		t.Fatalf("throws = %v, want boost 130 and damage 12×130/100 = 15", th)
	}
}

func TestPassiveNoBoostBelowLimit(t *testing.T) {
	l := run(t, "wait 100 wait 13 kick", "wait 100 wait 100", startX(30, 40), frames(200))
	h := events(l, "hit", "p1")
	if len(h) != 1 || h[0].Frame != 119 || h[0].Detail.Boost != 0 || *h[0].Detail.Damage != 9 {
		t.Fatalf("hits = %v, want an unboosted kick on frame 119", h)
	}
}

func TestPassiveMovingAwayCountsDouble(t *testing.T) {
	l := run(t, "move_back x60\nif passive > 0 { wait 1 }", "", startX(70, 80), frames(80))
	if v := checkValue(t, l, "p1", "passive"); v != 120 {
		t.Errorf("passive after 60 frames of move_back = %d, want 120", v)
	}
	if v := checkValue(t, run(t, "wait 60\nif passive > 0 { wait 1 }", "", frames(80)), "p1", "passive"); v != 60 {
		t.Errorf("passive after wait 60 = %d, want 60", v)
	}
}

func TestPassiveAttackResets(t *testing.T) {
	// p2 idles 200 frames, then whiffs a punch: its counter restarts, so p1's
	// kick on frame 216 is not boosted.
	l := run(t, "wait 100 wait 100 wait 10 kick", "wait 100 wait 100 punch wait 50", startX(30, 40), frames(260))
	h := events(l, "hit", "p1")
	if len(h) != 1 || h[0].Detail.Boost != 0 {
		t.Fatalf("hits = %v, want one unboosted kick", h)
	}
}

func TestPassiveStunDoesNotCount(t *testing.T) {
	// p1 is punched on frame 3 and stunned through frame 16; its counter only
	// holds the 3 actionable frames before the hit.
	l := run(t, "wait 40\nif passive > 0 { wait 1 }", "punch", startX(30, 36), frames(60))
	if v := checkValue(t, l, "p1", "passive"); v != 3 {
		t.Errorf("passive after a hitstun = %d, want 3", v)
	}
}

func TestPassiveIdleMirrorDraws(t *testing.T) {
	l := run(t, "wait 100", "wait 100", frames(600))
	if l.Result.Winner != nil || l.Result.HP != [2]int{100, 100} {
		t.Fatalf("idle mirror: %+v", l.Result)
	}
}

func TestAntiPassivitySensors(t *testing.T) {
	// guard: p2 has guarded 2 punches when it reads its streak.
	l := run(t, "repeat 2 { punch }", "high_block hold 25\nif guard > 0 { wait 1 }", startX(30, 36), frames(60), noBlockPush)
	if v := checkValue(t, l, "p2", "guard"); v != 2 {
		t.Errorf("guard = %d, want 2", v)
	}
	// passive: 100 idle frames read as 100; the enemy's reads the same way.
	l = run(t, "wait 100\nif passive > 0 { wait 1 }\nif enemy.passive > 0 { wait 1 }", "wait 100 wait 100", frames(120))
	if v := checkValue(t, l, "p1", "passive"); v != 100 {
		t.Errorf("passive = %d, want 100", v)
	}
	if v := checkValue(t, l, "p1", "enemy.passive"); v != 103 {
		t.Errorf("enemy.passive = %d, want 103 (read after the 2-frame read and the wait 1)", v)
	}
	// repeats: three punches in a row.
	l = run(t, "repeat 3 { punch }\nif repeats > 0 { wait 1 }", "", startX(10, 60), frames(60))
	if v := checkValue(t, l, "p1", "repeats"); v != 3 {
		t.Errorf("repeats = %d, want 3", v)
	}
	// passive is capped at 999 for scripts.
	l = run(t, "repeat 11 { wait 100 }\nif passive > 0 { wait 1 }", "", frames(1200))
	if v := checkValue(t, l, "p1", "passive"); v != 999 {
		t.Errorf("passive = %d, want the cap 999", v)
	}
}
