package sim

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden event logs")

var goldenPairs = [][2]string{
	{"rushdown", "turtle"},  // timeout
	{"turtle", "jumper"},    // ko (anti-air pokes)
	{"jumper", "lazy"},      // ko
	{"thrower", "rushdown"}, // ko
	{"lazy", "thrower"},     // ko
	{"ducker", "jumper"},    // ko (landing lows vs a ducker)
}

func TestGoldenLogs(t *testing.T) {
	reasons := map[string]bool{}
	for _, pr := range goldenPairs {
		name := pr[0] + "_vs_" + pr[1]
		t.Run(name, func(t *testing.T) {
			l, err := Simulate(DefaultConfig(), sample(t, pr[0]+".pf"), sample(t, pr[1]+".pf"))
			if err != nil {
				t.Fatal(err)
			}
			reasons[l.Result.Reason] = true
			path := filepath.Join("testdata", "golden", name+".json")
			got := l.IndentJSON()
			if *update {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run `go test ./sim -run Golden -update`)", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("log differs from %s; if the change is intended, rerun with -update and review the diff", path)
			}
		})
	}
	if !reasons["ko"] || !reasons["timeout"] {
		t.Errorf("golden set must include a KO and a timeout, got %v", reasons)
	}
}

func TestDeterminism(t *testing.T) {
	ss := samples(t)
	for i, a := range ss {
		b := ss[(i+1)%len(ss)]
		l1, _ := Simulate(DefaultConfig(), a, b)
		l2, _ := Simulate(DefaultConfig(), a, b)
		if !bytes.Equal(l1.JSON(), l2.JSON()) {
			t.Errorf("%s vs %s: two in-process runs differ", a.Name, b.Name)
		}
	}
}

func TestDeterminismAcrossProcesses(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the pf binary")
	}
	bin := filepath.Join(t.TempDir(), "pf")
	if out, err := exec.Command("go", "build", "-o", bin, "../cmd/pf").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	a, b := filepath.Join("..", "scripts", "rushdown.pf"), filepath.Join("..", "scripts", "turtle.pf")
	out, err := exec.Command(bin, "sim", a, b, "--json").Output()
	if err != nil {
		t.Fatal(err)
	}
	l, _ := Simulate(DefaultConfig(), sample(t, "rushdown.pf"), sample(t, "turtle.pf"))
	if !bytes.Equal(out, l.JSON()) {
		t.Errorf("subprocess output differs from in-process Simulate")
	}
}

// mirror returns f as seen from the other side of the stage.
func mirror(f FighterFrame, width int) FighterFrame {
	f.X = width - f.X
	f.Facing = -f.Facing
	return f
}

func TestSymmetryMirrorMatch(t *testing.T) {
	w := DefaultConfig().StageWidth
	for _, s := range samples(t) {
		l, _ := Simulate(DefaultConfig(), s, s)
		if l.Result.Winner != nil {
			t.Errorf("%s vs itself: winner %s, want draw", s.Name, *l.Result.Winner)
		}
		for _, f := range l.Frames {
			if f.P1 != mirror(f.P2, w) {
				t.Fatalf("%s vs itself: frame %d not mirrored: %+v vs %+v", s.Name, f.F, f.P1, f.P2)
			}
		}
	}
}

func TestSymmetrySwappedSides(t *testing.T) {
	w := DefaultConfig().StageWidth
	ss := samples(t)
	for i, a := range ss {
		for j, b := range ss {
			if i == j {
				continue
			}
			l1, _ := Simulate(DefaultConfig(), a, b)
			l2, _ := Simulate(DefaultConfig(), b, a)
			name := strings.TrimSuffix(a.Name, ".pf") + "/" + strings.TrimSuffix(b.Name, ".pf")
			if len(l1.Frames) != len(l2.Frames) {
				t.Errorf("%s: frame counts %d vs %d", name, len(l1.Frames), len(l2.Frames))
				continue
			}
			for k := range l1.Frames {
				if l2.Frames[k].P1 != mirror(l1.Frames[k].P2, w) || l2.Frames[k].P2 != mirror(l1.Frames[k].P1, w) {
					t.Errorf("%s: frame %d not mirrored", name, k)
					break
				}
			}
			winner := func(l *Log) string {
				if l.Result.Winner == nil {
					return "draw"
				}
				if *l.Result.Winner == "p1" {
					return l.Players[0].Name
				}
				return l.Players[1].Name
			}
			if winner(l1) != winner(l2) || l1.Result.Reason != l2.Result.Reason {
				t.Errorf("%s: winner %s/%s vs %s/%s", name, winner(l1), l1.Result.Reason, winner(l2), l2.Result.Reason)
			}
		}
	}
}

func TestConcurrentSimulate(t *testing.T) {
	ss := samples(t)
	want := make([][]byte, len(ss))
	for i, a := range ss {
		l, _ := Simulate(DefaultConfig(), a, ss[(i+1)%len(ss)])
		want[i] = l.JSON()
	}
	for i, a := range ss {
		t.Run(a.Name, func(t *testing.T) {
			t.Parallel()
			for k := 0; k < 3; k++ {
				l, _ := Simulate(DefaultConfig(), a, ss[(i+1)%len(ss)])
				if !bytes.Equal(l.JSON(), want[i]) {
					t.Errorf("concurrent run differs")
				}
			}
		})
	}
}
