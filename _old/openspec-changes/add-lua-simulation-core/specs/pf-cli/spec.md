# Spec Delta

## Purpose

Defines the `pf` command-line tool used to validate scripts, simulate a round between two scripts, and run the round-robin balance harness over the sample scripts.

## ADDED Requirements

### Requirement: pf validate
`pf validate <file>...` SHALL validate each file. For each invalid file it SHALL print its errors to stdout, one per line, as `<file>:<line>:<col>: <message>` (or `<file>:<line>: <message>` when the column is unknown). For each valid file it SHALL print `<file>: ok`. The exit code SHALL be 0 when all files are valid, 1 when any is invalid, and 2 on usage or I/O errors.

#### Scenario: Valid file
- **WHEN** `pf validate scripts/turtle.lua` is run
- **THEN** it prints `scripts/turtle.lua: ok` and exits 0

#### Scenario: Invalid file
- **WHEN** `pf validate bad.lua` is run on a file with `kik()` on line 3
- **THEN** it prints `bad.lua:3: unknown name "kik" — did you mean "kick"?` and exits 1

#### Scenario: Missing file
- **WHEN** `pf validate nope.lua` is run and the file does not exist
- **THEN** it prints an error to stderr and exits 2

### Requirement: pf sim
`pf sim <a.lua> <b.lua> [--json] [--frames N] [-v]` SHALL simulate a round with `a` as p1 and `b` as p2. By default it SHALL print a human-readable text log to stdout: one line per event (excluding `check` events unless `-v` is given) in the form `[f NNN] <actor> <description>`. Script `print` output SHALL also be shown only with `-v`, while `script_error` events SHALL always be shown. The event lines SHALL be followed by a result summary line with winner, reason, end frame and final HP. With `--json`, it SHALL print the exact event log JSON. `--frames` SHALL set the frame budget (default from config, must be ≥ 1). If either script is invalid, it SHALL print validation errors as `pf validate` does and exit 1.

#### Scenario: Text output
- **WHEN** `pf sim scripts/rushdown.lua scripts/turtle.lua` is run
- **THEN** stdout contains event lines such as `[f 045] p1 kick blocked by p2` and ends with a result line such as `result: p2 wins by timeout at f599 (hp 71-88)`

#### Scenario: JSON output is the log
- **WHEN** `pf sim a.lua b.lua --json` is run twice
- **THEN** both outputs are byte-identical and parse as the event log JSON

#### Scenario: Custom budget
- **WHEN** `pf sim a.lua b.lua --frames 120 --json` is run and no KO occurs
- **THEN** the log has 120 frames and `frame_budget` 120

### Requirement: pf selfplay
`pf selfplay <dir>` SHALL load every `*.lua` file in the directory, sorted by file name, and simulate every ordered pair of distinct scripts (each pair from both sides). It SHALL print a matrix where the cell for row A, column B shows A's record against B as `W-D-L` across both side assignments, with `—` on the diagonal and a totals column. Before the matrix it SHALL print any script that fails validation and exclude it. Output SHALL be deterministic.

#### Scenario: Matrix over samples
- **WHEN** `pf selfplay scripts/` is run with the five sample scripts
- **THEN** it prints a 5×5 matrix in which each off-diagonal cell's W+D+L equals 2, and cell (A,B) mirrors cell (B,A) with W and L swapped

#### Scenario: Invalid sample excluded
- **WHEN** the directory contains an invalid `.lua` file
- **THEN** its validation errors are printed and it is excluded from the matrix

### Requirement: Runtime errors are visible in sim output
`pf sim` SHALL exit 0 when the round completes, even if a script raised a runtime error. The text log SHALL show each `script_error` as `[f NNN] <actor> script error at line L: <message>`.

#### Scenario: Script error in text output
- **WHEN** `pf sim` runs a script that errors with `attempt to perform arithmetic on a nil value` on line 4 at frame 30
- **THEN** stdout contains `[f 030] p1 script error at line 4: attempt to perform arithmetic on a nil value` and the command exits 0
