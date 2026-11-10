# Tasks

## 1. Scaffolding and config

- [x] 1.1 Create `go.mod` (module `promptfighter`, `go 1.27`) and empty packages `sim`, `sim/dsl` and `cmd/pf` with the directories `scripts/`, `docs/` and `sim/testdata/golden/`. Verify that `go build ./...` and `go vet ./...` succeed.
- [x] 1.2 Implement `sim.Config`, `DefaultConfig()` with every default from the combat-rules spec and design D2, the ordered `ActionKind` list, and the attack-name vocabulary used by `in_range`. Add tests asserting the build-spec frame data table and the overlay defaults, plus a test pinning a hash of the encoded default config to `Version`. Verify that `go test ./sim -run Config` passes.

## 2. DSL: lexer, parser, validator

- [x] 2.1 Implement the lexer: identifiers, dotted sensors, `xN` count tokens, integers, `{ } ( )`, comparison operators, `#` comments, and 1-based byte line:col tracking. Add table tests for token streams and positions, including the `x3` vs identifier ambiguity and errors for `&&`, `||`, `!` and `;` with their hints. Verify that `go test ./sim/dsl -run Lex` passes.
- [x] 2.2 Implement the AST and the recursive-descent parser: statements, action argument forms, `if/elif/else`, `repeat`, `loop`, top-level handlers, and expression precedence (comparison > `not` > `and` > `or`, plus parentheses). Recover at statement boundaries and cap errors at 10. Add table tests for parse shapes and recovery with multiple errors. Verify that `go test ./sim/dsl -run Parse` passes.
- [x] 2.3 Implement the validator: type checking of conditions, action argument rules and count ranges, overlay-name rejection, empty `loop`, `loop` in a handler, duplicate or nested handlers, the 4096-byte, 40-statement and depth-4 limits, Levenshtein suggestions (≤2), and the hand-written hints from design D10. Implement `Error.String()` as `line:col: message`. Add table tests pinning the exact message for every scenario in the fight-script-language spec. Verify that `go test ./sim/dsl` passes.

## 3. Interpreter

- [x] 3.1 Compile conditions into postfix programs with short-circuit jumps and per-sensor cost. Implement the continuation-stack interpreter (design D5) that yields one per-frame intent from a read-only state snapshot, including 2-frame sensor reads with first-frame snapshots, `check` event emission, and the zero-cost loop guard. Test against a stub world: frame costs for attacks, moves, wait and blocks, short-circuit cost, stale snapshot, and literal-only conditions. Verify that `go test ./sim -run Interp` passes.
- [x] 3.2 Add stance handling (design D6): the 1-frame stance start, concurrent checks and wait, air-attack and low_jab overlays, one air attack per jump, and non-overlay actions waiting for the stance to end. Test each stance and overlay scenario in the script-execution spec. Verify that `go test ./sim -run Stance` passes.
- [x] 3.3 Add interrupt and resume: freezing on stun, structural "next statement" resume (inside blocks, during conditions, at the end of a loop or repeat), the `on_hit`/`on_block` handler stacks, handler re-entry that keeps the original resume point, `handler_enter` and a single `script_end`. Test every interrupt and handler scenario in the script-execution spec. Verify that `go test ./sim -run Handler` passes.

## 4. Frame simulation and rules

- [x] 4.1 Implement fighter state, the symmetric tick order (design D3) and movement: walking, jumps with y-curve and landing recovery, edge clamping, symmetric overlap reversion, and auto-facing. Add tests for edge clamp, no-overlap with both fighters moving, walking into the opponent, and jump-forward distance and peak. Verify that `go test ./sim -run Move` passes.
- [ ] 4.2 Implement attack phases and connection checks: range, ducking/airborne/invulnerable/stunned whiff conditions, the single-connect rule, and `whiff` events with reasons. Add a test for each whiff scenario in the combat-rules spec. Verify that `go test ./sim -run Whiff` passes.
- [ ] 4.3 Implement simultaneous resolution: hits (damage, hitstun, pushback with wall transfer, anti-air knockdown), blocks (defender blockstun and hold extension, attacker blockstun = recovery, design D7), throws (unblockable, knockdown, attack-beats-throw, tech) and trades. Add tests for each triangle edge, the trade, pushback, wall transfer and wrong-height block. Verify that `go test ./sim -run 'Hit|Block|Throw|Trade'` passes.
- [ ] 4.4 Implement combo counting and scaling (integer, floor 30%, knockdown on the 5th hit), knockdown (20f invulnerable, wake-up into `on_hit`), KO, double KO and timeout results. Test the scaling sequence with a test-only config (long hitstun, zero pushback), the combo knockdown cap, knockdown invulnerability, wake-up, early KO end, timeout draw and double KO. Verify that `go test ./sim -run 'Combo|Knockdown|Result'` passes.

## 5. Event log

- [ ] 5.1 Implement the log types (header, players, result, per-frame fighter entries with all replay-log fields, events with typed `Detail` using pointer fields for meaningful zeros), event ordering, and the `Simulate(cfg, a, b) (*Log, error)` entry point. Test header contents, frame count and indexing, `script_line`/`anim` during checks, and that a `false` check value serializes. Verify that `go test ./sim -run Log` passes.
- [ ] 5.2 Write `docs/RULES.md`: the frame data table, tick order, every gap-fill decision from design D2/D7/D8, sensor semantics and reaction windows, and the event `detail` fields per type. Verify that every event type and detail field emitted by the code is listed, by cross-checking against the `Detail` struct.

## 6. CLI

- [ ] 6.1 Implement `pf validate` (output format and exit codes 0/1/2). Test it with `testing` by calling the subcommand function with fixture files: valid, invalid, and missing. Verify that `go test ./cmd/pf -run Validate` passes.
- [ ] 6.2 Implement `pf sim` with the text log (check events only with `-v`, result summary line), `--json` and `--frames`, plus validation-error passthrough. Test text output shape, JSON byte-equality with `sim.Simulate`, and the `--frames 120` frame count. Verify that `go test ./cmd/pf -run Sim` passes.
- [ ] 6.3 Implement `pf selfplay`: sorted file discovery, invalid-script reporting and exclusion, both-sides round-robin, and a `W-D-L` matrix with diagonal `—` and a totals column. Test matrix shape and mirror consistency on a temp dir of small scripts. Verify that `go test ./cmd/pf -run Selfplay` passes.

## 7. Sample scripts and balance

- [ ] 7.1 Write `scripts/turtle.pf`, `rushdown.pf`, `jumper.pf`, `thrower.pf` and `lazy.pf` (lazy with zero sensor reads), each with strategy header comments. Verify that `go run ./cmd/pf validate scripts/*.pf` prints `ok` for all five.
- [ ] 7.2 Run `go run ./cmd/pf selfplay scripts/` and tune the scripts (not config, per the design risks) until every script beats at least one other and loses to at least one other. Add the automated balance test from the sample-scripts spec, plus a test asserting `lazy.pf` emits no `check` events. Verify that `go test ./... -run Balance` passes.

## 8. Golden logs, determinism, symmetry

- [ ] 8.1 Add golden-file tests with an `-update` flag for at least 4 sample pairs (including a KO and a timeout result). Verify that `go test ./sim -run Golden` passes and the files exist under `sim/testdata/golden/`.
- [ ] 8.2 Add a determinism test (two in-process runs plus a built `pf sim --json` subprocess, all byte-identical) and a symmetry test (every sample vs itself draws with mirrored x; every swapped pair mirrors frames and keeps the same winning script). Verify that `go test ./... -run 'Determinism|Symmetry'` passes.

## 9. Integration check and Phase 1 review

- [ ] 9.1 Run `go vet ./...` and `go test ./... -count=1` and confirm both pass. Run `go test -race ./sim` to confirm `Simulate` has no shared state.
- [ ] 9.2 Produce the Phase 1 review summary for the user: the design decisions where the spec was silent (from RULES.md), the `pf selfplay scripts/` matrix, and the balance observations (combo dormancy, duck vs air attacks, D7 block interpretation, anything selfplay surfaces). Stop before Phase 2.
