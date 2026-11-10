# Design

## Context

The repo currently holds only the `openspec/` scaffolding. The superseded DSL attempt lives under `_old/` and is excluded from the build: it has its own `go.mod`, which makes it a separate nested module, and the Go tool skips `_`-prefixed directories in `./...` patterns. `_old/sim/config.go` and its test, plus the rules decisions in `_old/openspec-changes/add-simulation-core/design.md` (D2, D3, D4, D7, D8), are still valid and are ported, not reinvented. The toolchain is Go 1.27.3.

Constraints carried over: no randomness, no floats in game logic, no dependence on map order, a symmetric tick, table-driven tests and golden logs. New constraint: one external dependency, `github.com/yuin/gopher-lua` v1.1.2 (pure Go, Lua 5.1).

The design rests on two gopher-lua facts, verified with a throwaway prototype on 2026-11-10:
1. `mainLoopWithContext` checks `ctx.Done()` once per VM instruction. A custom `context.Context` whose `Done()` counts its calls therefore gives an exact, deterministic instruction budget. Once the count passes the limit it keeps returning a closed channel, so the error is raised again on the next instruction after any `pcall`. The prototype confirmed that `pcall(function() while true do end end); punch()` fails at the `punch` line.
2. `LTable` iteration (`next`/`pairs`) uses an insertion-ordered `keys` slice, not Go map order, so it is deterministic.

## Goals / Non-Goals

**Goals:**
- A pure entry point `sim.Simulate(cfg, a, b Script) (*Log, error)` with no I/O and no global state. Each call creates its own two `LState`s, so concurrent calls are safe.
- Script semantics that read top to bottom like the old DSL (blocking calls) and keep its frame-cost economy.
- Validation errors that let an LLM fix a script after one paste.
- The same event log schema as the old design, plus two event types.

**Non-Goals:**
- Memory limits for scripts. gopher-lua has no allocation hook. Phase 1 is local and CLI-only (see Risks).
- Lua 5.4 features. Scripts are Lua 5.1, which is what gopher-lua implements.
- Match orchestration, HTTP, MCP and persistence (later phases).

## Decisions

### D1. Package layout
- `go.mod`: module `promptfighter`, `go 1.27`, requiring `github.com/yuin/gopher-lua v1.1.2`.
- `sim/script`: everything Lua. This covers sandbox construction, the static checker (over `github.com/yuin/gopher-lua/ast` from `parse.Parse`), syntax-error hint mapping, the API bindings and the coroutine driver. It exposes a small interface to `sim`:
  - `Load(cfg, name, src) (*Program, []Error)`
  - `(*Program).Step(view WorldView) Intent`, called once per actionable frame
  - `Interrupt(kind)` and `Resume()`
- `sim`: config, fighter state, tick and resolution, and the log. `sim` depends on `sim/script`, never the reverse. Shared value types (`ActionKind`, `Intent`, `SensorKind`) live in `sim/script` or a tiny `sim/types` if cycles appear.
- `cmd/pf`: a thin `flag`-based CLI. Each subcommand is a `func(args, stdout, stderr) int`.

Alternative considered: one `sim` package. Rejected because the Lua driver has its own lifecycle and test needs (a stub world), and keeping it separate stops gopher-lua types from leaking into the rules code.

### D2. Config table
Port `_old/sim/config.go` and its hash-pinning test. Change the version to `"2026.11-L1"` and add these tunables:

| tunable | value | why |
|---|---|---|
| `InstrBudget` | 10,000 instructions per resume | Plenty for real decision logic (hundreds of instructions); a spin loop fails in microseconds |
| `LoadInstrBudget` | 10,000 | Top-level chunk: define functions and constants only |
| `MaxSourceBytes` | 8192 | Lua is more verbose than the DSL; still bounded |
| `MaxPrints` / `MaxPrintBytes` | 100 / 200 | Keeps logs small |
| `CallStackSize` / `RegistryMaxSize` | 64 / 4096 | Bounds recursion and stack growth via `lua.Options` |

All other defaults (frame data, geometry, hold 10, sensor cost 2, count range 1–100) are unchanged from the old D2.

### D3. Tick order and geometry
Unchanged from the old D3 and D4. Each frame runs simultaneous phases Think → Advance → Move → Connect → Resolve → End check. Fighters are stored in a `[2]Fighter` array, events are merged p1 then p2, and there is no crossing. **Think** is where the Lua driver runs: each actionable fighter's `Program.Step` receives a read-only `WorldView` of the start-of-frame state.

### D4. Coroutine driver
Each fighter has one `LState` holding the sandboxed globals and a coroutine thread `co` for `main`. Every API function is a Go function that, when called from Lua:
1. Validates its arguments at runtime (integer, 1–100, valid attack name). This duplicates the static check because arguments may be computed values.
2. Records a **request** (`{kind, n, line}`) in the driver, reading the line from `L.GetStack(1)`/`GetInfo("l")`.
3. Calls `L.Yield(...)`.

The driver keeps a **pending request** with its remaining frames. On each actionable frame:
- If a request is pending and not finished, the driver advances it by one frame and emits that frame's intent (start, continue, guard, move or check). This needs no Lua.
- When a request finishes on frame f, the driver records its return value. On frame f+1 it resumes `co` with that value. The Lua code runs free under the instruction budget until the next API call yields, and that new request's first frame is frame f+1. This gives "`punch` frames 0–9, `wait` frame 10" with no gap.
- Stance starts (1 frame) and sensor reads (`cost` frames, value snapshotted from the `WorldView` on frame 1) are requests like any other.
- If the stance rules say a request cannot run yet (a non-overlay action during a stance), the driver holds it. Each held frame emits the stance-continue intent.

`Resume` returning `ResumeOK` means `main` finished, and the driver emits `script_end`. `ResumeError` means a script error, and the driver emits `script_error` and marks the fighter dead.

Alternative considered: run each script in its own goroutine and use channels as the yield mechanism. Rejected because goroutines bring scheduling nondeterminism risk and leak risk, and gopher-lua coroutines already provide exactly the suspend/resume we need.

### D5. Interrupts and handlers
When the rules interrupt a fighter (hit or own attack blocked):
- If `main` is the active thread, the pending request is cancelled and its return value is set to `"interrupted"`, or `"blocked"` for the on-block case. The interrupted sensor's value is its snapshot. `co` stays suspended at its `Yield` and is untouched.
- If a handler is the active thread, that handler thread is discarded (the GC frees it; it is never resumed). `main`'s saved return value is kept unchanged.
- When the fighter becomes actionable, if the matching handler exists, the driver creates a fresh thread `h` for it, emits `handler_enter` and drives `h` exactly like `main`. When `h` returns, the driver resumes `co` with the saved return value. If there is no handler, it resumes `co` immediately.

This makes "resume at the statement after" structural by construction. It is the same continuation the DSL design built by hand; here it is the Lua call stack.

Handler-entry ordering at wake-up from knockdown follows the old D8: wake-up counts as leaving the interrupt, so `on_hit` runs.

### D6. Instruction budget
A `budgetCtx` type implements `context.Context`. `Done()` increments a counter and returns an always-open channel until the limit, then a pre-closed channel. `Err()` returns a sentinel `errBudget`. It is installed with `thread.SetContext` on every thread the driver creates. `NewThread` would otherwise wrap the parent context in `WithCancel`, which bypasses per-instruction counting. The counter resets whenever an API call yields. When `Resume` returns an error, the driver checks `ctx.exhausted` (not the message text) to classify it as a budget error.

Alternative considered: an auto-yield that charges 1 idle frame per N instructions, mirroring the old zero-cost loop guard. Rejected because gopher-lua cannot yield from inside the context check. Failing loudly is also clearer for script authors.

### D7. Sandbox
Build the environment with `lua.Options{SkipOpenLibs: true, CallStackSize, RegistryMaxSize}`. Open `base`, `table`, `string` and `math`, then delete the forbidden entries listed in the fight-script-api spec. Replace `tostring` with a wrapper that maps table, function, userdata and thread values to their type name, and route `print` through the same wrapper. Install the API functions and a frozen `enemy` table (a metatable with `__newindex` that errors). Finally, set a metatable on `_G` whose `__index` raises `unknown name "x" — did you mean …` at runtime. Script globals assigned with plain assignment still work, because `__newindex` is not overridden on `_G`. The load instruction budget applies while the chunk runs. During loading, a `loading` flag makes every API function raise the "only inside main/on_hit/on_block" error.

Floating point: Lua numbers are float64, but only the script's own decisions use them. Every value crossing into the engine is validated as an integer, and engine state stays `int`. gopher-lua performs each arithmetic operation as a separate VM instruction, so there is no cross-instruction FMA fusion. The determinism tests compare the logs byte for byte.

### D8. Static checker
1. Check size first (fail fast).
2. `parse.Parse`. On a syntax error, extract the line and column from gopher-lua's error (`line:N(column:M) near 'tok'`). Then scan the source line for hint triggers in this order: `!=`, `&&`, `||`, a `!` prefix, `elif`, `{` after `then`, `do` or `)`, `//`, and C-style `for (`. Attach the first hint that matches. Report only that one error.
3. Walk the AST, tracking local scopes, and collect:
   - Global reads. These are checked against the known set: API, sandbox library, and every global assigned anywhere in the script, including `function name()` declarations. Unknown names get Levenshtein suggestions (distance ≤ 2, ties broken by table order). Special-cased hints cover `jump_kick`, `dive_punch` and `low_jab` (overlays), and `os`, `io`, `require` and similar (determinism).
   - Calls to API names. These are checked for arity and literal arguments (count range, `in_range` literal string or identifier-named attack).
   - `enemy.<field>` accesses. A field that is not a sensor is unknown and gets a suggestion. A sensor field that is not immediately called gets the "call it with `()`" hint.
   - API calls at chunk top level, which are not allowed.
4. Run the top-level chunk in the sandbox under the load budget. Then check that `main` is a function, and that `on_hit` and `on_block` are nil or functions.

The static checker is advisory-complete, not sound: Lua is dynamic, so the runtime checks from D4 and D7 remain the backstop. All errors are `Error{Line, Col, Msg}`. They are capped at 10 and sorted.

### D9. Event log
Unchanged from the old D9: typed structs, one `Detail` struct with `omitempty` fields and pointer fields for meaningful zeros, compact `json.Marshal`, and indented golden files. New `Detail` fields: `message` (script_error) and `text` (print). `script_line` is the line of the pending API call on the active thread. When a sensor read is snapshotted, the `check` event's `sensor` value is the call name, e.g. `in_range:sweep` or `enemy.airborne`.

### D10. Tests
- `sim/script`: table tests over the static checker that pin exact messages. Sandbox tests cover forbidden globals, `tostring` without addresses, and the frozen `enemy` table. Driver tests use a stub `WorldView` and assert the per-frame intent sequence. They cover frame costs, sensor snapshot and short-circuit, budget exhaustion, pcall-proofing, handlers, interrupts during reads and during handlers, runtime errors and print.
- `sim`: scenario tests for every rules scenario (a port of the old test plan), golden logs for at least 4 sample pairs, determinism (in-process twice plus a subprocess `pf sim --json`), symmetry, and balance through selfplay.
- `go test -race ./sim` with parallel `Simulate` calls to prove there is no shared state. This exercises gopher-lua's per-`LState` isolation.

## Risks / Trade-offs

- [No memory cap: `s = s .. s` doubling inside one budget can allocate gigabytes] → Phase 1 is local only. `string.rep` is removed and `RegistryMaxSize` bounds stack growth. Phase 2 must run simulations in a resource-limited worker (rlimit or cgroup) before accepting untrusted scripts. This is recorded in RULES.md as a Phase 2 prerequisite.
- [The static checker misses dynamic mistakes, such as a computed bad `in_range` name or `_G["kik"]()`] → The runtime checks raise `script_error` with a line number, and the error is visible in the `pf sim` output.
- [Free computation makes reactive scripts cheaper to write than in the DSL, but sensors still cost 2 frames] → This is intended. The cost economy is in sensors and actions, not in thinking. Selfplay will show whether caching readings changes balance.
- [gopher-lua upgrade changes VM instruction counts, so budget failure frames shift] → Pin the version in `go.mod`. The golden logs catch any drift. A budget change counts as a config change and gets a version bump.
- [gopher-lua's `_G` `__index` metatable may interact with `parse`-time global resolution or the stdlib's own lookups] → A sandbox test covers the stdlib functions being callable with the strict `_G` in place.
- [Carried over from the old change: combos are dormant at the defaults, duck may hard-counter air attacks, and the D7 block interpretation] → Same mitigations: a test-only config for combos, selfplay to expose balance problems, and flagging these in the Phase 1 review.

## Migration Plan

None (greenfield). The `_old/` directory can be deleted once this change is archived.
