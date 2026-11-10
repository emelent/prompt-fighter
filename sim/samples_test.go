package sim

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// samples loads /scripts/*.pf sorted by name.
func samples(t *testing.T) []Entrant {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "scripts", "*.pf"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no sample scripts: %v", err)
	}
	sort.Strings(paths)
	var out []Entrant
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, Entrant{Name: filepath.Base(p), Source: string(src)})
	}
	return out
}

func sample(t *testing.T, name string) Entrant {
	for _, e := range samples(t) {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("no sample %s", name)
	return Entrant{}
}

func TestBalanceLazyHasNoChecks(t *testing.T) {
	lazy := sample(t, "lazy.pf")
	if strings.Contains(lazy.Source, "in_range") || strings.Contains(lazy.Source, "if ") {
		t.Errorf("lazy.pf must not read sensors")
	}
	for _, opp := range samples(t) {
		for side := 0; side < 2; side++ {
			a, b, id := lazy, opp, "p1"
			if side == 1 {
				a, b, id = opp, lazy, "p2"
			}
			l, err := Simulate(DefaultConfig(), a, b)
			if err != nil {
				t.Fatal(err)
			}
			if c := events(l, "check", id); len(c) != 0 {
				t.Errorf("lazy (%s) vs %s: %d check events", id, opp.Name, len(c))
			}
		}
	}
}
