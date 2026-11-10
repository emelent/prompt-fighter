# Design

## Context

This change implements the Phase 1 decisions; proposal.md covers why. The places affected in the existing engine:
- `sim/config.go`: `DefaultConfig()` holds hitstun 5, pushback 3 and throw recovery 15. `TestConfigHashPinned` ties the encoded config to `DefaultVersion` (`2026.11-1`).
- `sim/sim.go` `connect()`:
  - The whiff checks are ordered range → invulnerable → `ducked` (any high attack vs a ducking target) → `airborne` (low or throw vs an airborne target) → `stunned`.
  - Guarding uses `a.spec.Height`, so air attacks are always high.
- `sim/sim.go` `preThink()`: it ends every air attack as soon as `!f.airborne(t)`.
- `sim/fighter.go` `frame()`: the `landing` case is checked before the attack case, so a carried attack would display as `landing`.
- Tests and golden logs: several tests hard-code frame numbers derived from hitstun 5 or pushback 3, or rely on air attacks ending at touchdown.

The probing done for this change gave these results:
- With hitstun 13 and pushback 2, `punch kick` is a 2-hit combo, `punch punch kick` is 3 hits, and repeated punches cap at 2.
- At throw recovery 15, the re-throw goes active on the wake-up frame (26) and beats every wake-up option except moving away.

## Goals / Non-Goals

**Goals:**
- Each air attack gets its zone once, at start, from data in the config, so the zones can be tuned without code changes.
- Keep the change inside `connect()`, `preThink()`, `frame()` and the config. No interpreter changes.
- Keep the symmetric tick and determinism as they are, with the side-symmetry tests still green.

**Non-Goals:**
- A vertical (y) hitbox for air-to-air. "Airborne" plus horizontal range is enough.
- Air blocking. Airborne fighters still can't start a block.
- Exposing the zone in the log schema. `anim` stays `jump_kick`/`dive_punch` and `format_version` stays `1`. A viewer can work out the zone from the start frame if it ever needs to.

## Decisions

### D1. Config: an air-zone table plus the new defaults
- Add `AirZones []AirZone` with `AirZone{From, To int; Kind AirZoneKind}`. The kinds are `rising_overhead`, `air_to_air`, `falling_overhead` and `landing_low`; each encodes to JSON as its name. The table is a slice in airborne-frame order, never a map. Defaults: 1–6, 7–13, 14–16, 17–20.
- Change the defaults: `Hitstun` 5 → 13, `HitPushback` 3 → 2, and `Attacks[throw].Recovery` 15 → 18. Bump `DefaultVersion` to `2026.11-2` and re-pin the hash.
- Ignore `AttackSpec.Height` for `jump_kick` and `dive_punch` from now on, because the zone decides their height. Set it to `HeightNone` in the defaults so nobody mistakes it for the live value.
- A config test asserts that the zones cover 1..`airFrames()` contiguously, with no gaps or overlaps. This catches bad overrides.

Alternative considered: hard-coding the zones in `connect()`. Rejected because the zones are balance values like every other tunable, and they belong in the versioned table.

### D2. Zone captured at attack start
`startAttack` records `act.zone` and an effective `act.height` when the attack is an air attack. The zone is the one containing `stanceK(t)` on the start frame, which is the airborne frame number. Overhead zones map to `HeightHigh`, `landing_low` to `HeightLow`, and `air_to_air` to `HeightNone`. Ground attacks copy `spec.Height` into `act.height`. From then on, `connect()` and `guarding()` read `act.height` instead of `a.spec.Height`, so one field drives every check.

### D3. Connection rules
`connect()` keeps one ordered chain of checks, and the first one that matches becomes the whiff reason:
1. `out_of_range`
2. `invulnerable`
3. `ducked`: only for a **non-air** high attack against a target in the duck hold.
4. `grounded`: an `air_to_air` attack against a target that isn't airborne.
5. `airborne`: an overhead air attack, a low attack (including `landing_low`) or a throw against an airborne target.
6. `stunned`: a throw against a target in hitstun.

Guarding stays `e.guarding(t, act.height)`. `HeightNone` never guards, and airborne targets never hold a block anyway. Hitting an airborne target still causes knockdown and resets y to 0 (the existing `resolve` path), so air-to-air needs no new resolution code.

### D4. Landing-low carry-through
`preThink()` ends an air attack at touchdown unless `act.zone == landing_low`. A carried attack goes on through landing recovery and past the end of the stance, using its normal phases. The fighter can't do anything else, because the act is still occupying it. `Complete()` fires when the act retires. Any statement that has to wait for the stance to end already does, so the interpreter needs no change. Across the default zone and frame data, a landing-low attack always lasts at least as long as landing recovery: jump_kick from airborne frame 17 ends on frame 24, and dive_punch from 17 ends on frame 21, which is the last landing frame.

`frame()` displays an in-progress attack ahead of `landing`: `state` is `attack`, `anim` is the attack name and `phase` is its phase. That way the replay shows the strike landing rather than a landing pose. y follows the normal grounded rule (0).

### D5. Throw recovery 18
Throw recovery 18 makes the throw total 24 frames. If the first throw is active on frame k, the knockdown runs through k+20, and the target can act on k+21. A repeated throw starts on k+19 and goes active on k+24. A punch started on k+21 goes active on k+24 too, so the existing rule that an attack beats a same-frame throw resolves it in the punch's favour. A block started on k+21 is still thrown, and so is an idle target. Jumping or moving away still makes the throw whiff. No resolution code changes; only the value moves.

### D6. Tests, goldens and samples
- Update the tests whose expectations come from the old values:
  - Hitstun and pushback: `TestHitAppliesHitstunAndPushback`, `TestHitPushbackTransfersAtWall`, the frame numbers in the `handler_test.go` scenarios, and `TestComboResetsWhenActionable`. That last one now pins a short hitstun explicitly, because combos link at the defaults.
  - Air attacks: `TestStanceAirAttackEndsOnLanding`, which becomes the falling-zone and landing-zone pair from the spec.
  - Throw timing: `TestKnockdownInvulnerable`, the throw-then-punch timing.
  - Config: `TestConfigBuildSpecFrameData` and `TestConfigHashPinned`.
- Add tests for every new spec scenario: each zone, `grounded`, zone fixed at start, carry-through display and timing, the wake-up triangle (punch beats the re-throw, a block is re-thrown), and the default combo chains.
- Regenerate the golden logs with `-update` and review that the diff only reflects the new values and rules. Pairs whose result reason changes still have to satisfy the KO-and-timeout coverage check. If they don't, swap pairs.
- Add `scripts/ducker.pf`: a duck hold, `low_jab` when in range, and a sweep against approaching enemies. Retune the samples until `TestBalanceSamplesAreNonTransitive` passes for all six. Tune the scripts only, never the config, which is what the earlier balance task did too.

## Risks / Trade-offs

- [Hitstun 13 also means 13 frames of advantage after any single hit, which strengthens rushdown] → Selfplay surfaces it. If one sample dominates, retune the scripts first. Only revisit hitstun with the user if no script tuning works.
- [The rising-overhead zone (frames 1–6) makes `jump_forward kick` an instant overhead that a ducker can't stop] → This is intended ("jumping is the answer to ducking"). `high_block` still guards it, and anti-air grounded attacks still knock jumpers down.
- [Landing-low attacks can be active right as the attacker lands next to a waking opponent] → A covered scenario. Knockdown invulnerability still applies, so a meaty landing low only works on an actionable target.
- [Golden diffs are large because every round changes] → This is expected for a config bump. Review the result and event summaries (`pf sim` text logs) rather than the raw JSON.
- [The probe for D5 used the CLI with the old recovery] → The new wake-up scenarios get dedicated tests, so the frame arithmetic in D5 is verified rather than assumed.

## Migration Plan

This is a config version bump (`2026.11-1` → `2026.11-2`). There are no persisted logs to migrate yet, because Phases 2–5 don't exist. To roll back, revert the change and regenerate the goldens.
