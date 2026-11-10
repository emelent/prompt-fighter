# Tasks

## 1. Scaffolding and config

- [ ] 1.1 Create `go.mod` (module `promptfighter`, `go 1.27`, `github.com/yuin/gopher-lua v1.1.2`) and empty packages `sim`, `sim/script` and `cmd/pf`, plus `scripts/`, `docs/` and `sim/testdata/golden/`. Verify that `go build ./...` and `go vet ./...` succeed and that `go list ./...` does not include anything under `_old/`.
- [ ] 1.2 Port `_old/sim/config.go` and `config_test.go` into `sim`. Set version `2026.11-L1` and add `InstrBudget`, `LoadInstrBudget`, `MaxSourceBytes`, `MaxPrints`, `MaxPrintBytes`, `CallStackSize` and `RegistryMaxSize` (design D2). Keep the tests for the frame data table, the overlay defaults and the version hash pin. Verify that `go test ./sim -run Config` passes.

## 2. Lua sandbox

- [ ] 2.1 Implement sandbox construction (design D7): the allowed library subset, removal of the forbidden entries, `tostring` that never shows addresses, the frozen `enemy` table, a strict `_G` whose `__index` raises an unknown-name error with a suggestion, and the `loading` flag that rejects API calls at top level. Test that every forbidden global is absent, that `tostring({})` is `"table"`, that assigning to `enemy` errors, that every allowed stdlib function is callable with the strict `_G` in place, and that `pairs` order is stable across two fresh states. Verify that `go test ./sim/script -run Sandbox` passes.
- [ ] 2.2 Implement `budgetCtx` (design D6) and install it on every thread with `SetContext`. Test that a spin loop in a coroutine fails after exactly `InstrBudget` instructions, that `pcall` cannot swallow the failure, that the counter resets on yield, and that two runs fail on the identical instruction. Verify that `go test ./sim/script -run Budget` passes.

## 3. Static checker

- [ ] 3.1 Implement the size check and syntax errors: parse with `parse.Parse`, extract line and column, and map the hint triggers (`!=`, `&&`, `||`, `!`, `elif`, braces, `//`, C-style `for`). Add table tests that pin the exact `line:col: message` for each fight-script-api syntax scenario. Verify that `go test ./sim/script -run Syntax` passes.
- [ ] 3.2 Implement the AST walk with scope tracking. It checks unknown globals (with Levenshtein suggestions and the overlay and forbidden-library hints), API arity and literal arguments, `enemy.<field>` validity and sensors used without `()`, and API calls at top level. Cap errors at 10, sorted by position. Add table tests pinning the exact message for every fight-script-api scenario, including script-defined helpers being known. Verify that `go test ./sim/script -run Check` passes.
- [ ] 3.3 Implement `Load`: static check, then the top-level chunk under `LoadInstrBudget`, then the entry-point checks (missing `main`, a handler that is not a function, an endless top-level chunk). Test each scenario. Verify that `go test ./sim/script -run Load` passes.

## 4. Coroutine driver

- [ ] 4.1 Implement API bindings that validate arguments at runtime, record a request with its source line and yield. Implement the driver's pending-request state machine and `Step(WorldView) Intent` (design D4). Cover frame costs for every action, sensor snapshot on the first frame, `check` events, return values, free computation, `script_end` and `print` (with caps). Test against a stub world, asserting per-frame intent sequences for every scenario under "Blocking calls and frame costs", "Sensor reads…" and "print". Verify that `go test ./sim/script -run Driver` passes.
- [ ] 4.2 Add stance handling: the 1-frame stance start, concurrent sensors and `wait`, air-attack and `low_jab` overlays, one air attack per jump, and held requests for non-overlay actions. Test each scenario under "Stances and overlays". Verify that `go test ./sim/script -run Stance` passes.
- [ ] 4.3 Add interrupt and handler handling (design D5): cancelling the pending request with the `"interrupted"`/`"blocked"` return, keeping the sensor snapshot as the return value, fresh handler threads, discarding a handler on re-interrupt, `handler_enter` events, and handlers after `script_end`. Add runtime-error handling (`script_error`, idling, no further handlers). Test each scenario under "Interrupts and handlers", "Runtime errors" and "Script end and idling". Verify that `go test ./sim/script -run 'Handler|Interrupt|Error'` passes.

## 5. Frame simulation and rules

- [ ] 5.1 Implement fighter state, the symmetric tick (design D3) with the driver plugged into Think, and movement: walking, jumps with the y-curve and landing recovery, edge clamping, symmetric overlap reversion and auto-facing. Reuse the logic from `_old/sim/fighter.go` and `_old/sim/sim.go` where it fits. Test edge clamp, no overlap when both move, walking into the opponent, and jump-forward distance and peak. Verify that `go test ./sim -run Move` passes.
- [ ] 5.2 Implement attack phases and connection checks, including the whiff conditions, the single-connect rule and `whiff` events with reasons. Test each whiff scenario in combat-rules. Verify that `go test ./sim -run Whiff` passes.
- [ ] 5.3 Implement simultaneous resolution: hits (pushback with wall transfer, anti-air knockdown), blocks (defender blockstun and hold extension, attacker blockstun equal to recovery), throws (unblockable, attack beats throw, tech) and trades. Feed outcomes back to the driver as attack return values and interrupts. Test each triangle edge, the trade, pushback, wall transfer and wrong-height block, and that `kick()` returns `"blocked"` when blocked. Verify that `go test ./sim -run 'Hit|Block|Throw|Trade'` passes.
- [ ] 5.4 Implement combo scaling (integer arithmetic, floor 30%, knockdown on the 5th hit), knockdown (20 invulnerable frames, wake-up into `on_hit`), and the KO, double-KO and timeout results. Test scaling with a test-only config, the combo cap, invulnerability, wake-up, early KO end, timeout draw and double KO. Verify that `go test ./sim -run 'Combo|Knockdown|Result'` passes.

## 6. Event log and docs

- [ ] 6.1 Implement the log types (header, players, result, per-frame entries, events with typed `Detail` including `message` and `text`, pointer fields for meaningful zeros), event ordering, and `Simulate(cfg, a, b)`. Test header contents, frame count and indexing, `script_line` and `anim` during sensor calls, a `false` check value serializing, and the `script_error` and `print` events. Verify that `go test ./sim -run Log` passes.
- [ ] 6.2 Write `docs/RULES.md`. It covers the frame data table, the tick order, every gap-fill decision (old D7/D8 plus this design), the full Lua API reference with return values, the sandbox contents, the instruction budget, sensor semantics and reaction windows, event `detail` fields per type, and the Phase 2 memory-isolation prerequisite. Verify by cross-checking the API list against the registered bindings and the event fields against the `Detail` struct.

## 7. CLI

- [ ] 7.1 Implement `pf validate`: output format with and without a column, and exit codes 0/1/2. Test it by calling the subcommand function with valid, invalid and missing fixture files. Verify that `go test ./cmd/pf -run Validate` passes.
- [ ] 7.2 Implement `pf sim` with the text log (`check` and `print` only with `-v`, `script_error` always shown, result summary line), `--json`, `--frames` and validation-error passthrough. Test the text output shape, JSON byte-equality with `sim.Simulate`, the `--frames 120` frame count and the script-error line with exit 0. Verify that `go test ./cmd/pf -run Sim` passes.
- [ ] 7.3 Implement `pf selfplay` over `*.lua`: sorted discovery, invalid-script reporting and exclusion, a both-sides round-robin, and the `W-D-L` matrix with a totals column. Test matrix shape and mirror consistency on a temp dir of small scripts. Verify that `go test ./cmd/pf -run Selfplay` passes.

## 8. Sample scripts and balance

- [ ] 8.1 Write `scripts/turtle.lua`, `rushdown.lua`, `jumper.lua`, `thrower.lua` and `lazy.lua` (lazy with no sensor calls), each with `--` strategy header comments. Verify that `go run ./cmd/pf validate scripts/*.lua` prints `ok` for all five.
- [ ] 8.2 Run `go run ./cmd/pf selfplay scripts/` and tune the scripts (not the config) until every script beats at least one other and loses to at least one other. Add the automated balance test and a test that `lazy.lua` emits no `check` events. Verify that `go test ./... -run Balance` passes.

## 9. Golden logs, determinism, symmetry, integration

- [ ] 9.1 Add golden-file tests with an `-update` flag for at least 4 sample pairs, including one KO and one timeout. Verify that `go test ./sim -run Golden` passes and the files exist under `sim/testdata/golden/`.
- [ ] 9.2 Add a determinism test (two in-process runs plus a built `pf sim --json` subprocess, all byte-identical, including a pair where one script hits `script_error`) and a symmetry test (every sample against itself draws with mirrored x; every swapped pair mirrors). Verify that `go test ./... -run 'Determinism|Symmetry'` passes.
- [ ] 9.3 Run `go vet ./...`, `go test ./... -count=1` and `go test -race ./sim` (with parallel `Simulate` calls), and confirm all pass. Then produce the Phase 1 review summary: gap-fill decisions, the selfplay matrix, balance observations (combo dormancy, duck vs air attacks, the block interpretation, the effect of free computation) and the memory-limit risk for Phase 2. Stop before Phase 2.
