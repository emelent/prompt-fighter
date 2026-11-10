# Spec Delta

## Purpose

Defines the Lua fight-script contract that players (and their LLMs) write against: the entry functions, the action and sensor API, the sandboxed standard library, static limits, and the validation errors returned for invalid scripts, which are written so they can be pasted straight back into an LLM.

## ADDED Requirements

### Requirement: Script structure and entry points
A fight script SHALL be Lua 5.1 source text. Loading the script SHALL run its top-level chunk once. After loading, the global `main` MUST be a function; it is the fighter's main program. The globals `on_hit` and `on_block` MAY be defined. If defined, each MUST be a function. Actions and sensors SHALL NOT be callable while the top-level chunk runs. A script whose `main` returns immediately SHALL be valid; that fighter idles.

#### Scenario: Minimal valid script
- **WHEN** the script `function main() while true do punch() end end` is validated
- **THEN** validation succeeds with no errors

#### Scenario: Example script with handlers is valid
- **WHEN** the script defines `main` as `while true do if in_range("sweep") and not enemy.airborne() then sweep() elseif enemy.airborne() then high_block(6) elseif health() < 30 then move_back(10) else move_forward(3) end end`, plus `function on_hit() move_back(2) end` and `function on_block() throw() end`
- **THEN** validation succeeds with no errors

#### Scenario: Missing main
- **WHEN** a script defines only `function on_hit() move_back(2) end`
- **THEN** validation fails with a message stating that the script must define `function main()` and showing a minimal example

#### Scenario: Handler not a function
- **WHEN** a script sets `on_hit = "back"`
- **THEN** validation fails with a message stating that `on_hit` must be a function, e.g. `function on_hit() move_back(2) end`

#### Scenario: Action at top level
- **WHEN** a script calls `punch()` outside any function
- **THEN** validation fails at that line with a message stating that actions and sensors can only be called from inside `main`, `on_hit` or `on_block`

### Requirement: Action API
The following global functions SHALL be provided as actions, each blocking the script until the action finishes or the fighter is interrupted:
- Attacks taking no arguments: `punch()`, `kick()`, `sweep()`, `throw()`.
- Stances taking no arguments: `jump()`, `jump_forward()`, `jump_back()`.
- `duck(n)`, `high_block(n)` and `low_block(n)`, where the hold length `n` is optional. When `n` is omitted, the configured default hold length is used.
- `move_forward(n)` and `move_back(n)`, where `n` is optional and defaults to 1.
- `wait(n)`, where `n` is required.

Every `n` MUST be an integer from 1 to 100 inclusive. Each action SHALL return a string outcome:
- Attacks return `"hit"`, `"blocked"`, `"whiff"`, `"teched"` (throw only) or `"interrupted"`. An overlay attack returns the same outcomes.
- Holds (`duck`, `high_block`, `low_block`) return `"blocked"` if at least one attack was guarded during the hold, `"done"` otherwise, or `"interrupted"`.
- Stances, moves and `wait` return `"done"` or `"interrupted"`.

#### Scenario: Valid argument forms
- **WHEN** a script's `main` calls `move_forward(3)`, `move_back()`, `high_block(6)`, `duck()`, `wait(4)` and `jump_forward()`
- **THEN** validation succeeds

#### Scenario: Attack outcome returned
- **WHEN** a fighter runs `local r = kick()` and the kick is guarded by a `high_block`
- **THEN** `r` is `"blocked"`

#### Scenario: Extra argument to an attack
- **WHEN** a script contains `kick(2)`
- **THEN** validation fails at that line with a message stating that `kick` takes no arguments and suggesting `for i = 1, 2 do kick() end`

#### Scenario: Missing wait duration
- **WHEN** a script contains `wait()`
- **THEN** validation fails with a message stating that `wait` requires a frame count such as `wait(4)`

#### Scenario: Literal count out of range
- **WHEN** a script contains `move_forward(0)` or `wait(500)`
- **THEN** validation fails with a message stating that the count must be an integer between 1 and 100

#### Scenario: Computed count out of range at runtime
- **WHEN** a script calls `wait(n)` with `n` computed as 2.5 at runtime
- **THEN** a runtime script error is raised stating that `wait` needs an integer between 1 and 100, with the source line

#### Scenario: Overlay names are not functions
- **WHEN** a script calls `jump_kick()`
- **THEN** validation fails with a message explaining that `jump_kick` happens by calling `kick()` while airborne, e.g. `jump_forward() kick()`

### Requirement: Sensor API
The following sensors SHALL be provided as functions. Each read blocks the script for the configured sensor cost:
- Integer sensors: `distance()`, `health()`, `enemy.health()`.
- Boolean sensors: `enemy.airborne()`, `enemy.ducking()`, `enemy.blocking()`, `enemy.attacking()`, `enemy.knocked_down()`.
- `in_range(name)`, where `name` is one of the strings `"punch"`, `"kick"`, `"sweep"`, `"throw"`, `"jump_kick"`, `"dive_punch"`, `"low_jab"`. It returns a boolean.

`enemy` SHALL be a read-only table: assigning to its fields SHALL raise an error.

#### Scenario: Sensor used without calling it
- **WHEN** a script contains `if enemy.airborne then kick() end`
- **THEN** validation fails at that line with a message stating that `enemy.airborne` is a sensor function and must be called: `enemy.airborne()`

#### Scenario: Unknown enemy sensor with suggestion
- **WHEN** a script calls `enemy.jumping()`
- **THEN** validation fails at that line with a message listing it as an unknown sensor and suggesting `enemy.airborne`

#### Scenario: in_range with an unknown attack
- **WHEN** a script calls `in_range("jump")` or `in_range(sweep)`
- **THEN** validation fails with a message listing the valid attack names as quoted strings, e.g. `in_range("sweep")`

### Requirement: Sandboxed standard library
Scripts SHALL run with only this deterministic subset of Lua available:
- From the base library: `assert`, `error`, `ipairs`, `next`, `pairs`, `pcall`, `select`, `tonumber`, `tostring`, `type`, `unpack`, `rawequal`, `rawget`, `rawset`, `setmetatable` and `getmetatable`.
- `math`, without `math.random` and `math.randomseed`.
- `string`, without `string.rep` and `string.dump`.
- `table`.
- `print`, which is redirected to the event log.

`io`, `os`, `debug`, `package`, `coroutine`, `require`, `load`, `loadstring`, `loadfile`, `dofile`, `collectgarbage`, `newproxy`, `getfenv` and `setfenv` SHALL NOT be available. `tostring` of a table or function SHALL return just its type name (`"table"`, `"function"`), never a memory address. `pairs` and `next` SHALL iterate in an order that depends only on the script's own operations.

#### Scenario: Forbidden library
- **WHEN** a script calls `os.time()`
- **THEN** validation fails with a message stating that `os` is not available in fight scripts because scripts must be deterministic

#### Scenario: No addresses leak
- **WHEN** a script prints `tostring({})`
- **THEN** the logged text is `table`

### Requirement: Unknown names
Every global name a script reads SHALL be checked statically. A global is known if it is part of the API or the sandboxed library, or if the script assigns it anywhere. Reading an unknown global SHALL fail validation. The closest known name by edit distance (distance ≤ 2) SHALL be suggested. At runtime, reading an undefined global SHALL raise a script error naming the global rather than yielding `nil`.

#### Scenario: Misspelled action with suggestion
- **WHEN** line 3 of a script contains `kik()`
- **THEN** the validator reports `3: unknown name "kik" — did you mean "kick"?`

#### Scenario: Script-defined helper is known
- **WHEN** a script defines `function combo() punch() kick() end` and calls `combo()` from `main`
- **THEN** validation succeeds

### Requirement: Syntax hints for common mistakes
Lua syntax errors SHALL be reported with line and column. Where the offending text matches a construct that LLMs commonly carry over from other languages, the error SHALL include a fix:
- `!=` → `~=`
- `&&`, `||`, `!` → `and`, `or`, `not`
- `elif` or `else if` used with a single `end` → `elseif`
- `{` used to open a block after `then`/`do` or a condition → `then ... end` / `do ... end`
- `while (...) {` or `for (...)` C-style loops → the Lua `while ... do ... end` or `for i = 1, n do ... end` form
- `//` comments → `--`
- `;` is accepted as Lua allows it and needs no hint.

#### Scenario: Not-equal operator
- **WHEN** line 2, column 18 of a script contains `if distance() != 10 then`
- **THEN** the validator reports `2:18: unexpected "!" — Lua writes "not equal" as ~=, e.g. distance() ~= 10`

#### Scenario: Braces instead of then/end
- **WHEN** a script contains `if enemy.ducking() { sweep() }`
- **THEN** the validator reports a syntax error with a hint to write `if enemy.ducking() then sweep() end`

### Requirement: Static limits
Validation SHALL enforce a source size of at most 8192 bytes, and the top-level chunk SHALL complete within the configured load instruction budget. The size check SHALL happen before parsing.

#### Scenario: Oversized source
- **WHEN** a script source is 8193 bytes
- **THEN** validation fails with a message stating the size and the 8192-byte limit, without attempting to parse further

#### Scenario: Endless top-level chunk
- **WHEN** a script's top-level chunk is `while true do end`
- **THEN** validation fails with a message stating that the top-level code ran too long and that loops belong inside `main`

### Requirement: Validation error format
Every validation error SHALL include a 1-based line, a 1-based column when known (counted in bytes), and a human-readable message with a concrete suggestion where a likely fix exists. Text rendering SHALL be one error per line in the form `line:col: message`, or `line: message` when the column is unknown. The validator SHALL report all errors it can find, up to 10, sorted by position. When there is a syntax error, only the syntax error SHALL be reported, since the rest of the script cannot be analysed.

#### Scenario: Multiple static errors reported together
- **WHEN** a script has an unknown name on line 2 and `kick(2)` on line 5
- **THEN** both errors are returned, line 2 first

#### Scenario: Valid script yields no errors
- **WHEN** a valid script is validated
- **THEN** the validator returns an empty error list
