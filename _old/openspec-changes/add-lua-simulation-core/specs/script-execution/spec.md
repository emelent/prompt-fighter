# Spec Delta

## Purpose

Defines how a validated Lua fight script runs over time: the frames each API call costs, the instruction budget on free Lua code, how sensor reads are snapshotted, how stances and overlays interact, how `on_hit`/`on_block` interrupt and resume `main`, and what happens on runtime errors and script end.

## ADDED Requirements

### Requirement: Blocking calls and frame costs
Each fighter's `main` SHALL run as a suspended program that advances by at most one frame of game time per simulation frame. Plain Lua code (arithmetic, locals, tables, control flow, helper functions) SHALL cost no frames. Only API calls consume frames:
- An attack occupies the fighter for its startup + active + recovery frames, then returns its outcome.
- `move_forward(n)` and `move_back(n)` cost `n` frames, one unit per frame.
- `wait(n)` costs `n` frames.
- `high_block(n)` and `low_block(n)` cost startup + hold + recovery frames.
- A stance start (`jump*`, `duck`) costs 1 frame (the frame it begins on). The call then returns and the stance continues in the background.
- A sensor read costs the configured sensor cost (default 2).

#### Scenario: Attack cost
- **WHEN** `main` is `punch() wait(1)` with the default config and the fighter is not interrupted
- **THEN** the punch occupies frames 0–9 (startup 0–2, active 3–4, recovery 5–9) and the `wait` occupies frame 10

#### Scenario: Movement cost
- **WHEN** `main` is `move_forward(3)` and nothing blocks movement
- **THEN** the fighter moves 1 unit toward the opponent on each of 3 consecutive frames, and `main` returns after them

#### Scenario: Free computation
- **WHEN** `main` computes `local n = 0 for i = 1, 50 do n = n + i end` and then calls `wait(n % 7 + 1)`
- **THEN** the loop costs no frames and the wait starts on frame 0

### Requirement: Instruction budget
Lua code SHALL run under a deterministic instruction budget, counted in VM instructions, not wall-clock time. That code runs between two consecutive frame-consuming calls, or from a resume until `main` or a handler returns. The default budget is 10,000 instructions. When the budget is exhausted, the script SHALL fail with a runtime error stating that it ran too long without acting. The error SHALL NOT be catchable by `pcall`. Because the budget resets only when a frame-consuming call is made, a loop that never calls an action or sensor SHALL always fail rather than hang.

#### Scenario: Spin loop fails deterministically
- **WHEN** `main` is `while true do if 1 > 2 then kick() end end`
- **THEN** on frame 0 the script raises a runtime error that it ran too long without calling an action or sensor, and the fighter idles for the rest of the round

#### Scenario: pcall cannot swallow the budget
- **WHEN** `main` is `pcall(function() while true do end end) punch()`
- **THEN** the script fails with the budget error and the `punch` never starts

#### Scenario: Same script, same failure frame
- **WHEN** a script that exhausts its budget on frame 137 is simulated twice
- **THEN** both runs fail on frame 137 with identical `script_error` events

### Requirement: Sensor reads cost frames and are snapshotted
A sensor's value SHALL be taken from the game state at the start of the first frame of its read. The call SHALL return that value once the read's cost has elapsed. A `check` event SHALL be emitted for every sensor read, recording the sensor (including the `in_range` argument), the value read, the source line of the call and the cost. Lua short-circuit `and`/`or` naturally means that unevaluated sensors cost nothing.

#### Scenario: Two sensors cost four frames
- **WHEN** a fighter evaluates `in_range("sweep") and not enemy.airborne()` and `in_range("sweep")` is true
- **THEN** evaluation takes 4 frames and emits 2 `check` events

#### Scenario: Short-circuit skips cost
- **WHEN** a fighter evaluates `in_range("sweep") and not enemy.airborne()` and `in_range("sweep")` is false
- **THEN** evaluation takes 2 frames and emits 1 `check` event

#### Scenario: Stale information
- **WHEN** the enemy becomes airborne on the second frame of a fighter's `enemy.airborne()` read
- **THEN** the read returns false, because the value was snapshotted on the first frame

#### Scenario: Caching a reading is free
- **WHEN** a fighter runs `local d = distance()` and then uses `d` in three comparisons
- **THEN** only one `check` event is emitted and only 2 frames elapse

### Requirement: Sensor semantics
Sensors SHALL report the following, from the reading fighter's perspective:
- `distance()`: the absolute difference between the fighters' x positions, in units.
- `health()` and `enemy.health()`: current HP.
- `in_range(a)`: whether `distance` is less than or equal to attack `a`'s range.
- `enemy.airborne()`: the enemy is in jump frames 1–20.
- `enemy.ducking()`: the enemy is in the hold phase of `duck`, including during a `low_jab` overlay.
- `enemy.blocking()`: the enemy is in the guard (hold) phase of `high_block` or `low_block`, or in blockstun.
- `enemy.attacking()`: the enemy has an attack, including an overlay attack, in its startup or active phase.
- `enemy.knocked_down()`: the enemy is in knockdown.

#### Scenario: Blocking sensor excludes block startup
- **WHEN** the enemy is on the startup frame of `high_block`
- **THEN** `enemy.blocking()` reads false

#### Scenario: Attacking sensor excludes recovery
- **WHEN** the enemy is in the recovery phase of `kick`
- **THEN** `enemy.attacking()` reads false

### Requirement: Stances and overlays
`jump()`, `jump_forward()`, `jump_back()` and `duck()` SHALL be stances. Each call returns after its 1-frame start while the stance continues. While a stance is active:
- Sensor reads, free Lua code and `wait` SHALL proceed concurrently.
- While airborne, the first `kick()` SHALL become `jump_kick` and the first `punch()` SHALL become `dive_punch`. Only one air attack is allowed per jump.
- While ducking (hold phase), `punch()` SHALL become `low_jab`. More than one `low_jab` is allowed per duck.
- An overlay SHALL NOT reset or pause the stance's timer. An overlay attack blocks the script until it finishes.
- Any other action, including a second air attack, SHALL block until the stance has fully ended (including landing or duck recovery) and then execute normally.

An air attack still in progress when the fighter lands SHALL end on landing. If a `low_jab` is in progress when the duck hold would expire, the hold SHALL extend until the `low_jab` finishes.

#### Scenario: Jump kick overlay
- **WHEN** `main` is `jump_forward() kick()`
- **THEN** the jump starts on frame 0, `jump_kick` starts on frame 1 while airborne, and the jump still lands after 20 airborne frames

#### Scenario: One air attack per jump
- **WHEN** `main` is `jump() kick() kick()`
- **THEN** the first `kick()` becomes `jump_kick`, and the second waits until landing recovery ends and then executes as a grounded `kick`

#### Scenario: Low jab overlay
- **WHEN** `main` is `duck(20) punch()`
- **THEN** the `punch()` executes as `low_jab` once the duck reaches its hold phase, and the fighter stays ducking

#### Scenario: Non-overlay action waits for stance
- **WHEN** `main` is `jump() move_forward(2)`
- **THEN** the movement begins only after the jump's 20 airborne frames and 2 landing frames have elapsed

#### Scenario: Checks run during a stance
- **WHEN** `main` is `jump_forward() if distance() < 12 then kick() end`
- **THEN** the jump starts on frame 0, the `distance()` read happens on frames 1–2 while airborne, and, if the read is true, `jump_kick` starts on frame 3

### Requirement: Interrupts and handlers
When a fighter is hit (entering hitstun or knockdown, including from a throw), its current action and any active stance SHALL be abandoned and its script frozen at the pending call. When it becomes actionable again, it SHALL run `on_hit` (if defined) to completion, and then the pending call in `main` SHALL return `"interrupted"`. When a fighter's own attack is blocked, the attack SHALL be abandoned and the fighter SHALL enter blockstun. When that blockstun ends, it SHALL run `on_block` (if defined), and then the pending attack call SHALL return `"blocked"`. A sensor read that is interrupted SHALL return the value it had already snapshotted.

If a fighter is interrupted while running a handler, the handler run SHALL be abandoned. After the new stun, the appropriate handler SHALL start again from its beginning, and `main` stays suspended at its original pending call. A `handler_enter` event SHALL be emitted each time a handler starts. A handler that never returns SHALL keep control indefinitely; `main` then does not resume.

#### Scenario: Resume after hitstun without handler
- **WHEN** a fighter running `local r = kick() move_back(2) punch()` is hit during the kick's startup and defines no `on_hit`
- **THEN** after hitstun `r` is `"interrupted"`, and the fighter executes `move_back(2)` then `punch()`

#### Scenario: on_hit then resume
- **WHEN** a fighter running `while true do move_forward(3) kick() end` with `function on_hit() move_back(2) end` is hit during `move_forward(3)`
- **THEN** after hitstun it emits `handler_enter` for `on_hit`, moves back 2 units, and then `move_forward` returns `"interrupted"` and `kick()` executes

#### Scenario: on_block after own attack is blocked
- **WHEN** a fighter's `kick()` is blocked and its script defines `function on_block() throw() end`
- **THEN** the kick is abandoned, the fighter is in blockstun for the kick's recovery length, then runs `throw()`, and then the `kick()` call returns `"blocked"`

#### Scenario: Interrupt during a sensor read
- **WHEN** a fighter is hit on the second frame of `if enemy.ducking() then sweep() else kick() end`, with the snapshot reading false
- **THEN** after hitstun (and `on_hit`, if any) `enemy.ducking()` returns false and the fighter executes `kick()`

#### Scenario: Interrupt inside handler
- **WHEN** a fighter is hit again while executing its `on_hit` handler
- **THEN** after the new hitstun, `on_hit` starts again from its beginning, and after it returns, the original pending call in `main` returns `"interrupted"`

### Requirement: Runtime errors
A Lua runtime error raised in `main` or a handler and not caught by the script, or an exhausted instruction budget, SHALL stop the script permanently. A `script_error` event SHALL be emitted once, with the message and source line. The fighter SHALL abandon any pending action and idle for the rest of the round, and handlers SHALL no longer run. The round itself SHALL continue normally.

#### Scenario: Nil arithmetic
- **WHEN** `main` executes `local x = nil + 1` on line 4 at frame 30
- **THEN** a `script_error` event is emitted on frame 30 with line 4 and a message describing the error, and the fighter idles for the rest of the round

#### Scenario: Errors caught by the script
- **WHEN** `main` runs `pcall(error, "x")` and then `punch()`
- **THEN** no `script_error` event is emitted and the punch executes

### Requirement: Script end and idling
When `main` returns, a `script_end` event SHALL be emitted once and the fighter SHALL idle for the rest of the round. A fighter idling after `main` has returned SHALL still run `on_hit`/`on_block` when interrupted, then return to idling.

#### Scenario: Finite script idles
- **WHEN** `main` is `kick()` in a 600-frame round
- **THEN** after the kick's 18 frames the fighter emits `script_end` and is in the idle state for the remaining frames

#### Scenario: Handler runs after script end
- **WHEN** an idle fighter whose `main` has returned is hit and defines `function on_hit() move_back(2) end`
- **THEN** it moves back 2 units after hitstun and returns to idle without emitting a second `script_end`

### Requirement: print
`print(...)` SHALL cost no frames and SHALL emit a `print` event containing the tab-joined `tostring` of its arguments, truncated to 200 bytes, and the source line. At most 100 `print` events per fighter per round SHALL be recorded; further calls SHALL be ignored.

#### Scenario: Print is logged
- **WHEN** `main` calls `print("dist", distance())` on line 3 and the read returns 40
- **THEN** a `print` event with text `dist\t40` and line 3 is emitted on the first frame after the read's last frame (the frame on which the code after the read runs)
