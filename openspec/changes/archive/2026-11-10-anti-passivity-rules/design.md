# Design

## Context

- The engine (`sim/`) is deterministic and integer-only. `world.tick` runs in fixed steps: snapshot, think, move, connect, resolve, whiffs, KO, record. `connect` decides per attacker whether its active frame connects, blocks, or whiffs with a reason. `resolve` applies both outcomes at once from the pre-resolution state, which is what keeps sides symmetric.
- Combos currently use `ComboScalePct` (80) and `ComboFloorPct` (30) in `resolve`. The 5th hit knocks down (`ComboKnockdownHits`).
- Blocks: in `resolve`'s `cBlock` case, the defender's hold is extended for 3 frames of blockstun, and the attacker gets `applyBlocked` (blockstun equal to the attack's recovery), which later runs `on_block`.
- Stuns are `stunHit`, `stunBlock` and `stunKnockdown`. `fighter.frame()` maps them to `state` and `anim`, and `applyHitstun` and `applyKnockdown` call `Interrupt("on_hit")`.
- Sensors are listed in `sim/dsl/vocab.go` (`sensorNames`) and evaluated from the start-of-frame `view` snapshot.
- Golden logs are regenerated with `go test ./sim -run Golden -update`. `TestConfigHashPinned` forces a version bump. `pf sim` text snapshots in `cmd/pf/testdata/text` are regenerated from `pf sim` output.
- The viewer's `calloutsAt` is a pure function of (log, f), and `choosePose` maps state and anim to a pose, with a golden-log coverage test.

## Goals / Non-Goals

**Goals:**
- Stay deterministic and integer-only, with side symmetry and byte-identical replays.
- Make every new rule visible in the log, in `pf sim` text, in sensors and in the viewer, so players and LLMs can see why something happened.

**Non-Goals:**
- No randomness of any kind.
- No new actions or handlers.
- No change to frame data, ranges or base damage apart from the stale reach reduction.
- No change to the log format version (`"1"`) or the HTTP and MCP APIs.

## Decisions

### 1. Three per-fighter counters, updated in fixed places of the tick
`fighter` gains three counters:
- `blockStreak` and `lastBlockT`
- `passive` (frames)
- `lastAtk` and `repeats`

Where they change:
- **Start of an attack:** in `begin` for `akAttack`, including air and `low_jab` overlays. It sets `passive = 0` and `blockStreak = 0`. It sets `repeats++` if `attack == lastAtk`, else `repeats = 1`, then `lastAtk = attack`. If `repeats ≥ StaleFrom` (6), it stores the reduced range on the act and emits `stale`.
- **End of tick, after resolve:** for each fighter that was not stunned or knocked down at any point this frame and did not start an attack this frame, `passive += 1`, or `+= 2` if its x moved away from the opponent this frame by `move_back` or the `jump_back` drift. Pushback doesn't count. The value is capped at 999 when read by sensors, and internally at 10000.
- **Block streak lapse:** checked at the start of each tick: `blockStreak = 0` once `t − lastBlockT ≥ 60`.

- **Why:** each rule then depends only on the fighter's own history and its position relative to the enemy, both of which are symmetric. Because counters change at fixed points of the tick, the outcome doesn't depend on whether a fighter is p1 or p2.
- **Alternative:** deriving the counters from the event list after the fact. This was rejected: the rules feed back into resolution, so the counters have to be live state.

### 2. Guard break is a new stun kind, resolved in `resolve`
In the `cBlock` branch, `if tgt.blockStreak == GuardBreakBlocks−1`, the branch becomes a guard break:
- the defender's act is abandoned
- `stun = stunGuardBreak` for `GuardBreakStun` (20) frames
- `Interrupt("on_hit")`
- block pushback
- `blockStreak = 0`
- emit `guard_break` (actor is the attacker), with no `block` event

Otherwise it's a normal block, with `blockStreak++` and `lastBlockT = t`.

The attacker gets no `applyBlocked`: its act stays connected and runs its normal recovery. `fighter.frame()` maps `stunGuardBreak` to `state`/`anim` `guard_break`.

`guarding()` returns false during the stun, so attacks hit. `inHitstun()` is false, so a following hit starts combo 1 and throws are not refused with `stunned`. A hit during guard-break stun replaces it with hitstun or knockdown as usual, and being hit resets `blockStreak`.

- **Why:** this reuses the existing stun machinery (no input, frame state, interrupt), so guard break needs no new interpreter rules.
- **Alternative:** letting the 4th attack deal chip damage through the block. That's less readable, and turtles could still out-trade it.

### 3. The passivity boost is applied in outcome computation and consumed on apply
In `resolve`, when computing a `cHit` or `cThrow` outcome:
- `boost = boostPct(target.passive)`, where `boostPct(p) = p < 120 ? 100 : min(150, 110 + 10 × ((p − 120) / 30))`.
- `dmg = max(1, base × comboPct × boost / 10000)`, where `comboPct = 100 + ComboGrowthPct × (combo − 1)` and throws use combo 1.

On apply, if `boost > 100`, record `Detail.Boost` and set `target.passive = 0`.

Both outcomes are computed before either is applied, so in a trade each fighter's boost comes from the other's pre-resolution counter. That keeps it symmetric.

- **Why:** a single integer expression, with no floating point, and the order of the multiplications fixed.

### 4. Stale reach is fixed at attack start
The reduced range is computed once, when the attack begins, as `range × max(StaleFloorPct, 100 − StaleStepPct × (repeats − StaleFrom + 1)) / 100`, and stored on the act as `effRange`. `connect` uses `effRange`. When `distance ≤ spec.Range` but `distance > effRange`, the whiff reason is `stale`. The `in_range(a)` sensor keeps reporting the full range, so it describes the attack, not the fighter's history. `repeats` lets scripts reason about staleness.

- **Why:** a constant per attack instance is easy to explain and to show in the log (`stale.detail.range`).

### 5. Sensors read the snapshot
`view` gains `guard`, `passive` and `repeats`. The new `dsl` sensors (`guard`, `passive`, `enemy.passive`, `repeats`) are integer sensors evaluated from the start-of-frame snapshot, at the usual 2-frame cost. Validator type rules apply unchanged.

### 6. Config and versioning
New `Config` fields, with these defaults:

| field | default |
|---|---|
| `GuardBreakBlocks` | 4 |
| `GuardStreakWindow` | 60 |
| `GuardBreakStun` | 20 |
| `PassiveLimit` | 120 |
| `PassiveBoostStartPct` | 110 |
| `PassiveBoostStepPct` | 10 |
| `PassiveBoostStepFrames` | 30 |
| `PassiveBoostMaxPct` | 150 |
| `StaleFrom` | 6 |
| `StaleStepPct` | 15 |
| `StaleFloorPct` | 40 |
| `ComboGrowthPct` | 15 |

`ComboScalePct` and `ComboFloorPct` are removed. The version becomes `2026.11-3`, and the pinned hash is updated.

### 7. Text lines and the viewer
New `pf sim` lines, mirrored exactly in `web/src/replay.ts`:
- `guard_break`: `<attack> breaks <target>'s guard`
- `stale`: `<attack> is stale (x<repeats>, range <range>)`
- a boosted hit: the existing line plus ` boosted x1.3`
- a whiff: reason `stale`

In the viewer:
- `choosePose` maps `guard_break` to a new dazed pose `GB`, with arms dropped and the torso leaning back, so it differs from hitstun.
- `calloutsAt` gains the kinds `guard_break`, `boost` (text `×1.3`, from `detail.boost`; the font gets `×` already, with digits and `.`) and `stale`.
- `Callout` gets an `alpha` (`1 − age/30`) that the stage applies to each glyph Image. `K.O.` stays opaque.
- Two callouts anchored to the same fighter in one frame stack vertically.

### 8. Rebalancing the samples
The rules mainly hurt `turtle.pf`, which blocks constantly and would be guard broken, and `lazy.pf`, whose blocks are short and spaced out, so it suffers less. `rushdown.pf`'s repeated punches may go stale.

Each sample is adjusted minimally to stay true to its archetype under the new rules. For example, turtle mixes in a jab or a throw after its 3rd guard (`guard >= 3`), and rushdown alternates punch and kick. The selfplay test then must still show every sample with at least one win and one loss.

The final matchup table is recorded in the task notes, and the sample comments are updated.

## Risks / Trade-offs

- **[Rebalancing may need several iterations]** → Run `pf selfplay scripts/` after each sample edit. The balance test is the gate. If no minimal edit works, the thresholds are config, so fall back to tuning them (still within the user's chosen values first).
- **[Golden-log and snapshot churn hides regressions]** → Regenerate only after the rule tests pass. Review the diffs of 2 pairs by hand against the new rules (a guard break in `rushdown_vs_turtle`, and boosts in pairs involving `lazy`).
- **[A boosted throw combined with knockdown is very strong against passive play]** → That's intended. The cap is 150%.
- **[`in_range` no longer guarantees a hit once an attack is stale]** → It's documented in the guide, and the `repeats` sensor makes it detectable.
- **[Symmetry]** → The existing mirror and swap tests cover it. Add mirror-match tests specific to the new rules: a guard-break pair and an idle pair.

## Migration Plan

- Existing scripts still validate. Fights change, and stored match logs keep the config version they were played with.
- No data migration.
- Rollback means reverting the engine and restoring config `2026.11-2`.
