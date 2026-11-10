# Proposal

## Why

Under the current rules, waiting, blocking and backing away are safe. Repeating one good move also works indefinitely: a block costs nothing but time, and a fighter that does nothing is never punished for it. Scripts that turtle, run away or loop a single attack therefore do well, which makes rounds dull to watch and teaches LLM players the wrong lesson. Combos also get weaker with every hit, which rewards the opposite of sustained pressure.

## What Changes

- **Guard break:** the 4th attack a fighter blocks in a row breaks its guard.
  - That attack deals no damage, but the defender is stunned for 20 frames and cannot block. The attacker is not put in blockstun, so it has time to punish.
  - The streak resets when the blocker starts an attack, goes 60 frames without blocking, is hit, or has its guard broken.
- **Passivity boost:** a fighter that goes 120 frames without starting an attack becomes passive.
  - Frames spent moving away from the opponent count double. Frames spent stunned or knocked down don't count.
  - Once passive, the opponent's next hit or throw deals extra damage: ×1.1 at 120 frames, rising by 0.1 every 30 frames to ×1.5 at 240 frames.
  - The passive counter resets when the fighter starts an attack, or when a boosted hit or throw lands on it.
- **Stale attacks:** starting the same attack 6 or more times in a row shrinks its reach, by 15% of the base range per repeat past the 5th, down to a floor of 40%.
  - Range drops to 85% on the 6th repeat, 70% on the 7th, 55% on the 8th and 40% from the 9th.
  - This is deterministic: there is no randomness. An attack that misses only because of the shrunken reach whiffs with the new reason `stale`.
  - Starting a different attack resets the count. Air attacks and `low_jab` count as their own attacks.
- **Combos grow:** combos now add 15% per hit (×1.0, ×1.15, ×1.3, ×1.45, ×1.6), replacing the old scaling of ×0.8 per hit with a 30% floor. The 5th hit still knocks down. **BREAKING** for scripts and balance that relied on decaying combos.
- **New log content:**
  - events `guard_break` and `stale`
  - a `boost` field on `hit` and `throw` events when the boost is above 100%
  - frame `state` and `anim` value `guard_break`
  - whiff reason `stale`
  - `pf sim` lines for the new events

  The log format stays `"1"`, because the additions are new values and fields only.
- **New sensors**, so scripts can react to the new rules:
  - `guard`: blocks in a row so far, 0–3.
  - `passive` and `enemy.passive`: frames of passivity, capped at 999.
  - `repeats`: how many times in a row the fighter has started its last attack.
- **Config:** new tunables for all of the above, `ComboScalePct`/`ComboFloorPct` replaced by `ComboGrowthPct`, and the version bumped to `2026.11-3`.
- **Updated to match:** golden logs, text snapshots, and the sample bots, retuned so the non-transitive balance test still passes.
- **Replay viewer:**
  - a guard-break pose
  - new callouts: `GUARD BREAK!` above the defender, `×1.3`-style boost callouts above the target of a boosted hit, and `STALE` above a fighter starting a stale attack
  - all callouts now fade out over their lifetime instead of disappearing
- **Docs:** `docs/RULES.md` and `docs/PLAYER_GUIDE.md` describe the new rules, sensors and strategy, and the guide's version becomes `2026.11-3`.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `combat-rules`:
  - "Blocks" now covers guard break.
  - New requirements: "Guard break", "Passivity boost", "Stale attacks" and "Combo growth".
  - "Combo scaling" is removed.
  - "Versioned config table" gets new defaults and a version bump.
- `replay-log`: "Per-frame state" adds the `guard_break` state and anim. "Events" adds `guard_break`, `stale`, the `boost` field and the `stale` whiff reason.
- `fight-script-language`: "Conditions" adds the sensors `guard`, `passive`, `enemy.passive` and `repeats`.
- `script-execution`: "Sensor semantics" defines the new sensors. In "Interrupts and handlers", a guard break interrupts like a hit and runs `on_hit`.
- `replay-viewer`: "Stage rendering" adds the guard-break pose. "Event callouts" adds the new callouts and the fade.
- `player-guide`: "Guide content" covers the new rules and names config `2026.11-3`.

## Impact

- **Engine (`sim/`):**
  - `config.go`: new fields and version.
  - `fighter.go`: block streak, passive counter, repeat counter, the guard-break stun and frame state.
  - `sim.go`: resolve guard break, the boost and combo growth, and shrink stale reach in `connect`.
  - `textlog.go`: lines for the new events.
  - `sim/dsl`: the new sensors.
- **Regenerated:** golden logs (`sim/testdata/golden`), the `pf sim` text snapshots (`cmd/pf/testdata/text`), and the pinned config hash.
- **Samples:** `scripts/*.pf` are retuned, especially `turtle.pf` (heavy blocking) and `lazy.pf` (idling), so that `pf selfplay` stays non-transitive.
- **Web:**
  - `web/src/replay.ts`: the event-line port.
  - `web/src/callouts.ts`, `web/src/stage.ts` and `web/src/fighterart.ts`: the new pose and callouts.
  - `web/src/types.ts`.
  - `web/dist` is rebuilt.
- **MCP:** no tool changes. The tools return the new events and sensors, and the guide text they serve is updated.
- **Players:** existing scripts still validate, but fights play out differently. Turtle and idle strategies are weaker, and long combos are stronger.
