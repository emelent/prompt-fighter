# Proposal

## Why

The Phase 1 review (`docs/ISSUES.md`, Phase 1 section) raised six balance and design questions about the engine defaults, and they have to be settled before Phase 2 builds on the engine. Three issues are real gameplay problems:
- **The throw loop is inescapable.** A repeated throw is active on the target's wake-up frame, so a wake-up attack can't beat it.
- **Combos never happen.** Hitstun is shorter than any follow-up attack.
- **Ducking beats every air attack.** All air attacks are high hits, so a ducking fighter dodges them.

The other three needed an explicit decision so the current behaviour becomes intended rather than accidental. The user has now made all six decisions.

## What Changes

- **Throw loop broken by a slightly longer throw recovery.** **BREAKING (config):** throw recovery goes from 15 to 18 frames, so throw totals 24. A thrower that throws again immediately now has its second throw go active on frame 29 after a throw that went active on frame 5. That is the same frame a punch started on wake-up becomes active, and an attack beats a throw that connects on the same frame. The wake-up triangle therefore holds: attacking beats the re-throw, the re-throw still beats a wake-up block, and jumping or backing off makes it whiff.
- **Combos tuned on.** **BREAKING (config):** default hitstun goes from 5 to 13 frames and hit pushback from 3 to 2 units. Short chains now link at the defaults: punch→kick is 2 hits and punch→punch→kick is 3. Repeated punches stop at 2 hits because pushback carries the target out of range. The config version is bumped and the golden logs are regenerated.
- **Air attacks by jump timing.** **BREAKING (rules):** the airborne frame (1–20) on which `jump_kick` or `dive_punch` starts decides what it can hit:

  | start frame | zone | hits | guarded by |
  |---|---|---|---|
  | 1–6 | rising overhead | grounded targets, ducking ones included | `high_block` |
  | 7–13 | peak air-to-air | airborne targets only; whiffs vs grounded (new reason `grounded`) | — |
  | 14–16 | falling overhead | grounded targets, ducking ones included | `high_block` |
  | 17–20 | landing low | grounded targets; carries on after touchdown instead of ending | `low_block` |

  Ducking now dodges only grounded high attacks (`punch`, `kick`). Unchanged: one air attack per jump, and hitting an airborne target knocks it down.
- **Blocked-attacker stun = attack recovery: kept** (confirmed). There is no spec change; `docs/RULES.md` drops the "needs confirmation" note.
- **Punch can't be reacted to: kept** (accepted as intended). Only the docs change.
- **New sample bot `ducker.pf`.** The samples are retuned so the non-transitive balance test still passes under the new defaults.
- `docs/RULES.md` is updated. The Phase 1 section of `docs/ISSUES.md` is marked resolved, and each decision is recorded.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `combat-rules`: connection and whiff conditions (air zones, the `grounded` reason, ducking dodges only grounded high attacks); hit hitstun and pushback values; throw recovery 18 and the wake-up rule (an attack on wake-up beats a repeated throw); a combo scenario that links at the defaults.
- `script-execution`: stances and overlays. An air attack started in the landing zone continues after touchdown.
- `sample-scripts`: the required sample set gains `ducker.pf`.

## Impact

- Code:
  - `sim/config.go`: new defaults, an air-zone table, and a version bump to `2026.11-2`.
  - `sim/sim.go` and `sim/fighter.go`: zone selection at air-attack start, connection checks, and the landing carry-through.
  - Tests whose expectations depend on hitstun 5 or pushback 3, or on air attacks ending at touchdown.
- Data: every golden log under `sim/testdata/golden/` is regenerated, because frame data changes alter every round.
- Scripts: a new `scripts/ducker.pf`; existing samples may be retuned.
- Downstream: the log schema is unchanged (`format_version` stays `1`). Consumers only see a new `whiff` reason value, `grounded`, and the new `config_version`.
