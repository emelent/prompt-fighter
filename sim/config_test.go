package sim

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"promptfighter/sim/dsl"
)

func TestConfigBuildSpecFrameData(t *testing.T) {
	c := DefaultConfig()
	tests := []struct {
		atk  dsl.Attack
		want AttackSpec
	}{
		{dsl.AtkPunch, AttackSpec{Startup: 3, Active: 2, Recovery: 5, Height: HeightHigh, Damage: 5, Range: 8}},
		{dsl.AtkKick, AttackSpec{Startup: 6, Active: 3, Recovery: 9, Height: HeightHigh, Damage: 9, Range: 12}},
		{dsl.AtkSweep, AttackSpec{Startup: 8, Active: 3, Recovery: 14, Height: HeightLow, Damage: 8, Range: 12, Knockdown: true}},
		{dsl.AtkThrow, AttackSpec{Startup: 5, Active: 1, Recovery: 15, Height: HeightNone, Damage: 12, Range: 5, Knockdown: true}},
	}
	for _, tt := range tests {
		t.Run(tt.atk.String(), func(t *testing.T) {
			if got := c.Attacks[tt.atk]; got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
	if c.MaxHP != 100 || c.FrameBudget != 600 || c.StageWidth != 100 || c.StartX != [2]int{30, 70} {
		t.Errorf("stage/round defaults wrong: %+v", c)
	}
	if c.JumpRise != 10 || c.JumpFall != 10 || c.JumpLanding != 2 || c.JumpSpeedX != 1 {
		t.Errorf("jump defaults wrong")
	}
	if c.DuckStartup != 2 || c.DuckRecovery != 2 || c.BlockStartup != 1 || c.BlockRecovery != 1 {
		t.Errorf("duck/block defaults wrong")
	}
	if c.Hitstun != 5 || c.HitPushback != 3 || c.Blockstun != 3 || c.BlockPushback != 2 || c.KnockdownFrames != 20 {
		t.Errorf("stun defaults wrong")
	}
	if c.ComboScalePct != 80 || c.ComboFloorPct != 30 || c.ComboKnockdownHits != 5 || c.SensorCost != 2 {
		t.Errorf("combo/sensor defaults wrong")
	}
	if l := c.Limits; l.MaxBytes != 4096 || l.MaxStatements != 40 || l.MaxDepth != 4 {
		t.Errorf("limits wrong: %+v", l)
	}
}

func TestConfigOverlayDefaults(t *testing.T) {
	c := DefaultConfig()
	tests := []struct {
		atk  dsl.Attack
		want AttackSpec
	}{
		{dsl.AtkJumpKick, AttackSpec{Startup: 3, Active: 5, Height: HeightHigh, Damage: 10, Range: 10}},
		{dsl.AtkDivePunch, AttackSpec{Startup: 2, Active: 4, Height: HeightHigh, Damage: 6, Range: 8}},
		{dsl.AtkLowJab, AttackSpec{Startup: 3, Active: 2, Recovery: 5, Height: HeightLow, Damage: 4, Range: 7}},
	}
	for _, tt := range tests {
		if got := c.Attacks[tt.atk]; got != tt.want {
			t.Errorf("%s: got %+v, want %+v", tt.atk, got, tt.want)
		}
	}
}

func TestConfigVocabulary(t *testing.T) {
	if got := fmt.Sprint(InRangeAttacks()); got != "[punch kick sweep throw jump_kick dive_punch low_jab]" {
		t.Errorf("in_range vocabulary = %s", got)
	}
	if n := len(ActionOrder()); n != int(dsl.NumActions) || ActionOrder()[0] != dsl.ActPunch {
		t.Errorf("ActionOrder wrong: %v", ActionOrder())
	}
}

// TestConfigHashPinned fails when a default changes without a version bump.
// When you change a default: bump DefaultVersion and update both constants.
func TestConfigHashPinned(t *testing.T) {
	const (
		pinnedVersion = "2026.11-1"
		pinnedHash    = "f993e64b110ea8d5becc5d7e187049d71b1de9939f79535b4e27b586aa0e66b9"
	)
	c := DefaultConfig()
	got := fmt.Sprintf("%x", sha256.Sum256(encode(c, false)))
	if c.Version != pinnedVersion || got != pinnedHash {
		t.Fatalf("default config changed (version %q, hash %s); bump DefaultVersion and update the pins", c.Version, got)
	}
}
