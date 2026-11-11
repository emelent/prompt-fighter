# replay-log Specification

## Purpose

Defines the event log JSON produced by a round simulation. It is the only contract between the engine and every consumer (CLI, server, MCP tools, replay viewer). It must be complete enough that a renderer never needs game logic, and byte-for-byte deterministic.

## Requirements

### Requirement: Log header
The log SHALL be a single JSON object containing:
- `format_version`: starting at `"1"`, bumped on any breaking schema change.
- `config_version`
- `frame_budget`
- `max_hp`: each fighter's starting HP
- `players`: an array of exactly 2 objects with `id` (`"p1"`, `"p2"`), `name` (the script name supplied by the caller) and `script` (the full source text).
- `result`: an object with `winner` (`"p1"`, `"p2"` or `null`), `reason` (`ko`, `double_ko` or `timeout`), `end_frame` (index of the last simulated frame) and `hp` (final HP for each player).

#### Scenario: Header contents
- **WHEN** `turtle.pf` (p1) and `rushdown.pf` (p2) are simulated with the default config
- **THEN** the log's `players[0].script` equals the exact contents of `turtle.pf`, `config_version` equals the default config version, `frame_budget` is 1200 and `max_hp` is 200

#### Scenario: Custom frame budget recorded
- **WHEN** a round is simulated with a frame budget of 300
- **THEN** `frame_budget` is 300 and `frames` has at most 300 entries

### Requirement: Per-frame state
`frames` SHALL be an array with one entry per simulated frame, indexed from 0, each recording the state at the end of that frame. Each entry SHALL contain `f` (frame index) and, for each player in order p1 then p2:
- `x`, `y` (integers)
- `facing` (`1` = facing +x, `-1` = facing -x)
- `state`: one of `idle`, `walk`, `attack`, `block`, `duck`, `airborne`, `landing`, `hitstun`, `blockstun`, `guard_break`, `knockdown`, `ko`
- `anim`: e.g. `idle`, `check`, `walk_forward`, `walk_back`, `punch`, `kick`, `sweep`, `throw`, `jump`, `jump_kick`, `dive_punch`, `duck`, `low_jab`, `high_block`, `low_block`, `hitstun`, `blockstun`, `guard_break`, `knockdown`, `ko`, `wait`
- `anim_frame`: frames elapsed since `anim` began, starting at 0
- `phase`: `startup`, `active`, `recovery` or empty
- `hp`
- `invulnerable` (boolean)
- `combo`: the current combo count on this fighter as defender
- `script_line`: the 1-based source line of the statement the interpreter is executing, or 0 when not executing (stunned, or script ended)

#### Scenario: Frame count on timeout
- **WHEN** a round runs to the 1200-frame timeout
- **THEN** `frames` has exactly 1200 entries with `f` from 0 to 1199

#### Scenario: Line highlighting data
- **WHEN** p1 is on the second frame of a sensor read on line 4 of its script
- **THEN** that frame's p1 entry has `script_line` 4 and `anim` `check` (or its stance's animation when a stance is active)

#### Scenario: Renderer needs no game logic
- **WHEN** a fighter is in the active phase of `jump_kick`
- **THEN** its frame entry has `state` `airborne`, `anim` `jump_kick`, `phase` `active` and its current `x` and `y`, with no other derivation required

### Requirement: Events
`events` SHALL be an array of objects `{frame, type, actor, target?, detail}`, where `type` is one of `hit`, `block`, `whiff`, `throw`, `trade`, `knockdown`, `ko`, `check`, `script_end`, `handler_enter`, `guard_break`, `stale`. `detail` SHALL be a JSON object whose fields depend on the type and are documented in `docs/RULES.md`. At minimum:
- `hit`: attack, damage, combo, hp, and `boost` (percent) when above 100
- `block`: attack
- `whiff`: attack, reason (including `stale`)
- `throw`: outcome, and `boost` when above 100
- `guard_break`: attack (actor is the attacker, target the defender)
- `stale`: attack, repeats, range (the reduced range in units)
- `trade`: both attacks and damages
- `check`: sensor, value, line, cost
- `handler_enter`: handler

Events SHALL be ordered by frame, then p1's events before p2's within a frame, then by emission order.

#### Scenario: Check event
- **WHEN** p2 reads `distance` on line 2 and gets 40
- **THEN** an event `{"type":"check","actor":"p2","detail":{"sensor":"distance","value":40,"line":2,"cost":2}}` is emitted on the read's first frame

#### Scenario: Hit event
- **WHEN** p1's kick hits p2 for 9 damage leaving 91 HP
- **THEN** an event with `type` `hit`, `actor` `p1`, `target` `p2` and detail containing `attack: "kick"`, `damage: 9`, `combo: 1`, `hp: 91` is emitted

#### Scenario: Guard break event
- **WHEN** p1's punch breaks p2's guard
- **THEN** an event with `type` `guard_break`, `actor` `p1`, `target` `p2` and `detail.attack` `punch` is emitted, and no `block` event for that punch

#### Scenario: Boosted hit event
- **WHEN** p1's kick lands on p2 while p2's passive counter is 150
- **THEN** the `hit` event's detail includes `boost: 120`

### Requirement: Determinism
Simulating the same two scripts with the same config and frame budget SHALL produce byte-identical JSON output, across runs and across processes. The simulation SHALL NOT use randomness, wall-clock time, floating point in game logic, or map iteration order.

#### Scenario: Run twice
- **WHEN** the same script pair is simulated twice in one process and once in a separate process
- **THEN** all three JSON outputs are byte-identical

### Requirement: Side symmetry
Swapping which script is p1 and which is p2 SHALL produce a mirrored round: the same per-frame states with x mirrored as `100 - x`, facing negated and players exchanged, and the same result with the winner exchanged. A script played against itself SHALL therefore always draw.

#### Scenario: Mirror match
- **WHEN** any sample script is simulated against itself
- **THEN** the result is a draw and every frame has `p1.x == 100 - p2.x`

#### Scenario: Swapped sides
- **WHEN** `rushdown.pf` vs `thrower.pf` is simulated, and then `thrower.pf` vs `rushdown.pf`
- **THEN** the second log's frames equal the first log's frames with players exchanged and x mirrored, and the winner is the same script in both
