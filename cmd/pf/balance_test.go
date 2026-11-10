package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

const samplesDir = "../../scripts"

func TestBalanceSamplesAreNonTransitive(t *testing.T) {
	var out bytes.Buffer
	names, m, err := selfplay(samplesDir, &out)
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("a sample script is invalid:\n%s", out.String())
	}
	for _, want := range []string{"jumper", "lazy", "rushdown", "thrower", "turtle"} {
		found := false
		for _, n := range names {
			found = found || n == want
		}
		if !found {
			t.Errorf("missing sample %s.pf", want)
		}
	}
	for i, n := range names {
		beats, losesTo := 0, 0
		for j := range names {
			if i == j {
				continue
			}
			switch r := m[i][j]; {
			case r.w > r.l:
				beats++
			case r.l > r.w:
				losesTo++
			}
		}
		if beats == 0 || losesTo == 0 {
			t.Errorf("%s beats %d and loses to %d samples; each must beat one and lose to one", n, beats, losesTo)
		}
	}
}

func TestBalanceSamplesValidate(t *testing.T) {
	paths, _ := filepath.Glob(filepath.Join(samplesDir, "*.pf"))
	code, out, _ := call(append([]string{"validate"}, paths...)...)
	if code != 0 || len(paths) < 5 {
		t.Errorf("code %d (%d files)\n%s", code, len(paths), out)
	}
}
