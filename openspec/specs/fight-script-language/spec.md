# fight-script-language Specification

## Purpose

Defines the `.pf` fight-script language that players (and their LLMs) write: its syntax, the actions and sensors it can use, the static limits on a script, and the validation errors returned for invalid scripts. Those errors are written to be pasted straight back into an LLM.

## Requirements

### Requirement: Script structure
A script SHALL consist of a sequence of top-level statements forming the main program, plus at most one `on_hit { ... }` and at most one `on_block { ... }` handler block, which MAY appear anywhere at top level. Statements SHALL be separated by whitespace only; newlines are not significant. `#` SHALL start a comment that runs to end of line. Keywords, actions and sensors SHALL be lowercase and case-sensitive. A script whose main program is empty SHALL be valid; that fighter idles.

#### Scenario: Example script from the build spec is valid
- **WHEN** the script `loop { if in_range(sweep) and not enemy.airborne { sweep } elif enemy.airborne { high_block hold 6 } elif health < 30 { move_back x10 } else { move_forward x3 } } on_hit { move_back x2 } on_block { throw }` is validated
- **THEN** validation succeeds with no errors

#### Scenario: Single-line blocks
- **WHEN** a script contains `if enemy.ducking { sweep } else { kick }` on one line
- **THEN** it parses identically to the same statements spread over several lines

#### Scenario: Duplicate handler
- **WHEN** a script defines `on_hit { ... }` twice
- **THEN** validation fails with an error at the second `on_hit` stating that only one `on_hit` handler is allowed

#### Scenario: Handler not at top level
- **WHEN** `on_block { throw }` appears inside a `loop` body
- **THEN** validation fails with an error stating that handlers must be declared at the top level of the script

### Requirement: Action statements
The language SHALL support these action statements and argument forms, and reject any other combination:
- Attacks with no arguments: `punch`, `kick`, `sweep`, `throw`.
- Stances with no arguments: `jump`, `jump_forward`, `jump_back`.
- Holds: `duck`, `high_block`, `low_block`, each optionally followed by `hold N`. When `hold` is omitted, the configured default hold length is used.
- Movement: `move_forward` and `move_back`, each optionally followed by `xN` (written with no space, e.g. `x3`). When omitted, the count is 1.
- `wait N`, where `N` is required.
- `N` SHALL be an integer from 1 to 100 inclusive.

#### Scenario: Valid argument forms
- **WHEN** a script contains `move_forward x3`, `move_back`, `high_block hold 6`, `duck`, `wait 4` and `jump_forward`
- **THEN** validation succeeds

#### Scenario: Wrong argument form for an action
- **WHEN** a script contains `kick x2`
- **THEN** validation fails at the `x2` token with a message stating that `kick` takes no count and suggesting `repeat 2 { kick }`

#### Scenario: Missing wait duration
- **WHEN** a script contains `wait` with no number after it
- **THEN** validation fails with a message stating that `wait` requires a frame count such as `wait 4`

#### Scenario: Count out of range
- **WHEN** a script contains `move_forward x0` or `wait 500`
- **THEN** validation fails with a message stating that the count must be between 1 and 100

#### Scenario: Overlay names are not standalone actions
- **WHEN** a script contains `jump_kick` as a statement
- **THEN** validation fails with a message explaining that `jump_kick` is produced by issuing `kick` while airborne, e.g. `jump_forward kick`

### Requirement: Control flow statements
The language SHALL support `if <cond> { ... }` followed by zero or more `elif <cond> { ... }` and an optional `else { ... }`; `repeat N { ... }` with `N` from 1 to 100; and `loop { ... }`, which repeats its body forever. A `loop` body SHALL NOT be empty. `loop` SHALL NOT appear inside a handler.

#### Scenario: Empty loop rejected
- **WHEN** a script contains `loop { }`
- **THEN** validation fails with a message stating that a `loop` body must contain at least one statement

#### Scenario: Loop inside handler rejected
- **WHEN** a script contains `on_hit { loop { punch } }`
- **THEN** validation fails with a message stating that `loop` is not allowed inside handlers because a handler must finish so the main script can resume

#### Scenario: Dangling elif
- **WHEN** a script contains `elif health < 30 { kick }` without a preceding `if`
- **THEN** validation fails with a message stating that `elif` must follow an `if` or `elif` block

### Requirement: Conditions
A condition SHALL be an expression built from:
- Boolean sensors: `enemy.airborne`, `enemy.ducking`, `enemy.blocking`, `enemy.attacking`, `enemy.knocked_down`, and `in_range(<attack>)`, where `<attack>` is one of `punch`, `kick`, `sweep`, `throw`, `jump_kick`, `dive_punch`, `low_jab`.
- Integer sensors: `distance`, `health`, `enemy.health`, `guard`, `passive`, `enemy.passive`, `repeats`.
- Integer literals from 0 to 999.
- Comparisons `<`, `<=`, `>`, `>=`, `==`, `!=` between two integer operands.
- `not`, `and`, `or`, and parentheses.

Precedence from highest to lowest SHALL be: parentheses, comparison, `not`, `and`, `or`. Type errors SHALL be rejected: a boolean used in a comparison, an integer used as a boolean, or a comparison chained onto another comparison.

#### Scenario: Valid compound condition
- **WHEN** a condition is `(distance <= 12 or enemy.attacking) and not enemy.knocked_down`
- **THEN** validation succeeds

#### Scenario: Integer used as boolean
- **WHEN** a condition is `if health { kick }`
- **THEN** validation fails with a message stating that `health` is a number and must be compared, e.g. `health < 30`

#### Scenario: Boolean compared to number
- **WHEN** a condition is `enemy.airborne == 1`
- **THEN** validation fails with a message stating that `enemy.airborne` is true/false and should be used directly, e.g. `if enemy.airborne { ... }`

#### Scenario: Unknown sensor with suggestion
- **WHEN** a condition is `enemy.jumping`
- **THEN** validation fails at that token with a message listing it as an unknown sensor and suggesting `enemy.airborne`

#### Scenario: in_range with a non-attack argument
- **WHEN** a condition is `in_range(jump)`
- **THEN** validation fails with a message listing the valid attack names for `in_range`

#### Scenario: Anti-passivity sensors
- **WHEN** a condition is `passive > 90 or guard >= 3 or repeats >= 5`
- **THEN** validation succeeds

### Requirement: Static limits
The validator SHALL enforce:
- Source size of at most 4096 bytes.
- At most 40 statements, where each action, `if`, `elif`, `repeat` and `loop` counts as one (and `else` and handler declarations do not), counted across the main program and both handlers.
- A nesting depth of at most 4, where a statement's depth is the number of enclosing `{ }` blocks, including a handler block.

#### Scenario: Too many statements
- **WHEN** a script contains 41 `punch` statements
- **THEN** validation fails with a message stating the script has 41 statements and the maximum is 40, suggesting `repeat` or `loop` to shorten it

#### Scenario: Nesting at the limit is allowed
- **WHEN** an action is nested inside exactly 4 blocks, e.g. `loop { if distance < 10 { repeat 2 { if enemy.ducking { sweep } } } }`
- **THEN** validation succeeds

#### Scenario: Nesting too deep
- **WHEN** an action is nested inside 5 blocks
- **THEN** validation fails at the innermost block's opening position with a message stating the maximum nesting depth is 4

#### Scenario: Oversized source
- **WHEN** a script source is 4097 bytes
- **THEN** validation fails with a message stating the size and the 4096-byte limit, without attempting to parse further

### Requirement: Validation error format
Every validation error SHALL include a 1-based line, a 1-based column (counted in bytes) and a human-readable message. Where a likely fix exists, the message SHALL include a concrete suggestion. Unknown identifiers SHALL be checked against the known vocabulary, and the closest match by edit distance (distance ≤ 2) SHALL be suggested. The validator SHALL report all errors it can find, up to 10, sorted by position. Its text rendering SHALL be one error per line in the form `line:col: message`.

#### Scenario: Misspelled action with suggestion
- **WHEN** line 3, column 5 of a script contains `kik`
- **THEN** the validator reports `3:5: unknown action "kik" — did you mean "kick"?`

#### Scenario: Multiple errors reported together
- **WHEN** a script contains an unknown action on line 2 and a missing `}` at end of file
- **THEN** both errors are returned, ordered by position

#### Scenario: Valid script yields no errors
- **WHEN** a valid script is validated
- **THEN** the validator returns an empty error list
