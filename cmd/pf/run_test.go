package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"promptfighter/sim"
)

func call(args ...string) (code int, out, errOut string) {
	var o, e bytes.Buffer
	code = run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestValidate(t *testing.T) {
	good, bad := filepath.Join("testdata", "good.pf"), filepath.Join("testdata", "bad.pf")
	tests := []struct {
		name     string
		args     []string
		code     int
		out, err string
	}{
		{"valid", []string{"validate", good}, 0, good + ": ok\n", ""},
		{"invalid", []string{"validate", bad}, 1, bad + `:3:5: unknown action "kik" — did you mean "kick"?` + "\n", ""},
		{"mixed", []string{"validate", good, bad}, 1, good + ": ok\n" + bad + `:3:5: unknown action "kik" — did you mean "kick"?` + "\n", ""},
		{"missing", []string{"validate", "testdata/nope.pf"}, 2, "", "nope.pf"},
		{"usage", []string{"validate"}, 2, "", "usage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errOut := call(tt.args...)
			if code != tt.code || out != tt.out || !strings.Contains(errOut, tt.err) {
				t.Errorf("code %d out %q err %q; want %d %q containing %q", code, out, errOut, tt.code, tt.out, tt.err)
			}
		})
	}
}

func writeScripts(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestSimText(t *testing.T) {
	dir := writeScripts(t, map[string]string{"a.pf": "loop { move_forward x2 kick }", "b.pf": "loop { if distance < 13 { high_block hold 8 } }"})
	code, out, _ := call("sim", filepath.Join(dir, "a.pf"), filepath.Join(dir, "b.pf"))
	if code != 0 {
		t.Fatalf("code %d", code)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if !strings.Contains(out, "p1 kick blocked by p2") {
		t.Errorf("missing block line:\n%s", out)
	}
	for _, ln := range lines[:len(lines)-1] {
		if !strings.HasPrefix(ln, "[f ") || len(ln) < 8 || ln[6] != ']' {
			t.Errorf("bad event line %q", ln)
		}
		if strings.Contains(ln, " checks ") {
			t.Errorf("check event without -v: %q", ln)
		}
	}
	last := lines[len(lines)-1]
	if !strings.HasPrefix(last, "result: ") || !strings.Contains(last, " at f") || !strings.Contains(last, "(hp ") {
		t.Errorf("bad result line %q", last)
	}
	_, vout, _ := call("sim", "-v", filepath.Join(dir, "a.pf"), filepath.Join(dir, "b.pf"))
	if !strings.Contains(vout, "p2 checks distance") {
		t.Errorf("-v should include checks")
	}
}

func TestSimJSONMatchesSimulate(t *testing.T) {
	a, b := "loop { kick }", "loop { punch }"
	dir := writeScripts(t, map[string]string{"a.pf": a, "b.pf": b})
	_, out1, _ := call("sim", filepath.Join(dir, "a.pf"), filepath.Join(dir, "b.pf"), "--json")
	_, out2, _ := call("sim", "--json", filepath.Join(dir, "a.pf"), filepath.Join(dir, "b.pf"))
	l, err := sim.Simulate(sim.DefaultConfig(), sim.Entrant{Name: "a.pf", Source: a}, sim.Entrant{Name: "b.pf", Source: b})
	if err != nil {
		t.Fatal(err)
	}
	if out1 != string(l.JSON()) || out1 != out2 {
		t.Errorf("JSON output differs from sim.Simulate")
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(out1), &doc); err != nil {
		t.Errorf("output is not JSON: %v", err)
	}
}

func TestSimFrames(t *testing.T) {
	dir := writeScripts(t, map[string]string{"a.pf": "wait 1", "b.pf": "wait 1"})
	code, out, _ := call("sim", filepath.Join(dir, "a.pf"), filepath.Join(dir, "b.pf"), "--frames", "120", "--json")
	var l sim.Log
	if code != 0 || json.Unmarshal([]byte(out), &l) != nil || len(l.Frames) != 120 || l.FrameBudget != 120 {
		t.Errorf("code %d frames %d budget %d", code, len(l.Frames), l.FrameBudget)
	}
	if code, _, _ := call("sim", filepath.Join(dir, "a.pf"), filepath.Join(dir, "b.pf"), "--frames", "0"); code != 2 {
		t.Errorf("--frames 0 should be a usage error, got %d", code)
	}
}

func TestSimInvalidScript(t *testing.T) {
	dir := writeScripts(t, map[string]string{"a.pf": "kick", "bad.pf": "kik"})
	bad := filepath.Join(dir, "bad.pf")
	code, out, _ := call("sim", filepath.Join(dir, "a.pf"), bad)
	if code != 1 || out != bad+`:1:1: unknown action "kik" — did you mean "kick"?`+"\n" {
		t.Errorf("code %d out %q", code, out)
	}
}

func TestSelfplay(t *testing.T) {
	dir := writeScripts(t, map[string]string{
		"c_idle.pf":    "",
		"a_kicker.pf":  "loop { move_forward x3 kick }",
		"b_blocker.pf": "loop { high_block hold 20 }",
		"d_bad.pf":     "kik",
	})
	code, out, _ := call("selfplay", dir)
	if code != 0 {
		t.Fatalf("code %d\n%s", code, out)
	}
	if !strings.Contains(out, "excluded") || !strings.Contains(out, `d_bad.pf:1:1: unknown action "kik"`) {
		t.Errorf("invalid script not reported:\n%s", out)
	}
	names, m, err := selfplay(dir, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(names, ",") != "a_kicker,b_blocker,c_idle" {
		t.Errorf("names %v, want sorted and without the invalid script", names)
	}
	for i := range m {
		for j := range m {
			if i == j {
				continue
			}
			r, s := m[i][j], m[j][i]
			if r.w+r.d+r.l != 2 || r.w != s.l || r.l != s.w || r.d != s.d {
				t.Errorf("cell (%d,%d)=%v vs (%d,%d)=%v", i, j, r, j, i, s)
			}
		}
	}
	matrix := out[strings.LastIndex(out[:strings.Index(out, "total")], "\n")+1:]
	rows := strings.Split(strings.TrimRight(matrix, "\n"), "\n")
	if len(rows) != 4 || !strings.HasSuffix(rows[0], "total") || strings.Count(matrix, "—") != 3 || !strings.HasPrefix(rows[1], "a_kicker ") {
		t.Errorf("matrix shape:\n%s", out)
	}
	_, out2, _ := call("selfplay", dir)
	if out != out2 {
		t.Errorf("selfplay output not deterministic")
	}
}

func TestVersion(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		code, out, _ := call(arg)
		if code != 0 || out != "dev\n" {
			t.Errorf("pf %s: %d %q, want 0 \"dev\"", arg, code, out)
		}
	}
}
