# combat-rules Specification

## Purpose

Defines the deterministic rules of a Prompt Fighter round: the versioned config table of tunables, stage and fighter geometry, action frame data, and how hits, blocks, throws, trades, combos, knockdowns, KOs and timeouts resolve.

## Requirements

### Requirement: Versioned config table
All tunables SHALL live in a single config value with a version string. That covers stage geometry, HP, frame budget, per-action frame data, ranges, damage, stun and pushback values, combo parameters, the air-attack zone table, sensor cost, default hold length, script limits, and the guard-break, passivity, stale-attack and combo-growth parameters. Game logic SHALL use only integer arithmetic: integer units for positions and integer percentages for damage scaling. A built-in default config SHALL exist, and its version SHALL be recorded in every event log. Any change to a default value SHALL bump the config version.

#### Scenario: Defaults match the build spec
- **WHEN** the default config is loaded
- **THEN** punch is 3/2/5 frames (startup/active/recovery) with high height, 5 damage and range 8; kick is 6/3/9, high, 9 damage, range 12; sweep is 8/3/14, low, 8 damage, range 12, causing knockdown; throw is 5/1/18, 12 damage, range 5; hitstun is 13 frames; hit pushback is 2 units; HP is 200; the frame budget is 1200; and the stage width is 100

#### Scenario: Overlay defaults
- **WHEN** the default config is loaded
- **THEN** jump_kick is 10 damage, range 10, startup 3, active 5; dive_punch is 6 damage, range 8, startup 2, active 4; both take their height and targets from the air-attack zone; low_jab is low, 4 damage, range 7, startup 3, active 2, recovery 5

#### Scenario: Air zone defaults
- **WHEN** the default config is loaded
- **THEN** the air-attack zones by airborne start frame are 1–6 rising overhead, 7–13 peak air-to-air, 14–16 falling overhead and 17–20 landing low

#### Scenario: Anti-passivity defaults
- **WHEN** the default config is loaded
- **THEN** its version is `2026.11-4`; a guard breaks on the 4th block in a row, the streak lapses after 60 frames, and guard-break stun lasts 20 frames; the passivity limit is 120 frames, the boost starts at 110% and grows 10 points every 30 frames up to 150%; attacks go stale from the 6th repeat, losing 15% of their range per repeat down to 40%; and combos grow by 15 percentage points per hit

### Requirement: Stage, geometry and facing
The stage SHALL span x = 0 to 100 units. Player 1 SHALL start at x=30 and player 2 at x=70, both at y=0 with full HP. Each fighter SHALL have a half-width of 2 units, so fighter centers stay within [2, 98] and are never closer than 4 units apart. Range checks SHALL compare the center-to-center distance against the attack's range. Grounded fighters SHALL always face each other. Fighters SHALL NOT pass through or over each other.

#### Scenario: Edge clamp
- **WHEN** a fighter at x=3 executes `move_back x5` while facing right
- **THEN** its x becomes 2 and stays at 2 for the remaining move frames

#### Scenario: No overlap
- **WHEN** two fighters 5 units apart both execute `move_forward` on the same frame
- **THEN** both moves are cancelled for that frame and their distance stays 5

#### Scenario: Walking into an opponent
- **WHEN** a fighter 4 units from its opponent executes `move_forward x3`
- **THEN** it does not move, and the 3 frames are still consumed

### Requirement: Movement and jumps
`move_forward`/`move_back` SHALL move 1 unit per frame toward or away from the opponent. A jump SHALL be airborne for 20 frames (10 rising, 10 falling), followed by 2 grounded landing-recovery frames during which the fighter cannot act. `jump_forward`/`jump_back` SHALL move 1 unit per airborne frame horizontally, subject to edge and overlap clamping. `y` SHALL rise by 2 units per frame to a peak of 20 and fall symmetrically. Facing SHALL NOT change while airborne.

#### Scenario: Jump forward distance
- **WHEN** a fighter at x=30 executes `jump_forward` with the opponent far away
- **THEN** it lands at x=50 after 20 airborne frames, with y=20 at the peak

### Requirement: Attack phases and connection
An attack SHALL be in startup, active and recovery phases in order. During each active frame, an attack that has not yet connected SHALL connect if the target is within range and none of these whiff conditions apply:
- The target is knocked down (invulnerable).
- A grounded high attack (`punch`, `kick`) meets a ducking target.
- A low attack meets an airborne target.
- An air-to-air attack meets a grounded target.
- An overhead air attack meets an airborne target.
- A throw meets a target that is airborne, in hitstun or knocked down.

An air attack (`jump_kick`, `dive_punch`) SHALL take its zone from the airborne frame (1–20) on which it starts, and keep that zone for its whole duration:
- Frames 1–6, **rising overhead**: hits grounded targets, ducking ones included, and is guarded by `high_block`.
- Frames 7–13, **peak air-to-air**: hits airborne targets only. No block guards it.
- Frames 14–16, **falling overhead**: as rising overhead.
- Frames 17–20, **landing low**: hits grounded targets and is guarded by `low_block`.

An attack SHALL connect at most once. An attack whose active frames end without connecting SHALL emit a `whiff` event that records the reason from its last active frame: `out_of_range`, `ducked`, `airborne`, `grounded`, `invulnerable` or `stunned`.

#### Scenario: High attack whiffs on ducking target
- **WHEN** a `kick` is active, the target is in range, and the target is in the duck hold phase
- **THEN** the kick does not connect, and a `whiff` event with reason `ducked` is emitted when its active frames end

#### Scenario: Sweep whiffs on airborne target
- **WHEN** a `sweep` is active and the target is airborne
- **THEN** the sweep does not connect

#### Scenario: Throw whiffs on hitstun target
- **WHEN** a `throw` becomes active while the target is in hitstun
- **THEN** the throw does not connect and emits `whiff` with reason `stunned`

#### Scenario: Rising overhead hits a ducking target
- **WHEN** a fighter runs `jump_forward kick` so the `jump_kick` starts on airborne frame 2, and the target in range is in the duck hold phase
- **THEN** the jump kick hits the ducking target, and the attacker's jump continues and lands normally

#### Scenario: Overhead is guarded by high_block
- **WHEN** a rising or falling overhead air attack connects with a target in the hold phase of `high_block`
- **THEN** the attack is blocked

#### Scenario: Air-to-air whiffs on a grounded target
- **WHEN** a fighter runs `jump wait 7 kick` so the `jump_kick` starts on airborne frame 9, against a grounded target in range
- **THEN** the jump kick does not connect and emits `whiff` with reason `grounded`

#### Scenario: Air-to-air hits an airborne target
- **WHEN** a peak-zone air attack is active and in range while the target is airborne
- **THEN** it hits, and the target is knocked down and placed at y=0

#### Scenario: Landing low is guarded by low_block, not high_block
- **WHEN** a fighter runs `jump wait 16 kick` so the `jump_kick` starts on airborne frame 18, against a grounded target in range
- **THEN** the attack is blocked by a target holding `low_block` and hits a target holding `high_block`

#### Scenario: Zone is fixed at start
- **WHEN** a `jump_kick` starts on airborne frame 6 and is still active on airborne frames 9–13
- **THEN** it remains a rising overhead for its whole duration and does not connect with an airborne target

### Requirement: Hits
When a non-throw attack connects with a target that is not guarding against its height, the target SHALL take damage. It SHALL enter hitstun for 13 frames, or knockdown when the attack causes knockdown, and be pushed back 2 units away from the attacker. If the target cannot move the full pushback because of the stage edge, the remainder SHALL be applied to the attacker in the opposite direction. Any attack connecting with an airborne target SHALL cause knockdown and return the target to y=0. A `hit` event SHALL record attacker, target, attack, damage, combo count and resulting HP.

#### Scenario: Hit applies hitstun and pushback
- **WHEN** a `punch` connects with an idle target at distance 6
- **THEN** the target loses 5 HP, enters hitstun for 13 frames with no script input, and the distance becomes 8

#### Scenario: Pushback transfers at the wall
- **WHEN** a target at x=3 is hit by an attacker at x=9 standing to its right
- **THEN** the target moves to x=2 and the attacker moves 1 unit away from it, to x=10

#### Scenario: Anti-air knockdown
- **WHEN** a `kick` connects with an airborne target
- **THEN** the target takes damage, is placed at y=0 and enters knockdown

### Requirement: Blocks
`high_block` SHALL guard against high attacks and `low_block` against low attacks, but only during the block's hold phase. Startup and recovery frames do not guard. When an attack connects with a target guarding its height, and the block does not break the guard (see Guard break):
- The defender SHALL take no damage, enter blockstun for 3 frames (during which it keeps guarding and its hold is extended to cover the blockstun if needed) and be pushed back 2 units, with wall transfer as for hits.
- The attacker's attack SHALL be abandoned, and the attacker SHALL enter blockstun for the attack's recovery length.
- A `block` event SHALL be emitted.

A block of the wrong height SHALL NOT guard.

#### Scenario: High block stops a kick
- **WHEN** a `kick` connects with a target in the hold phase of `high_block`
- **THEN** the target takes no damage, is in blockstun for 3 frames and pushed back 2 units, and the kicker is in blockstun for 9 frames

#### Scenario: Wrong-height block
- **WHEN** a `sweep` connects with a target holding `high_block`
- **THEN** the sweep hits and the target is knocked down

### Requirement: Guard break
Each fighter SHALL keep a block streak: the number of attacks it has guarded in a row. The streak SHALL reset to 0 when the fighter starts an attack, when 60 frames pass after its last guarded attack, when it is hit or thrown, and when its guard breaks. When an attack connects with a guarding target whose streak is already 3, the block SHALL instead break the guard:
- The defender SHALL take no damage, have its block abandoned, be pushed back 2 units (with wall transfer as for hits), and enter guard-break stun for 20 frames. During the stun it has no script input and cannot guard, and attacks and throws connect with it as with an idle fighter. Hits during the stun start a new combo at 1.
- The attacker SHALL NOT enter blockstun. Its attack counts as connected and continues into its normal recovery.
- A `guard_break` event SHALL be emitted in place of the `block` event.

#### Scenario: Fourth block breaks the guard
- **WHEN** a fighter holding `high_block` guards three punches in a row, and a fourth punch connects within 60 frames of the third
- **THEN** the fourth punch emits `guard_break` and no `block` event, the defender takes no damage and is in guard-break stun for 20 frames, and the puncher goes straight into its 5 recovery frames

#### Scenario: Punishing a broken guard
- **WHEN** an attacker breaks a guard with `punch` and then starts `kick` as soon as the punch recovers
- **THEN** the kick connects as a hit while the defender is still in guard-break stun

#### Scenario: Streak lapses
- **WHEN** a fighter guards three attacks, then 60 frames pass without it guarding another
- **THEN** its streak is 0, and the next guarded attack is an ordinary block

#### Scenario: Attacking resets the streak
- **WHEN** a fighter guards three attacks, starts a `punch`, then guards another attack
- **THEN** that block is ordinary and its streak is 1

### Requirement: Passivity boost
Each fighter SHALL keep a passive counter, in frames. On every frame where the fighter is neither stunned nor knocked down and does not start an attack, the counter SHALL increase by 1. It increases by 2 on frames where the fighter moves away from the opponent, by `move_back` or by the horizontal motion of `jump_back`. The counter SHALL reset to 0 when the fighter starts an attack (including a throw, an air attack or a `low_jab`).

When a fighter's hit or throw lands on a target whose passive counter is at least 120, its damage SHALL be multiplied by the boost: 110% at 120 frames, plus 10 percentage points for every further full 30 frames, capped at 150%. After the boosted hit or throw lands, the target's passive counter SHALL reset to 0. The boost SHALL NOT apply to blocked attacks or guard breaks, which deal no damage. Damage with both a boost and combo growth SHALL be `max(1, base × combo% × boost% / 10000)` with integer division. A hit or throw with a boost above 100% SHALL record `boost` in its event.

#### Scenario: Boost thresholds
- **WHEN** the target's passive counter is 119, 120, 150, 239 and 240 frames when a hit lands
- **THEN** the boost is 100%, 110%, 120%, 140% and 150% respectively

#### Scenario: Waiting is punished
- **WHEN** p1's `kick` (base 9) lands as the first hit of a combo on p2, whose passive counter is 150
- **THEN** the kick deals 9 × 100 × 120 / 10000 = 10 damage, the `hit` event records `boost: 120`, and p2's passive counter resets to 0

#### Scenario: Running away counts double
- **WHEN** a fighter does nothing but `move_back` for 60 frames
- **THEN** its passive counter is 120 and the opponent's next hit is boosted

#### Scenario: Attacking resets the counter
- **WHEN** a fighter that has been passive for 200 frames starts a `punch`, even one that whiffs
- **THEN** its passive counter is 0 and the opponent's next hit is not boosted

### Requirement: Stale attacks
Each fighter SHALL count how many times in a row it has started the same attack. Overlay attacks (`jump_kick`, `dive_punch`, `low_jab`) count as their own attacks, separate from `kick` and `punch`. Starting a different attack SHALL reset the count to 1. Other actions SHALL NOT affect it. From the 6th repeat on, the attack is stale, and its range SHALL be multiplied by `max(40, 100 − 15 × (repeats − 5))` percent with integer division. That gives 85% on the 6th repeat, 70% on the 7th, 55% on the 8th and 40% from the 9th. A `stale` event SHALL be emitted when a stale attack starts. A stale attack whose target is within the attack's full range but beyond its reduced range SHALL whiff with reason `stale`. Staleness SHALL NOT change damage or frame data, and it involves no randomness.

#### Scenario: Sixth punch in a row
- **WHEN** a fighter starts its 6th `punch` in a row with the target at distance 7
- **THEN** a `stale` event is emitted with repeats 6, the punch's range is 8 × 85 / 100 = 6, and the punch whiffs with reason `stale`

#### Scenario: Mixing attacks resets
- **WHEN** a fighter has punched 5 times in a row, then kicks, then punches again
- **THEN** the kick and the following punch both have repeats 1 and full range

### Requirement: Throws and the triangle
A throw that connects SHALL ignore blocking, deal its damage and knock the target down. Attacks beat throws: if a thrower is hit on any frame before or during its throw's active frame, the throw is abandoned along with the rest of its action. If both fighters' throws connect on the same frame, both SHALL be teched: no damage, both pushed back 2 units, a `throw` event with outcome `tech` for each, and both continue into recovery. A successful throw SHALL emit a `throw` event with outcome `success` and a `knockdown` event. Throw recovery SHALL be long enough that a throw repeated immediately after a successful throw does not become active before an attack started on the target's first actionable frame. The triangle therefore holds on wake-up as well.

#### Scenario: Throw beats block
- **WHEN** a `throw` becomes active within range of a target holding `high_block` or `low_block`
- **THEN** the target takes 12 damage and is knocked down

#### Scenario: Attack beats throw
- **WHEN** a `punch` connects with a fighter on frame 3 of its throw startup
- **THEN** the thrower takes punch damage and enters hitstun, and its throw never becomes active

#### Scenario: Simultaneous throws tech
- **WHEN** both fighters' throws become active and in range on the same frame
- **THEN** neither takes damage and both are pushed apart by 2 units

#### Scenario: Wake-up attack beats a repeated throw
- **WHEN** a fighter throws, throws again immediately, and the thrown target starts `punch` on its first actionable frame after the knockdown
- **THEN** the punch hits the thrower and the second throw never connects

#### Scenario: Repeated throw beats a wake-up block
- **WHEN** a fighter throws, throws again immediately, and the thrown target starts `high_block` on its first actionable frame after the knockdown
- **THEN** the second throw connects and knocks the target down again

### Requirement: Simultaneous resolution and trades
Each frame SHALL resolve both fighters simultaneously from the same pre-resolution state, so that the outcome never depends on which player is player 1. If both fighters' attacks connect on the same frame, both SHALL take their respective damage and stun, and a `trade` event SHALL be emitted in addition to the two `hit` events. When an attack and a throw would both connect on the same frame, only the attack SHALL apply.

#### Scenario: Punch trade
- **WHEN** two fighters 7 units apart both start `punch` on the same frame
- **THEN** both punches connect on the same frame, both fighters lose 5 HP, and one `trade` event is emitted

### Requirement: Combo growth
A defender's combo count SHALL increment on each hit that lands while it is still in hitstun from a previous hit, and SHALL reset when the defender becomes actionable or is knocked down. The combo multiplier SHALL be `100 + 15 × (combo − 1)` percent, so the first hit deals base damage and each further hit 15 percentage points more. Damage SHALL be `max(1, base × combo% / 100)` with integer division, combined with any passivity boost as described in Passivity boost. The 5th hit of a combo SHALL cause knockdown.

#### Scenario: Growth sequence
- **WHEN** the multiplier is computed for hits 1 through 5 of a combo
- **THEN** the multipliers are 100, 115, 130, 145 and 160 percent, and a base-10 attack deals 10, 11, 13, 14 and 16 damage

#### Scenario: Combo knockdown cap
- **WHEN** a defender takes its 5th hit within one combo
- **THEN** it enters knockdown instead of hitstun and its combo count resets

#### Scenario: Combos link at the defaults
- **WHEN** a fighter runs `punch punch kick` against an idle target at distance 6 with the default config
- **THEN** all three hits connect with combo counts 1, 2 and 3 and damage 5, 5 and 11, leaving the target at 79 HP

#### Scenario: Pushback ends punch strings
- **WHEN** a fighter runs `punch punch punch` against an idle target at distance 6 with the default config
- **THEN** the first two punches hit and the third whiffs with reason `out_of_range`

### Requirement: Knockdown
A knocked-down fighter SHALL stay on the ground for 20 frames, invulnerable to all attacks and throws, with no script input. It then stands in the idle state and becomes actionable. A `knockdown` event SHALL be emitted when knockdown begins.

#### Scenario: Invulnerable on the ground
- **WHEN** an attack is active and in range against a knocked-down target
- **THEN** it does not connect and whiffs with reason `invulnerable`

#### Scenario: Wake-up
- **WHEN** 20 frames of knockdown have elapsed
- **THEN** the fighter is idle, vulnerable and runs `on_hit` if it has one

### Requirement: Round end and result
A round SHALL end at the end of the first frame on which any fighter's HP reaches 0 (KO), or after the frame budget is exhausted (timeout). HP SHALL NOT go below 0. On KO, the fighter with HP remaining wins; if both reach 0 on the same frame, the result is a draw (`double_ko`). On timeout, the higher HP wins and equal HP is a draw. A `ko` event SHALL be emitted for each KO'd fighter.

#### Scenario: KO ends the round early
- **WHEN** player 2's HP reaches 0 on frame 412
- **THEN** the round ends after frame 412 with player 1 the winner by `ko`, and no frames after 412 are simulated

#### Scenario: Timeout draw
- **WHEN** the frame budget expires with both fighters at 63 HP
- **THEN** the result is a draw by `timeout`

#### Scenario: Double KO
- **WHEN** a trade reduces both fighters to 0 HP on the same frame
- **THEN** the result is a draw by `double_ko` and two `ko` events are emitted
