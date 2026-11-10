# Design

## Context

This is a greenfield repository: it holds only the `openspec/` scaffolding, with no code. The toolchain is Go 1.27. See proposal.md for the motivation and scope (Phase 1 only). The behavioral contract lives in the six spec files under `specs/`. This document covers how to build it, and records every decision made where the build spec was silent. All of those decisions are also mirrored in `docs/RULES.md` so the Phase 1 review can check them in one place.

Constraints: no randomness, no floats in game logic, no dependence on map order, stdlib only, table-driven tests, and golden logs.

## Goals / Non-Goals

**Goals:**
- A pure function `Simulate(cfg, scriptA, scriptB) → Log` with no I/O and no global state, so the HTTP and MCP layers can call it concurrently later.
- A tick order that is symmetric between players, so swapping sides mirrors the result. This is enforced by tests, not just intended.
- Validation errors good enough that an LLM can fix a script after one paste.
- A log schema stable enough for Phases 2–4 to build on without changes.

**Non-Goals:**
- Match (best-of-3) orchestration, persistence, HTTP and MCP. Those are Phases 2–3.
- Loading config from files or flags beyond `--frames`. The config is a Go value; overrides are for tests.
- Performance tuning beyond "selfplay over 5 scripts finishes in well under a second".
- Cross-ups or side switching (see D4).

## Decisions

### D1. Package layout
- `go.mod` at the root, module `promptfighter`, `go 1.27`.
- `sim/dsl`: tokens, lexer, AST, parser, validator (types, limits, suggestions), and an `Error{Line, Col, Msg}` type. It has no game knowledge beyond the vocabulary tables, which it receives from config so that limits stay configurable.
- `sim`: `Config` and `DefaultConfig()`; the fighter state; the interpreter; the tick and resolution logic; and the log types with JSON encoding.
- `cmd/pf`: a thin CLI built on `flag`. Each subcommand is a function returning an exit code, so it can be tested without `os.Exit`.

Alternative considered: separate `sim/interp`, `sim/rules` and `sim/log` packages. Rejected because the interpreter and the rules are tightly coupled through fighter state, and splitting them would mean exporting internals.

### D2. Config table
`Config` is a plain struct with `Version string` and an `Actions map[ActionKind]ActionSpec`. Only lookups touch that map, never iteration. Where code needs an ordered walk (validator vocabulary, docs), it uses a fixed `[]ActionKind` slice. The default version is `"2026.11-1"`. A test pins a hash of the JSON-encoded `DefaultConfig()` to its version string, so changing a default without bumping the version fails CI.

Defaults that the build spec left unspecified:

| tunable | value | why |
|---|---|---|
| fighter half-width | 2 (min separation 4, x ∈ [2,98]) | Lets throw range 5 actually reach; start distance 40 |
| default `hold` | 10 frames | Holds without `hold N` should still do something useful |
| count range (`xN`, `hold`, `wait`, `repeat`) | 1–100 | Prevents single-statement round burning; literals 0–999 |
| jump_kick | startup 3, active 5 | Must fit inside a 20-frame jump |
| dive_punch | startup 2, active 4, range 8 | Faster but weaker than jump_kick |
| low_jab | 3/2/5 | Mirrors punch |
| jump y-velocity | ±2/frame, peak 20 | Visual only |
| sensor cost | 2 | From spec, but kept as a tunable |

### D3. Tick order: simultaneous and symmetric
Each frame runs these phases. Each phase computes both fighters' results from the same input before any mutation:
1. **Think.** Each actionable fighter's interpreter advances one frame against a read-only snapshot of the start-of-frame state, producing at most one *intent* (start action X, continue current action, or idle). Sensor values are snapshotted here.
2. **Advance.** Action, stance and stun timers advance, and phase transitions happen (startup → active, and so on).
3. **Move.** Both fighters compute their desired x. Edges are clamped. If the result would be closer than the minimum separation, every fighter that moved toward the other this frame has its move reverted. This is symmetric by construction.
4. **Connect.** For each fighter with an active, not-yet-connected attack, compute whether it would connect against the post-move state: hit, guarded, throw, or a whiff condition.
5. **Resolve.** Apply all connections together. Attack-beats-throw and throw-tech are resolved here first, then damage, stun, knockdown and pushback (with wall transfer), and interrupts.
6. **End check.** KO detection, then the frame snapshot is recorded.

The fighters are stored in a `[2]Fighter` array and processed in index order only where order cannot matter (pure per-fighter computation). Events are appended to per-fighter buffers and merged as p1 then p2, which gives deterministic but side-ordered events. That ordering is the only asymmetry, and it does not affect game state.

Alternative considered: sequential p1-then-p2 updates. Rejected because it gives p1 a systematic edge in every tie.

### D4. Geometry: no crossing
Fighters keep their relative order forever: airborne fighters still collide. That makes facing constant in practice (still recorded per frame) and avoids cross-up rules the spec does not define. Trade-off: jumping over an opponent is impossible. This is noted for review.

### D5. Interpreter as an explicit continuation stack
The AST is walked by an explicit stack of frames `{block, index, kind(loop/repeat/if/handler), remaining}` rather than Go recursion. This makes three things straightforward:
- Interrupt and resume: the saved resume point is "the current stack with the top index advanced past the interrupted statement". This structurally implements "the statement after", including the last-statement-in-loop case.
- Multi-frame sensor evaluation: a condition is compiled to a small postfix program with short-circuit jumps, and the interpreter keeps a condition program counter (PC) plus the pending sensor's remaining cost.
- The zero-cost loop guard: each loop frame records the tick on which its current iteration began.

Handlers run on a separate stack. The saved main resume point is kept unchanged across nested handler interrupts (spec: Interrupts and handlers).

### D6. Stances run in the background
A fighter has an optional `stance` (jump or duck, with its own timer) and an optional `action` (attack, overlay attack, block, move, wait, or check). The interpreter's next statement is checked against the stance:
- Concurrent: checks, `wait`, control flow.
- Overlay: `kick`/`punch` while airborne and no air attack used yet; `punch` during the duck hold.
- Otherwise: blocked until the stance ends. The interpreter holds on that statement, and those frames show the stance animation.

### D7. Blocked attacks and the defender's blockstun
The build spec says `on_block` runs "after blockstun when *your* attack was blocked". This is read as: the attacker is interrupted and stunned. The attacker's stun length equals the blocked attack's **recovery**, which preserves "blocks beat attacks". A flat 3 frames would make blocked attacks recover *faster* than whiffs. The defender is not interrupted. It gets 3 frames of guaranteed guarding (blockstun) and keeps its hold. Resulting frame advantage on block for the defender, all defaults: punch +2, kick +6, sweep +11.

### D8. Other gap fills
| situation | decision |
|---|---|
| Throw success | Knockdown (20f). The thrower's recovery of 15 gives the thrower −5 on wake-up, so throws aren't loops |
| Throw vs throw on the same frame | Tech: no damage, both pushed back 2 |
| Attack hits an airborne target | Knockdown and y snaps to 0 (no air juggles) |
| Low attacks vs airborne | Whiff |
| Knockdown wake-up | Counts as leaving the interrupt, so `on_hit` runs |
| `enemy.attacking` | Startup and active phases only, not recovery |
| `enemy.blocking` | Guard (hold) phase and blockstun only |
| Air attack in progress at landing | Ends on landing |
| low_jab outlasting the duck hold | Hold extends until low_jab finishes |
| Stance start | Costs 1 interpreter frame; the next statement begins on the following frame |
| KO | Simulation stops after the KO frame; double KO = draw |
| Frame indexing | `frames[i]` is the state at the end of tick i, for i = 0..budget−1 |
| Pushback at a wall | The remainder transfers to the attacker |
| Statement count | Action, `if`, `elif`, `repeat` and `loop` each count 1; `else` and handlers count 0 |

### D9. Event log encoding
Typed structs with `encoding/json`. `detail` is a struct per event type with `omitempty` fields (one `Detail` struct with optional fields, so field order is fixed). No maps are serialized. The log is encoded once with `json.Marshal`, producing compact output; golden files store indented JSON for readable diffs, and the determinism test compares compact bytes.

### D10. Validator error quality
- The parser recovers at statement boundaries (the next action keyword or `}`) so that several errors can be reported, capped at 10.
- Suggestions use Levenshtein distance ≤ 2 against the action, sensor and keyword tables, with ties broken by table order.
- Hand-written hints cover the predictable LLM mistakes: `kick x2`, `wait` without N, `jump_kick` used as a statement, `health` used as a boolean, `enemy.airborne == 1`, `else if` (suggest `elif`), `while`/`for` (suggest `loop`/`repeat`), `&&`/`||`/`!` (suggest `and`/`or`/`not`), and `;` (state that it is not needed).
- The error messages are pinned by table tests, since players depend on them.

### D11. Tests
- `sim/dsl`: table tests of input → AST shape or the exact `line:col: message` lists.
- `sim`: scenario tests built with a helper `run(t, a, b, opts) *Log`, plus `frameOf(log, f).P1`, and assertions on states and events. Each rules scenario in the specs maps to a named test.
- Golden logs in `sim/testdata/golden/<a>_vs_<b>.json` for at least 4 pairs, regenerated with `go test ./sim -update`.
- Determinism: in-process double run, plus an `exec` of the built `pf sim --json` compared against the in-process result.
- Symmetry: for every sample pair, compare the swapped run against a mirrored transform of the original.
- Balance: a selfplay test asserts the sample-scripts requirement.

## Risks / Trade-offs

- [Combos are effectively dormant at the defaults: 5-frame hitstun is shorter than every attacker's recovery plus the next startup, and pushback adds distance] → Implement and test combo scaling with a test-only config (long hitstun, no pushback). Flag this in the Phase 1 review as a balance item rather than silently changing the defaults.
- [Duck makes every air attack whiff, because all air attacks are high. Duck may hard-counter jumper] → Selfplay will show it. If needed, make `dive_punch` low-hitting. That would be a config change plus a version bump, so it is decided in review.
- [Sensor cost 2 makes reactive scripts slow (`if enemy.attacking { high_block }` takes 2 frames of a 3–8 frame startup)] → That is intended by the spec. Document the reaction window in RULES.md.
- [Sample scripts may not form a cycle at first] → Tune the scripts first. Only touch config defaults if no script tuning works, and report any such change in the review.
- [The interpretation in D7 differs from the user's intent] → It is isolated to one resolution function and two config values, and called out explicitly in the review summary.
- [Typed `Detail` with `omitempty` drops legitimate zero or false values (e.g. a check reading `false`, or a final `hp: 0`)] → Use pointer fields for values where zero or false is meaningful (`value`, `hp`, `damage`). A test asserts that a check reading `false` still serializes `"value": false`.

## Migration Plan

Not applicable (greenfield). Rollback means deleting the change's code.

## Open Questions

- Whether `pf sim` should take an optional `--config` JSON override for balance experiments. This can be added later without spec changes, since it is CLI-only.
