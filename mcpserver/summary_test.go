package mcpserver

import (
	"fmt"
	"strings"
	"testing"

	"promptfighter/sim"
)

func simLog(t *testing.T, a, b string, mod func(*sim.Config)) *sim.Log {
	t.Helper()
	cfg := sim.DefaultConfig()
	if mod != nil {
		mod(&cfg)
	}
	l, err := sim.Simulate(cfg, sim.Entrant{Name: "alice", Source: a}, sim.Entrant{Name: "bob", Source: b})
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestSummaryStats(t *testing.T) {
	// Hand-checked: punch punch kick from distance 6 is a 3-hit combo for
	// 5+5+11 damage (combos grow 15% per hit); p2 checks distance once per loop iteration.
	l := simLog(t, "punch punch kick", "loop { if distance > 50 { wait 1 } }", func(c *sim.Config) {
		c.StartX = [2]int{30, 36}
		c.FrameBudget = 60
	})
	s := summarise(l, false)
	p1, p2 := s.Stats.P1, s.Stats.P2
	if p1.Hits != 3 || p1.DamageDealt != 21 || p1.MaxCombo != 3 || p1.Whiffs != 0 || p1.Checks != 0 {
		t.Errorf("p1 stats %+v", p1)
	}
	if p2.Hits != 0 || p2.Checks == 0 || p2.Blocks != 0 || p2.Knockdowns != 0 {
		t.Errorf("p2 stats %+v", p2)
	}
	if s.Players[0] != (PlayerRef{"p1", "alice"}) || s.Players[1] != (PlayerRef{"p2", "bob"}) {
		t.Errorf("players %+v", s.Players)
	}
	if s.Result.Reason != "timeout" || s.Result.HP != [2]int{200, 179} || *s.Result.Winner != "p1" {
		t.Errorf("result %+v", s.Result)
	}
	if strings.Join(s.Events, "\n") != strings.Join(l.EventLines(false), "\n") || s.EventsTruncated || s.EventsTotal != len(s.Events) {
		t.Errorf("events %v", s.Events)
	}
	withChecks := summarise(l, true)
	if withChecks.EventsTotal != len(l.EventLines(true)) || withChecks.EventsTotal <= s.EventsTotal {
		t.Errorf("include_checks should add check lines")
	}
}

func TestSummaryThrowsBlocksKnockdowns(t *testing.T) {
	l := simLog(t, "throw wait 30 kick", "wait 30 high_block hold 60", func(c *sim.Config) {
		c.StartX = [2]int{30, 34}
		c.FrameBudget = 80
	})
	s := summarise(l, false)
	if s.Stats.P1.Throws != 1 || s.Stats.P1.DamageDealt != 12 || s.Stats.P2.Knockdowns != 1 {
		t.Errorf("throw stats %+v / %+v", s.Stats.P1, s.Stats.P2)
	}
	if s.Stats.P2.Blocks != 1 {
		t.Errorf("p2 should have blocked the kick: %+v\n%s", s.Stats.P2, strings.Join(s.Events, "\n"))
	}
}

func TestSummaryCap(t *testing.T) {
	l := &sim.Log{Players: [2]sim.Player{{ID: "p1", Name: "a"}, {ID: "p2", Name: "b"}}}
	for i := 0; i < 250; i++ {
		l.Events = append(l.Events, sim.Event{Frame: i, Type: "script_end", Actor: "p1"})
	}
	s := summarise(l, false)
	if len(s.Events) != 200 || !s.EventsTruncated || s.EventsTotal != 250 {
		t.Errorf("len %d truncated %v total %d", len(s.Events), s.EventsTruncated, s.EventsTotal)
	}
	if txt := s.text(func(string) string { return "" }); !strings.Contains(txt, "events (first 200 of 250):") {
		t.Errorf("text header missing:\n%s", txt[:200])
	}
}

func TestSummaryText(t *testing.T) {
	l := simLog(t, "punch", "", func(c *sim.Config) { c.StartX = [2]int{30, 36}; c.FrameBudget = 20 })
	txt := summarise(l, false).text(func(id string) string { return fmt.Sprintf("(%s-label)", id) })
	lines := strings.Split(txt, "\n")
	if lines[0] != l.ResultLine() || !strings.HasPrefix(lines[1], "p1 (p1-label): hits 1") || !strings.HasPrefix(lines[2], "p2 (p2-label): hits 0") || lines[3] != "events (3):" {
		t.Errorf("text:\n%s", txt)
	}
}

func TestBotsLoaded(t *testing.T) {
	bots, err := loadBots(sim.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, b := range bots {
		names = append(names, b.Name)
		if b.Summary == "" || strings.Contains(b.Summary, "#") || b.Source == "" {
			t.Errorf("bot %s: summary %q", b.Name, b.Summary)
		}
	}
	if got := strings.Join(names, ","); got != "ducker,jumper,lazy,rushdown,thrower,turtle" {
		t.Errorf("bots %s", got)
	}
	if got := botSummary("# a — b.\n# Strategy: c\n#\n# d\nkick # not this"); got != "a — b. Strategy: c d" {
		t.Errorf("summary %q", got)
	}
}
