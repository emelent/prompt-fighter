package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSimTextUnchanged pins the pf sim text output (with and without -v) for
// the golden pairs to snapshots taken before the formatter moved into sim.
func TestSimTextUnchanged(t *testing.T) {
	snaps, _ := filepath.Glob(filepath.Join("testdata", "text", "*.txt"))
	if len(snaps) < 12 {
		t.Fatalf("expected 12 snapshots, found %d", len(snaps))
	}
	for _, snap := range snaps {
		base := strings.TrimSuffix(filepath.Base(snap), ".txt")
		verbose := strings.HasSuffix(base, ".v")
		pair := strings.TrimSuffix(base, ".v")
		a, b, _ := strings.Cut(pair, "_vs_")
		args := []string{"sim", filepath.Join("..", "..", "scripts", a+".pf"), filepath.Join("..", "..", "scripts", b+".pf")}
		if verbose {
			args = append(args, "-v")
		}
		code, out, _ := call(args...)
		want, _ := os.ReadFile(snap)
		if code != 0 || out != string(want) {
			t.Errorf("%s: pf sim text output changed", base)
		}
	}
}
