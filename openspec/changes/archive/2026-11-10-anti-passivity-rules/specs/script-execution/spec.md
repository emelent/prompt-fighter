# Spec Delta

## MODIFIED Requirements

### Requirement: Sensor semantics
Sensors SHALL report the following, from the reading fighter's perspective:
- `distance`: the absolute difference between the fighters' x positions, in units.
- `health` and `enemy.health`: current HP.
- `in_range(a)`: whether `distance` is less than or equal to attack `a`'s range.
- `enemy.airborne`: the enemy is in jump frames 1–20.
- `enemy.ducking`: the enemy is in the hold phase of `duck`, including during a `low_jab` overlay.
- `enemy.blocking`: the enemy is in the guard (hold) phase of `high_block` or `low_block`, or in blockstun (not in guard-break stun).
- `enemy.attacking`: the enemy has an attack, including an overlay attack, in its startup or active phase.
- `enemy.knocked_down`: the enemy is in knockdown.
- `guard`: the reading fighter's block streak (0–3; see Guard break in combat rules).
- `passive` and `enemy.passive`: the reading fighter's and the enemy's passive counters, capped at 999. A value of 120 or more means the opponent's next hit or throw is boosted.
- `repeats`: how many times in a row the reading fighter has started its most recent attack (0 before its first attack). The next start of that same attack is stale when `repeats` is 5 or more.

#### Scenario: Blocking sensor excludes block startup
- **WHEN** the enemy is on the startup frame of `high_block`
- **THEN** `enemy.blocking` reads false

#### Scenario: Attacking sensor excludes recovery
- **WHEN** the enemy is in the recovery phase of `kick`
- **THEN** `enemy.attacking` reads false

#### Scenario: Reading your own passivity
- **WHEN** a fighter has neither attacked nor been stunned for the first 100 frames and then reads `passive`
- **THEN** the value is 100

### Requirement: Interrupts and handlers
When a fighter is hit (entering hitstun or knockdown, including from a throw) or has its guard broken, its current action and any active stance SHALL be abandoned and its interpreter frozen. When it becomes actionable again, it SHALL run `on_hit` if defined, then resume at the statement after the interrupted one. When a fighter's own attack is blocked, its attack SHALL be abandoned and it SHALL enter blockstun. When that blockstun ends, it SHALL run `on_block` if defined, then resume at the statement after the interrupted one.

"The statement after" SHALL be resolved structurally:
- An interrupted action inside a block continues with the next statement in that block.
- An interruption during an `if`/`elif` condition skips the whole `if` chain.
- When the interrupted statement was the last in a `repeat` or `loop` body, the next iteration begins (or the `repeat` finishes when its count is exhausted).

If a fighter is interrupted while running a handler, the handler SHALL be abandoned, the appropriate handler SHALL run again from its start after the new stun, and the resume point SHALL remain the one saved by the original interrupt. A `handler_enter` event SHALL be emitted each time a handler starts.

#### Scenario: Resume after hitstun without handler
- **WHEN** a fighter running `kick move_back x2 punch` is hit during the kick's startup and has no `on_hit`
- **THEN** after hitstun ends it executes `move_back x2` then `punch`

#### Scenario: on_hit then resume
- **WHEN** a fighter running `loop { move_forward x3 kick }` with `on_hit { move_back x2 }` is hit during `move_forward x3`
- **THEN** after hitstun it emits `handler_enter` for `on_hit`, moves back 2 units, then executes `kick`

#### Scenario: on_block after own attack is blocked
- **WHEN** a fighter's `kick` is blocked and its script has `on_block { throw }`
- **THEN** the kick is abandoned, the fighter is in blockstun for the kick's recovery length, then runs `throw`, then resumes after the `kick` statement

#### Scenario: Interrupt during condition evaluation
- **WHEN** a fighter is hit during the 2-frame read of `if enemy.ducking { sweep } else { kick }` followed by `wait 5`
- **THEN** after hitstun (and `on_hit`, if any) it resumes at `wait 5`

#### Scenario: Interrupt inside handler
- **WHEN** a fighter is hit again while executing its `on_hit` handler
- **THEN** after the new hitstun, `on_hit` starts again from its first statement, and afterward the fighter resumes at the original resume point

#### Scenario: on_hit after a guard break
- **WHEN** a fighter's guard is broken and its script has `on_hit { move_back x3 }`
- **THEN** after the 20-frame guard-break stun it emits `handler_enter` for `on_hit` and moves back 3 units
