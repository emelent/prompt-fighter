package sim

import (
	"testing"

	"promptfighter/sim/dsl"
)

// comboConfig makes punch combos possible: long hitstun, no pushback, and
// no stale reach (these tests repeat one attack on purpose).
func comboConfig(c *Config) {
	c.Hitstun, c.HitPushback = 20, 0
	c.Attacks[dsl.AtkPunch].Damage = 10
	c.StaleFrom = 99
}

func TestComboGrowthSequence(t *testing.T) {
	l := run(t, "repeat 5 { punch }", "", startX(30, 36), frames(80), comboConfig, func(c *Config) { c.ComboKnockdownHits = 99 })
	hits := events(l, "hit", "p1")
	if len(hits) != 5 {
		t.Fatalf("hits = %d, want 5", len(hits))
	}
	wantDmg := []int{10, 11, 13, 14, 16}
	for i, h := range hits {
		if *h.Detail.Damage != wantDmg[i] || h.Detail.Combo != i+1 {
			t.Errorf("hit %d: damage %d combo %d, want %d/%d", i+1, *h.Detail.Damage, h.Detail.Combo, wantDmg[i], i+1)
		}
	}
	if hp := *hits[4].Detail.HP; hp != 100-64 {
		t.Errorf("final hp %d", hp)
	}
}

func TestDamagePercents(t *testing.T) {
	c := DefaultConfig()
	check := func(name string, f func(int) int, in, want []int) {
		t.Helper()
		for i, x := range in {
			if got := f(x); got != want[i] {
				t.Errorf("%s(%d) = %d, want %d", name, x, got, want[i])
			}
		}
	}
	check("comboPct", c.comboPct, []int{1, 2, 3, 4, 5}, []int{100, 115, 130, 145, 160})
	check("boostPct", c.boostPct, []int{0, 119, 120, 149, 150, 239, 240, 5000}, []int{100, 100, 110, 110, 120, 140, 150, 150})
	check("staleRangePct", c.staleRangePct, []int{1, 5, 6, 7, 8, 9, 20}, []int{100, 100, 85, 70, 55, 40, 40})
}

func TestComboKnockdownCap(t *testing.T) {
	l := run(t, "repeat 6 { punch }", "", startX(30, 36), frames(80), comboConfig)
	hits := events(l, "hit", "p1")
	if len(hits) < 5 || hits[4].Detail.Combo != 5 {
		t.Fatalf("hits = %v", hits)
	}
	kd := events(l, "knockdown", "p2")
	if len(kd) != 1 || kd[0].Frame != hits[4].Frame || kd[0].Detail.Reason != "combo" {
		t.Fatalf("knockdown %v, want on the 5th hit", kd)
	}
	if f := hits[4].Frame; p(l, f, 1).State != "knockdown" || p(l, f, 1).Combo != 0 {
		t.Errorf("after 5th hit: %+v, want knockdown with combo reset", p(l, f, 1))
	}
	if len(hits) != 5 {
		t.Errorf("6th punch hit a knocked-down target")
	}
}

func TestComboResetsWhenActionable(t *testing.T) {
	// With a short hitstun the defender is actionable between punches, so every
	// hit starts a new combo.
	l := run(t, "repeat 3 { punch }", "", startX(30, 36), frames(40), func(c *Config) { c.Hitstun, c.HitPushback = 5, 0 })
	for _, h := range events(l, "hit", "p1") {
		if h.Detail.Combo != 1 {
			t.Errorf("hit %s combo %d, want 1", evStr(h), h.Detail.Combo)
		}
	}
}

func TestKnockdownInvulnerable(t *testing.T) {
	// throw (24 frames) knocks down at frame 5; with a 40-frame knockdown
	// (through frame 45) the following punch is active 27-28 on a downed target.
	l := run(t, "throw punch", "", startX(30, 34), frames(50), func(c *Config) { c.KnockdownFrames = 40 })
	w := whiffs(l, "p1")
	if len(w) != 1 || w[0] != `f28 p1 whiff {"attack":"punch","reason":"invulnerable"}` {
		t.Errorf("whiffs = %v", w)
	}
	for f := 5; f <= 45; f++ {
		if ff := p(l, f, 1); ff.State != "knockdown" || !ff.Invulnerable {
			t.Errorf("frame %d: %+v", f, ff)
		}
	}
}

func TestKnockdownWakeUpRunsOnHit(t *testing.T) {
	l := run(t, "throw", "on_hit { move_back x2 }", startX(30, 34), frames(40))
	if k := events(l, "knockdown", "p2"); len(k) != 1 || k[0].Frame != 5 {
		t.Fatalf("knockdown = %v", k)
	}
	if got := handlerEnters(l, "p2"); len(got) != 1 || got[0] != 26 {
		t.Fatalf("handler_enter = %v, want [26]", got)
	}
	if ff := p(l, 26, 1); ff.Invulnerable || ff.Anim != "walk_back" {
		t.Errorf("wake-up frame: %+v", ff)
	}
}

func TestResultKOEndsEarly(t *testing.T) {
	l := run(t, "punch", "", startX(30, 36), func(c *Config) { c.MaxHP = 5 })
	r := l.Result
	if r.Reason != "ko" || r.Winner == nil || *r.Winner != "p1" || r.EndFrame != 3 || r.HP != [2]int{5, 0} {
		t.Errorf("result %+v", r)
	}
	if len(l.Frames) != 4 {
		t.Errorf("frames %d, want 4 (no frames after the KO)", len(l.Frames))
	}
	if ko := events(l, "ko", ""); len(ko) != 1 || ko[0].Actor != "p2" || ko[0].Frame != 3 {
		t.Errorf("ko events %v", ko)
	}
	if s := p(l, 3, 1).State; s != "ko" {
		t.Errorf("state %q", s)
	}
}

func TestResultHPNeverNegative(t *testing.T) {
	l := run(t, "kick", "", startX(30, 40), func(c *Config) { c.MaxHP = 3 })
	if hp := l.Result.HP[1]; hp != 0 {
		t.Errorf("hp %d, want 0", hp)
	}
}

func TestResultTimeoutDraw(t *testing.T) {
	l := run(t, "", "")
	r := l.Result
	if r.Reason != "timeout" || r.Winner != nil || r.EndFrame != 599 || len(l.Frames) != 600 {
		t.Errorf("result %+v frames %d", r, len(l.Frames))
	}
}

func TestResultTimeoutHigherHPWins(t *testing.T) {
	l := run(t, "punch", "", startX(30, 36))
	if r := l.Result; r.Reason != "timeout" || r.Winner == nil || *r.Winner != "p1" || r.HP != [2]int{100, 95} {
		t.Errorf("result %+v", r)
	}
}

func TestResultDoubleKO(t *testing.T) {
	l := run(t, "punch", "punch", startX(30, 37), func(c *Config) { c.MaxHP = 5 })
	if r := l.Result; r.Reason != "double_ko" || r.Winner != nil || r.EndFrame != 3 {
		t.Errorf("result %+v", r)
	}
	if ko := events(l, "ko", ""); len(ko) != 2 {
		t.Errorf("ko events %v, want 2", ko)
	}
}
