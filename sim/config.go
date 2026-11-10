// Package sim runs deterministic Prompt Fighter rounds and produces event logs.
package sim

import "promptfighter/sim/dsl"

// Height is the hit height of an attack.
type Height int

const (
	HeightNone Height = iota // throws
	HeightHigh
	HeightLow
)

func (h Height) String() string {
	switch h {
	case HeightHigh:
		return "high"
	case HeightLow:
		return "low"
	}
	return "none"
}

// MarshalText makes heights readable in the encoded config.
func (h Height) MarshalText() ([]byte, error) { return []byte(h.String()), nil }

// AirZoneKind classifies an air attack by the airborne frame it starts on.
type AirZoneKind int

const (
	ZoneNone AirZoneKind = iota // not an air attack
	ZoneRisingOverhead
	ZoneAirToAir
	ZoneFallingOverhead
	ZoneLandingLow
)

func (z AirZoneKind) String() string {
	switch z {
	case ZoneRisingOverhead:
		return "rising_overhead"
	case ZoneAirToAir:
		return "air_to_air"
	case ZoneFallingOverhead:
		return "falling_overhead"
	case ZoneLandingLow:
		return "landing_low"
	}
	return "none"
}

// MarshalText makes zones readable in the encoded config.
func (z AirZoneKind) MarshalText() ([]byte, error) { return []byte(z.String()), nil }

// height is the hit height an air attack takes from its zone.
func (z AirZoneKind) height() Height {
	switch z {
	case ZoneRisingOverhead, ZoneFallingOverhead:
		return HeightHigh
	case ZoneLandingLow:
		return HeightLow
	}
	return HeightNone
}

// AirZone maps the airborne start frames From..To (inclusive, 1-based) to a zone.
type AirZone struct {
	From int         `json:"from"`
	To   int         `json:"to"`
	Kind AirZoneKind `json:"kind"`
}

// AttackSpec is the frame data for one attack.
type AttackSpec struct {
	Startup   int    `json:"startup"`
	Active    int    `json:"active"`
	Recovery  int    `json:"recovery"`
	Height    Height `json:"height"`
	Damage    int    `json:"damage"`
	Range     int    `json:"range"`
	Knockdown bool   `json:"knockdown"`
}

// Total returns startup + active + recovery.
func (a AttackSpec) Total() int { return a.Startup + a.Active + a.Recovery }

// Config is the single versioned table of every tunable. Game logic only uses
// integer arithmetic on these values.
type Config struct {
	Version string `json:"version"`

	StageWidth  int    `json:"stage_width"`
	StartX      [2]int `json:"start_x"`
	HalfWidth   int    `json:"half_width"` // min separation is 2*HalfWidth; x is clamped to [HalfWidth, StageWidth-HalfWidth]
	MaxHP       int    `json:"max_hp"`
	FrameBudget int    `json:"frame_budget"`

	// Attacks is indexed by dsl.Attack (punch, kick, sweep, throw, jump_kick, dive_punch, low_jab).
	// The Height of jump_kick and dive_punch is unused: air attacks take it from AirZones.
	Attacks [dsl.NumAttacks]AttackSpec `json:"attacks"`

	// AirZones decides what an air attack can hit from the airborne frame it
	// starts on. Ordered by frame; must cover 1..JumpRise+JumpFall exactly once.
	AirZones []AirZone `json:"air_zones"`

	JumpRise      int `json:"jump_rise"`        // airborne frames going up
	JumpFall      int `json:"jump_fall"`        // airborne frames coming down
	JumpLanding   int `json:"jump_landing"`     // grounded recovery frames after landing
	JumpSpeedX    int `json:"jump_speed_x"`     // units per airborne frame for jump_forward/jump_back
	JumpYPerFrame int `json:"jump_y_per_frame"` // visual height change per airborne frame

	DuckStartup   int `json:"duck_startup"`
	DuckRecovery  int `json:"duck_recovery"`
	BlockStartup  int `json:"block_startup"`
	BlockRecovery int `json:"block_recovery"`
	DefaultHold   int `json:"default_hold"`
	WalkSpeed     int `json:"walk_speed"`

	Hitstun         int `json:"hitstun"`
	HitPushback     int `json:"hit_pushback"`
	Blockstun       int `json:"blockstun"`
	BlockPushback   int `json:"block_pushback"`
	KnockdownFrames int `json:"knockdown_frames"`

	ComboScalePct      int `json:"combo_scale_pct"`
	ComboFloorPct      int `json:"combo_floor_pct"`
	ComboKnockdownHits int `json:"combo_knockdown_hits"`

	SensorCost int `json:"sensor_cost"`

	Limits dsl.Limits `json:"limits"`
}

// DefaultVersion is the version string of DefaultConfig. Bump it whenever a
// default value changes (TestConfigHashPinned enforces this).
const DefaultVersion = "2026.11-2"

// DefaultConfig returns the build-spec defaults plus the gap-fill values
// documented in docs/RULES.md.
func DefaultConfig() Config {
	c := Config{
		Version:     DefaultVersion,
		StageWidth:  100,
		StartX:      [2]int{30, 70},
		HalfWidth:   2,
		MaxHP:       100,
		FrameBudget: 600,

		JumpRise:      10,
		JumpFall:      10,
		JumpLanding:   2,
		JumpSpeedX:    1,
		JumpYPerFrame: 2,

		DuckStartup:   2,
		DuckRecovery:  2,
		BlockStartup:  1,
		BlockRecovery: 1,
		DefaultHold:   10,
		WalkSpeed:     1,

		Hitstun:         13,
		HitPushback:     2,
		Blockstun:       3,
		BlockPushback:   2,
		KnockdownFrames: 20,

		ComboScalePct:      80,
		ComboFloorPct:      30,
		ComboKnockdownHits: 5,

		SensorCost: 2,
		Limits:     dsl.DefaultLimits(),

		AirZones: []AirZone{
			{From: 1, To: 6, Kind: ZoneRisingOverhead},
			{From: 7, To: 13, Kind: ZoneAirToAir},
			{From: 14, To: 16, Kind: ZoneFallingOverhead},
			{From: 17, To: 20, Kind: ZoneLandingLow},
		},
	}
	c.Attacks[dsl.AtkPunch] = AttackSpec{Startup: 3, Active: 2, Recovery: 5, Height: HeightHigh, Damage: 5, Range: 8}
	c.Attacks[dsl.AtkKick] = AttackSpec{Startup: 6, Active: 3, Recovery: 9, Height: HeightHigh, Damage: 9, Range: 12}
	c.Attacks[dsl.AtkSweep] = AttackSpec{Startup: 8, Active: 3, Recovery: 14, Height: HeightLow, Damage: 8, Range: 12, Knockdown: true}
	c.Attacks[dsl.AtkThrow] = AttackSpec{Startup: 5, Active: 1, Recovery: 18, Height: HeightNone, Damage: 12, Range: 5, Knockdown: true}
	c.Attacks[dsl.AtkJumpKick] = AttackSpec{Startup: 3, Active: 5, Recovery: 0, Height: HeightNone, Damage: 10, Range: 10}
	c.Attacks[dsl.AtkDivePunch] = AttackSpec{Startup: 2, Active: 4, Recovery: 0, Height: HeightNone, Damage: 6, Range: 8}
	c.Attacks[dsl.AtkLowJab] = AttackSpec{Startup: 3, Active: 2, Recovery: 5, Height: HeightLow, Damage: 4, Range: 7}
	return c
}

// ActionOrder is the canonical ordered list of statement actions.
func ActionOrder() []dsl.Action {
	out := make([]dsl.Action, dsl.NumActions)
	for i := range out {
		out[i] = dsl.Action(i)
	}
	return out
}

// InRangeAttacks is the vocabulary accepted by in_range(...), in canonical order.
func InRangeAttacks() []string { return dsl.AttackNames() }

func (c *Config) airFrames() int { return c.JumpRise + c.JumpFall }

// airZone returns the zone for an air attack starting on airborne frame k.
func (c *Config) airZone(k int) AirZoneKind {
	for _, z := range c.AirZones {
		if k >= z.From && k <= z.To {
			return z.Kind
		}
	}
	return ZoneAirToAir
}
func (c *Config) minSep() int { return 2 * c.HalfWidth }
