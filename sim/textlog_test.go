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
