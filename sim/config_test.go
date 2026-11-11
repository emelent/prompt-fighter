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
		{dsl.AtkThrow, AttackSpec{Startup: 5, Active: 1, Recovery: 18, Height: HeightNone, Damage: 12, Range: 5, Knockdown: true}},
	}
	for _, tt := range tests {
		t.Run(tt.atk.String(), func(t *testing.T) {
			if got := c.Attacks[tt.atk]; got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
	if c.MaxHP != 200 || c.FrameBudget != 1200 || c.StageWidth != 100 || c.StartX != [2]int{30, 70} {
		t.Errorf("stage/round defaults wrong: %+v", c)
	}
	if c.JumpRise != 10 || c.JumpFall != 10 || c.JumpLanding != 2 || c.JumpSpeedX != 1 {
		t.Errorf("jump defaults wrong")
	}
	if c.DuckStartup != 2 || c.DuckRecovery != 2 || c.BlockStartup != 1 || c.BlockRecovery != 1 {
		t.Errorf("duck/block defaults wrong")
	}
	if c.Hitstun != 13 || c.HitPushback != 2 || c.Blockstun != 3 || c.BlockPushback != 2 || c.KnockdownFrames != 20 {
		t.Errorf("stun defaults wrong")
	}
	if c.ComboGrowthPct != 15 || c.ComboKnockdownHits != 5 || c.SensorCost != 2 {
		t.Errorf("combo/sensor defaults wrong")
	}
	if c.Version != "2026.11-4" || c.GuardBreakBlocks != 4 || c.GuardStreakWindow != 60 || c.GuardBreakStun != 20 {
		t.Errorf("guard break defaults wrong")
	}
	if c.PassiveLimit != 120 || c.PassiveBoostStartPct != 110 || c.PassiveBoostStepPct != 10 || c.PassiveBoostStepFrames != 30 || c.PassiveBoostMaxPct != 150 {
		t.Errorf("passivity defaults wrong")
	}
	if c.StaleFrom != 6 || c.StaleStepPct != 15 || c.StaleFloorPct != 40 {
		t.Errorf("stale defaults wrong")
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
		{dsl.AtkJumpKick, AttackSpec{Startup: 3, Active: 5, Height: HeightNone, Damage: 10, Range: 10}},
		{dsl.AtkDivePunch, AttackSpec{Startup: 2, Active: 4, Height: HeightNone, Damage: 6, Range: 8}},
		{dsl.AtkLowJab, AttackSpec{Startup: 3, Active: 2, Recovery: 5, Height: HeightLow, Damage: 4, Range: 7}},
	}
	for _, tt := range tests {
		if got := c.Attacks[tt.atk]; got != tt.want {
			t.Errorf("%s: got %+v, want %+v", tt.atk, got, tt.want)
		}
	}
}

func TestConfigAirZoneDefaults(t *testing.T) {
	c := DefaultConfig()
	want := []AirZone{
		{From: 1, To: 6, Kind: ZoneRisingOverhead},
		{From: 7, To: 13, Kind: ZoneAirToAir},
		{From: 14, To: 16, Kind: ZoneFallingOverhead},
		{From: 17, To: 20, Kind: ZoneLandingLow},
	}
	if fmt.Sprint(c.AirZones) != fmt.Sprint(want) {
		t.Errorf("air zones %v, want %v", c.AirZones, want)
	}
}

// TestConfigAirZonesContiguous checks the zones cover airborne frames
// 1..airFrames exactly once, in order.
func TestConfigAirZonesContiguous(t *testing.T) {
	c := DefaultConfig()
	next := 1
	for _, z := range c.AirZones {
		if z.From != next || z.To < z.From || z.Kind == ZoneNone {
			t.Fatalf("zone %+v: want From=%d and a non-empty range", z, next)
		}
		next = z.To + 1
	}
	if next != c.airFrames()+1 {
		t.Errorf("zones end at %d, want %d", next-1, c.airFrames())
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
		pinnedVersion = "2026.11-4"
		pinnedHash    = "3264f44e55575f49b7946f4a15b9f0ce5345ad056e2a4b0d06cfe55b2440a008"
	)
	c := DefaultConfig()
	got := fmt.Sprintf("%x", sha256.Sum256(encode(c, false)))
	if c.Version != pinnedVersion || got != pinnedHash {
		t.Fatalf("default config changed (version %q, hash %s); bump DefaultVersion and update the pins", c.Version, got)
	}
}
