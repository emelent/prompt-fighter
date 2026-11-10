# Proposal

## Why

Prompt Fighter is a fighting game where players write a **fight script** (with whatever LLM they like) instead of using a controller. The first attempt (`_old/openspec-changes/add-simulation-core`) invented a custom `.pf` language. That meant writing a lexer, a parser, a type checker and an interpreter, and asking every LLM to learn a language it has never seen. Lua is small, sandboxable and well known to every LLM. With an embedded Lua runtime, LLMs write working scripts on the first try, and the engine work goes into combat rules instead of language tooling.

This change replaces that earlier change. Its scope is the same, **Phase 1 of the build spec only** (the Go simulation core). Phases 2–5 (HTTP, MCP, replay viewer, submission UI) remain separate later changes.

## What Changes

- Fight scripts are **Lua 5.1 source files (`.lua`)** run by the embedded pure-Go runtime [gopher-lua](https://github.com/yuin/gopher-lua) in a deterministic sandbox. The custom `.pf` DSL is dropped. (The old code is kept under `_old/` for reference and is not part of the build.)
- **Blocking-call script model.** A script defines `function main()` and may define `function on_hit()` and `function on_block()`. Actions (`punch()`, `move_forward(3)`, `high_block(6)`, …) and sensors (`distance()`, `enemy.airborne()`, `in_range("sweep")`, …) are Go functions that suspend the fighter's coroutine and cost frames. Ordinary Lua code between calls is free, but it is bounded by a deterministic per-frame instruction budget.
- Frame costs, the 2-frame sensor reads with stale snapshots, stances and overlays, and the hit/block interrupt semantics all carry over from the old design. "Resume at the next statement" now simply means **the interrupted call returns `"interrupted"`** after the handler has run.
- A static checker over the gopher-lua AST catches unknown names (with "did you mean"), calls with the wrong arguments, sensors used without `()`, a missing `main`, and syntax that LLMs commonly carry over from other languages (`!=`, `&&`, `elif`, `{ }` blocks). Each error comes with a hint the LLM can act on.
- New Go module at the repo root:
  - `/sim/script`: Lua sandbox setup, static checker, API bindings and coroutine driver.
  - `/sim`: config table, fighter state, frame simulation and rules, and the event log.
  - `/cmd/pf`: `validate`, `sim` and `selfplay` (unchanged behavior, `.lua` files).
  - `/scripts`: `turtle.lua`, `rushdown.lua`, `jumper.lua`, `thrower.lua`, `lazy.lua`.
  - `/docs/RULES.md`: engine rules reference, including the Lua API reference.
- The event log keeps the old schema, plus two new event types: `script_error` (a runtime error or exhausted budget, after which the fighter idles) and `print` (`print()` output, kept in the log for debugging).

## Capabilities

### New Capabilities
- `fight-script-api`: the Lua script contract. Covers the required and optional entry functions, the action and sensor API with argument rules, the sandboxed standard library, the static limits, and the static checker with its error format and hints.
- `script-execution`: how a script's coroutine turns into per-frame fighter intent. Covers frame costs, the instruction budget, sensor snapshots, stances and overlays, interrupts and handlers (the `"interrupted"` return), runtime errors and idling.
- `combat-rules`: config table, geometry, frame data, and hit, block, throw, trade, combo, knockdown, KO and timeout resolution (carried over from the old change).
- `replay-log`: event log JSON contract, determinism and side symmetry (carried over, plus the `script_error` and `print` events).
- `pf-cli`: `pf validate`, `pf sim` and `pf selfplay` over `.lua` files.
- `sample-scripts`: the five sample bots and the requirement that every bot beats at least one other and loses to at least one other.

### Modified Capabilities
- None. `openspec/specs/` is empty, and the old change was never archived.

## Impact

- New code: `go.mod`, `/sim`, `/sim/script`, `/cmd/pf`, `/scripts`, `/docs/RULES.md`, `/sim/testdata`.
- **Dependency:** `github.com/yuin/gopher-lua` v1.1.x. This replaces the old change's stdlib-only constraint. It is pure Go, so no cgo is needed.
- Superseded: `_old/openspec-changes/add-simulation-core` and the old `sim/dsl` code. These are not deleted, only moved out of the build.
- Downstream contract: the event log schema and the Lua API defined here are what Phases 2–4 (and the player guide) will build on.
