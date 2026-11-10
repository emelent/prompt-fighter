package sim

import "testing"

func whiffs(l *Log, actor string) []string {
	var out []string
	for _, e := range events(l, "whiff", actor) {
		out = append(out, evStr(e))
	}
	return out
}

func TestWhiffReasons(t *testing.T) {
	longStun := func(c *Config) { c.Hitstun, c.HitPushback = 20, 0 }
	tests := []struct {
		name, a, b string
		x2         int
		mods       []func(*Config)
		want       string
	}{
		{"out of range", "kick", "", 70, nil, `f8 p1 whiff {"attack":"kick","reason":"out_of_range"}`},
		{"high attack vs ducking", "kick", "duck hold 30", 40, nil, `f8 p1 whiff {"attack":"kick","reason":"ducked"}`},
		{"punch vs ducking", "punch", "duck hold 30", 36, nil, `f4 p1 whiff {"attack":"punch","reason":"ducked"}`},
		{"sweep vs airborne", "sweep", "jump", 40, nil, `f10 p1 whiff {"attack":"sweep","reason":"airborne"}`},
		{"throw vs airborne", "throw", "jump", 34, nil, `f5 p1 whiff {"attack":"throw","reason":"airborne"}`},
		{"throw vs hitstun", "punch throw", "", 35, []func(*Config){longStun}, `f15 p1 whiff {"attack":"throw","reason":"stunned"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mods := append([]func(*Config){startX(30, tt.x2), frames(60)}, tt.mods...)
			l := run(t, tt.a, tt.b, mods...)
			got := whiffs(l, "p1")
			if len(got) == 0 || got[len(got)-1] != tt.want {
				t.Errorf("whiffs = %v\nwant last = %s", got, tt.want)
			}
			if tt.name != "throw vs hitstun" && len(events(l, "hit", "p1")) != 0 {
				t.Errorf("unexpected hit: %v", events(l, "hit", "p1"))
			}
		})
	}
}

func TestWhiffSweepHitsDuckingTarget(t *testing.T) {
	l := run(t, "sweep", "duck hold 30", startX(30, 40), frames(30))
	if h := events(l, "hit", "p1"); len(h) != 1 {
		t.Errorf("hits = %v, low attack should hit a ducking target", h)
	}
}

func TestWhiffConnectsOnce(t *testing.T) {
	// kick has 3 active frames; it must connect at most once and not whiff afterwards.
	l := run(t, "kick", "", startX(30, 40), frames(30), func(c *Config) { c.HitPushback = 0; c.Hitstun = 1 })
	if h := events(l, "hit", "p1"); len(h) != 1 || h[0].Frame != 6 {
		t.Errorf("hits = %v, want exactly one at frame 6", h)
	}
	if w := whiffs(l, "p1"); len(w) != 0 {
		t.Errorf("a connected attack must not whiff: %v", w)
	}
}
