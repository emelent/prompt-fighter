# Spec Delta

## Purpose

Defines how a validated fight script runs over time: the frames each statement costs, how sensor reads are snapshotted, how stances and overlays work together, and how `on_hit`/`on_block` handlers interrupt and resume the main program.

## ADDED Requirements

### Requirement: Frame cost accounting
Each fighter's interpreter SHALL advance by at most one frame per simulation frame. Statement costs SHALL be:
- An attack occupies the interpreter for its startup + active + recovery frames.
- `move_forward xN` and `move_back xN` cost N frames, one unit per frame.
- `wait N` costs N frames.
- `high_block` and `low_block` cost startup + hold + recovery frames.
- A stance start (`jump*`, `duck`) costs 1 interpreter frame (the frame it begins on); the stance then continues in the background.
- `if`/`elif`/`repeat`/`loop` bookkeeping costs 0 frames apart from their sensor reads.

When a `loop` iteration completes without any frame having elapsed since it began, the back-edge SHALL cost 1 idle frame, so a fighter can never spin without time passing.

#### Scenario: Attack cost
- **WHEN** a script is `punch wait 1` with default config and the fighter is not interrupted
- **THEN** the punch occupies frames 0–9 (startup 0–2, active 3–4, recovery 5–9) and the `wait` occupies frame 10

#### Scenario: Movement cost
- **WHEN** a script is `move_forward x3` and nothing blocks movement
- **THEN** the fighter moves 1 unit toward the opponent on each of 3 consecutive frames and the script ends after them

#### Scenario: Zero-cost loop guard
- **WHEN** a script is `loop { if 1 > 2 { kick } }`
- **THEN** each loop iteration costs exactly 1 idle frame, and the simulation completes the full frame budget

### Requirement: Sensor reads cost 2 frames and are snapshotted
Each sensor read in a condition SHALL cost 2 frames (configurable). Operands SHALL be evaluated left to right with short-circuit `and`/`or`, so unevaluated sensors cost nothing. Integer literals SHALL cost nothing. A sensor's value SHALL be taken from the game state at the start of the first of its 2 frames. A `check` event SHALL be emitted for every sensor read, recording the sensor, the value read, the source line and the cost.

#### Scenario: Two sensors cost four frames
- **WHEN** a fighter evaluates `in_range(sweep) and not enemy.airborne` and `in_range(sweep)` is true
- **THEN** evaluation takes 4 frames and emits 2 `check` events

#### Scenario: Short-circuit skips cost
- **WHEN** a fighter evaluates `in_range(sweep) and not enemy.airborne` and `in_range(sweep)` is false
- **THEN** evaluation takes 2 frames and emits 1 `check` event

#### Scenario: Stale information
- **WHEN** the enemy becomes airborne on the second frame of a fighter's `enemy.airborne` read
- **THEN** the read returns false, because the value was snapshotted on the first frame

#### Scenario: Literal comparisons are free
- **WHEN** a fighter evaluates `health < 30`
- **THEN** evaluation takes 2 frames (one sensor read; the literal is free)

### Requirement: Sensor semantics
Sensors SHALL report the following, from the reading fighter's perspective:
- `distance`: the absolute difference between the fighters' x positions, in units.
- `health` and `enemy.health`: current HP.
- `in_range(a)`: whether `distance` is less than or equal to attack `a`'s range.
- `enemy.airborne`: the enemy is in jump frames 1–20.
- `enemy.ducking`: the enemy is in the hold phase of `duck`, including during a `low_jab` overlay.
- `enemy.blocking`: the enemy is in the guard (hold) phase of `high_block` or `low_block`, or in blockstun.
- `enemy.attacking`: the enemy has an attack, including an overlay attack, in its startup or active phase.
- `enemy.knocked_down`: the enemy is in knockdown.

#### Scenario: Blocking sensor excludes block startup
- **WHEN** the enemy is on the startup frame of `high_block`
- **THEN** `enemy.blocking` reads false

#### Scenario: Attacking sensor excludes recovery
- **WHEN** the enemy is in the recovery phase of `kick`
- **THEN** `enemy.attacking` reads false

### Requirement: Stances and overlays
`jump`, `jump_forward`, `jump_back` and `duck` SHALL be stances. After a stance starts, the interpreter continues with the next statement on the following frame while the stance runs. While a stance is active:
- Sensor reads, `if`/`elif`/`else`, `repeat` bookkeeping and `wait` SHALL proceed concurrently.
- While airborne, the first `kick` SHALL become `jump_kick` and the first `punch` SHALL become `dive_punch`. Only one air attack is allowed per jump.
- While ducking (hold phase), `punch` SHALL become `low_jab`. More than one `low_jab` is allowed per duck.
- An overlay SHALL NOT reset or pause the stance's timer. An overlay attack occupies the interpreter until it finishes.
- Any other action, including a second air attack, SHALL wait until the stance has fully ended (including landing or duck recovery) and then execute normally.

An air attack still in progress when the fighter lands SHALL end on landing. If a `low_jab` is in progress when the duck hold would expire, the hold SHALL extend until the `low_jab` finishes.

#### Scenario: Jump kick overlay
- **WHEN** a script is `jump_forward kick`
- **THEN** the jump starts on frame 0, `jump_kick` starts on frame 1 while airborne, and the jump still lands after 20 airborne frames

#### Scenario: One air attack per jump
- **WHEN** a script is `jump kick kick`
- **THEN** the first `kick` becomes `jump_kick`, and the second `kick` waits until landing recovery ends and then executes as a grounded `kick`

#### Scenario: Low jab overlay
- **WHEN** a script is `duck hold 20 punch`
- **THEN** the `punch` executes as `low_jab` once the duck reaches its hold phase, and the fighter stays ducking

#### Scenario: Non-overlay action waits for stance
- **WHEN** a script is `jump move_forward x2`
- **THEN** the movement begins only after the jump's 20 airborne frames and 2 landing frames have elapsed

#### Scenario: Checks run during a stance
- **WHEN** a script is `jump_forward if distance < 12 { kick }`
- **THEN** the jump starts on airborne frame 1, the `distance` read happens on airborne frames 2–3, and, if the read is true, `jump_kick` starts on airborne frame 4

### Requirement: Interrupts and handlers
When a fighter is hit (entering hitstun or knockdown, including from a throw), its current action and any active stance SHALL be abandoned and its interpreter frozen. When it becomes actionable again, it SHALL run `on_hit` if defined, then resume at the statement after the interrupted one. When a fighter's own attack is blocked, its attack SHALL be abandoned and it SHALL enter blockstun. When that blockstun ends, it SHALL run `on_block` if defined, then resume at the statement after the interrupted one.

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

### Requirement: Script end and idling
When the main program finishes, a `script_end` event SHALL be emitted once and the fighter SHALL idle for the rest of the round. A fighter idling after its script has ended SHALL still run `on_hit`/`on_block` when interrupted, then return to idling.

#### Scenario: Finite script idles
- **WHEN** a script is `kick` in a 600-frame round
- **THEN** after the kick's 18 frames the fighter emits `script_end` and is in the idle state for the remaining frames

#### Scenario: Handler runs after script end
- **WHEN** an idle fighter whose script has ended is hit and has `on_hit { move_back x2 }`
- **THEN** it moves back 2 units after hitstun and returns to idle without emitting a second `script_end`
