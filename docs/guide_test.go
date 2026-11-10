package docs

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"promptfighter/sim"
	"promptfighter/sim/dsl"
)

// TestGuideInSync fails when the player guide misses engine vocabulary,
// frame data or limits.
func TestGuideInSync(t *testing.T) {
	g := PlayerGuide
	cfg := sim.DefaultConfig()
	if !strings.Contains(g, "`"+cfg.Version+"`") {
		t.Errorf("guide does not name config version %s", cfg.Version)
	}
	for a := dsl.Action(0); a < dsl.NumActions; a++ {
		if !strings.Contains(g, "`"+a.String()+"`") {
			t.Errorf("guide is missing action %q", a)
		}
	}
	for s := dsl.Sensor(0); s < dsl.NumSensors; s++ {
		if !strings.Contains(g, "`"+s.String()) {
			t.Errorf("guide is missing sensor %q", s)
		}
	}
	for i, name := range dsl.AttackNames() {
		if !strings.Contains(g, "`"+name+"`") {
			t.Errorf("guide is missing in_range attack %q", name)
		}
		spec := cfg.Attacks[i]
		row := fmt.Sprintf("| %s | %d | %d | %d |", name, spec.Startup, spec.Active, spec.Recovery)
		if !strings.Contains(g, row) {
			t.Errorf("guide frame data row missing or wrong: want %q", row)
		}
	}
	lim := cfg.Limits
	for _, want := range []string{
		fmt.Sprintf("%d bytes", lim.MaxBytes),
		fmt.Sprintf("%d statements", lim.MaxStatements),
		fmt.Sprintf("nesting depth %d", lim.MaxDepth),
		fmt.Sprintf("%d frames of hitstun", cfg.Hitstun),
		fmt.Sprintf("**%d frames**", cfg.FrameBudget),
	} {
		if !strings.Contains(g, want) {
			t.Errorf("guide is missing %q", want)
		}
	}
	for _, z := range cfg.AirZones {
		if !strings.Contains(g, fmt.Sprintf("| %d–%d |", z.From, z.To)) {
			t.Errorf("guide air-zone table missing %d–%d", z.From, z.To)
		}
	}
}

var pfBlock = regexp.MustCompile("(?s)```pf\n(.*?)```")

// TestGuideExamplesValid validates every ```pf block in the guide.
func TestGuideExamplesValid(t *testing.T) {
	blocks := pfBlock.FindAllStringSubmatch(PlayerGuide, -1)
	if len(blocks) < 4 {
		t.Fatalf("expected several pf examples, found %d", len(blocks))
	}
	for i, b := range blocks {
		if errs := dsl.Validate(b[1], sim.DefaultConfig().Limits); len(errs) > 0 {
			t.Errorf("pf example %d is invalid: %v\n%s", i+1, errs, b[1])
		}
	}
}

func TestRulesEmbedded(t *testing.T) {
	if !strings.HasPrefix(Rules, "# Prompt Fighter — Engine Rules") {
		t.Errorf("RULES.md not embedded")
	}
}
