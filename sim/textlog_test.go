package sim

import (
	"strings"
	"testing"
)

func TestTextLogLines(t *testing.T) {
	l := run(t, "punch", "if distance < 10 { wait 1 }", startX(30, 36), frames(20))
	lines := l.EventLines(false)
	for _, ln := range lines {
		if strings.Contains(ln, " checks ") {
			t.Errorf("check line without includeChecks: %q", ln)
		}
	}
	if want := "[f 003] p1 punch hits p2 for 5 (combo 1, hp 95)"; len(lines) == 0 || lines[0] != want {
		t.Errorf("lines %q, want first %q", lines, want)
	}
	withChecks := l.EventLines(true)
	if len(withChecks) != len(lines)+1 || !strings.Contains(strings.Join(withChecks, "\n"), "p2 checks distance = 6 (line 1)") {
		t.Errorf("with checks %q", withChecks)
	}
	if got := l.ResultLine(); got != "result: p1 wins by timeout at f19 (hp 100-95)" {
		t.Errorf("result line %q", got)
	}
	d := run(t, "", "", frames(5))
	if got := d.ResultLine(); got != "result: draw by timeout at f4 (hp 100-100)" {
		t.Errorf("draw line %q", got)
	}
}

func TestTextLogAntiPassivityLines(t *testing.T) {
	has := func(l *Log, want string) {
		t.Helper()
		if !strings.Contains(strings.Join(l.EventLines(false), "\n"), want) {
			t.Errorf("no line %q in:\n%s", want, strings.Join(l.EventLines(false), "\n"))
		}
	}
	has(run(t, "repeat 4 { punch }", "high_block hold 100", startX(30, 36), frames(60), noBlockPush), "p1 punch breaks p2's guard")
	has(run(t, "repeat 6 { punch }", "", startX(10, 60), frames(80)), "p1 punch is stale (x6, range 6)")
	has(run(t, "wait 100 wait 44 kick", "wait 100 wait 100", startX(30, 40), frames(200)), "[f 150] p1 kick hits p2 for 10 (combo 1, hp 90) boosted x1.2")
	for pct, want := range map[int]string{100: "", 110: " boosted x1.1", 150: " boosted x1.5", 125: " boosted x1.25"} {
		if got := boostSuffix(pct); got != want {
			t.Errorf("boostSuffix(%d) = %q, want %q", pct, got, want)
		}
	}
}
