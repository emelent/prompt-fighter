# Proposal

## Why

Prompt Fighter is a fighting game where players write a **fight script** (with whatever LLM they like) instead of using a controller. Everything else depends on one piece: a deterministic engine that parses two scripts, runs them against each other frame by frame, and outputs an event log. The HTTP server, MCP server, replay viewer and submission UI (build-spec Phases 2–5) all consume that engine or its log. So the engine comes first, and it has to be stable and well-tested before anything is built on it.

This change covers **Phase 1 of the build spec only** (simulation core in Go). Phases 2–5 will each be their own change. They are out of scope here.

## What Changes

- New Go module at the repo root (stdlib-only) with:
  - `/sim/dsl`: lexer, parser and validator for `.pf` fight scripts. It enforces the size, statement and nesting limits, and its errors carry `line:col` and a hint written so an LLM can act on it.
  - `/sim`: a single versioned config table holding every tunable; the script interpreter (frame-cost accounting, sensor snapshots, stances and overlays, `on_hit`/`on_block` handlers, resume after a hitstun interrupt); and the frame simulation (movement, hit/block/throw/trade resolution, combo scaling, knockdown, KO and timeout).
  - Event log: a JSON document with a header (format version, config version, frame budget, both scripts, result), a complete per-frame state for each fighter, and a typed event list. Output is byte-identical for identical inputs.
  - `/cmd/pf`: the CLI, with `validate`, `sim` (text log, or JSON with `--json`, and `--frames`) and `selfplay` (a round-robin win/draw matrix used as the balance harness).
  - `/scripts`: sample bots `turtle.pf`, `rushdown.pf`, `jumper.pf`, `thrower.pf` and `lazy.pf`. They double as test fixtures and as future dummy opponents. Each one beats at least one other and loses to at least one other.
  - `/docs/RULES.md`: the engine rules reference, including every decision made where the build spec was silent. It feeds the Phase 3 `PLAYER_GUIDE.md`.
- Tests: table tests for the parser and validator (including the exact error messages); interpreter and rules tests; golden event-log files; and tests showing determinism (run twice, compare bytes) and side symmetry (swap players ⇒ mirrored result).

## Capabilities

### New Capabilities
- `fight-script-language`: the `.pf` DSL. Covers syntax, the action and sensor vocabulary, the type rules for conditions, the static limits, and the format and quality of validation errors.
- `script-execution`: how a validated script turns into per-frame fighter intent over time. Covers frame costs, the 2-frame sensor reads with stale snapshots, stances and overlays, handlers, interrupt and resume semantics, and idling once a script ends.
- `combat-rules`: stage, round and fighter rules. Covers the versioned config table, action frame data, movement and collision, and the hit, block, throw, trade, combo-scaling, knockdown, KO and timeout resolution.
- `replay-log`: the event log JSON contract (header, frames, events), byte-level determinism, and player-side symmetry.
- `pf-cli`: the `pf validate`, `pf sim` and `pf selfplay` commands and what they output.
- `sample-scripts`: the bundled sample bots and the balance property they must satisfy against each other.

### Modified Capabilities
- None. This is a greenfield project with no existing specs.

## Impact

- New code: `go.mod`, `/sim`, `/sim/dsl`, `/cmd/pf`, `/scripts`, `/docs/RULES.md`, test data under `/sim/testdata`.
- Dependencies: Go standard library only (Go 1.27, the current local toolchain).
- Downstream contract: the event log schema and config versioning defined here are what Phases 2–4 will consume. Changing them later means bumping the format or config version.
- Not affected or not included: HTTP server, MCP server, `/web`, match (best-of-3) orchestration, and `PLAYER_GUIDE.md`/`API.md`. Those belong to later phases.
