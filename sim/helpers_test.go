package sim

import (
	"fmt"
	"strings"
	"testing"
)

// run simulates a (p1) vs b (p2) with the default config, optionally modified.
func run(t *testing.T, a, b string, mods ...func(*Config)) *Log {
	t.Helper()
	// Mechanic tests are written against 100 HP and a 600-frame round, so
	// they don't depend on the default round settings; the defaults
	// themselves are covered by the config, log and golden-log tests.
	cfg := DefaultConfig()
	cfg.MaxHP, cfg.FrameBudget = 100, 600
	for _, m := range mods {
		m(&cfg)
	}
	l, err := Simulate(cfg, Entrant{Name: "a", Source: a}, Entrant{Name: "b", Source: b})
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}
	return l
}

func frames(n int) func(*Config) { return func(c *Config) { c.FrameBudget = n } }

// startX places the fighters.
func startX(x1, x2 int) func(*Config) { return func(c *Config) { c.StartX = [2]int{x1, x2} } }

// p returns fighter i's (0 = p1) state on frame f.
func p(l *Log, f, i int) FighterFrame {
	if i == 0 {
		return l.Frames[f].P1
	}
	return l.Frames[f].P2
}

// events filters events by type and actor ("" matches any).
func events(l *Log, typ, actor string) []Event {
	var out []Event
	for _, e := range l.Events {
		if (typ == "" || e.Type == typ) && (actor == "" || e.Actor == actor) {
			out = append(out, e)
		}
	}
	return out
}

// animRow renders fighter i's anim per frame over [from, to] as "anim/phase" tokens.
func animRow(l *Log, i, from, to int) string {
	var parts []string
	for f := from; f <= to && f < len(l.Frames); f++ {
		ff := p(l, f, i)
		s := ff.Anim
		if ff.Phase != "" {
			s += "/" + ff.Phase[:1]
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

// firstFrame returns the first frame >= from where cond holds for fighter i, or -1.
func firstFrame(l *Log, i, from int, cond func(FighterFrame) bool) int {
	for f := from; f < len(l.Frames); f++ {
		if cond(p(l, f, i)) {
			return f
		}
	}
	return -1
}

func anim(name string) func(FighterFrame) bool {
	return func(ff FighterFrame) bool { return ff.Anim == name }
}

func evStr(e Event) string {
	return fmt.Sprintf("f%d %s %s %s", e.Frame, e.Actor, e.Type, strings.TrimSpace(string(encode(e.Detail, false))))
}
