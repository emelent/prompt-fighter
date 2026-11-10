# Tasks

## 1. Config

- [x] 1.1 In `sim/config.go`, add the new tunables from design §6 with their defaults. Replace `ComboScalePct`/`ComboFloorPct` with `ComboGrowthPct`, and set the version to `2026.11-3`. Update `sim/config_test.go` (defaults, including the new "anti-passivity defaults" scenario) and the pinned hash. Verify `go test ./sim -run Config` passes.

## 2. Combo growth

- [x] 2.1 Replace the combo multiplier in `resolve` with `100 + ComboGrowthPct × (combo − 1)`, keeping `max(1, …)` and integer division. Update `sim/result_test.go` and the combo tests to the spec's growth sequence (10, 11, 13, 14, 16 for base 10) and to `punch punch kick` = 5/5/11 → 79 HP. Verify `go test ./sim -run 'Combo|Result'` passes.

## 3. Guard break

- [x] 3.1 Add `blockStreak`/`lastBlockT` and `stunGuardBreak` to the fighter. Implement the streak (increment on each guarded attack; reset on attack start, after the 60-frame lapse, when hit, and on guard break) and the `resolve` branch from design §2:
  - no damage, pushback, 20 frames of stun, `on_hit` interrupt
  - the attacker goes into its own recovery, not blockstun
  - a `guard_break` event in place of `block`
  - frame `state`/`anim` `guard_break`

  Verify `go build ./...`.
- [x] 3.2 Add `sim/guardbreak_test.go`, which verifies `go test ./sim -run GuardBreak` passes, covering the spec scenarios:
  - the 4th block breaks, the attacker recovers normally, and punishing with a kick connects
  - the streak lapses after 60 frames
  - attacking resets the streak
  - `on_hit` runs after the stun
  - a guard-break mirror match stays symmetric

## 4. Passivity boost

- [x] 4.1 Add the `passive` counter. It increases by 1 per frame when the fighter is not stunned or knocked down and isn't starting an attack, by 2 when moving away by `move_back` or `jump_back` drift (not pushback), and resets on attack start. Apply `boostPct` in hit and throw outcomes per design §3, record `Detail.Boost` when above 100, and consume the counter on apply. Verify `go build ./...`.
- [x] 4.2 Add `sim/passive_test.go`, which verifies `go test ./sim -run Passive` passes, covering:
  - the boost thresholds (119/120/150/239/240 → 100/110/120/140/150)
  - the kick at passive 150 dealing 10 damage with `boost: 120` and resetting the counter
  - 60 frames of `move_back` reaching 120
  - attacking (even a whiff) resetting the counter
  - stun frames not counting
  - a boosted throw
  - an idle-vs-idle mirror staying a draw

## 5. Stale attacks

- [x] 5.1 Track `lastAtk`/`repeats` on attack start, with overlays counted as their own attacks. From the 6th repeat, compute `effRange` (design §4), emit `stale` with attack, repeats and range, and whiff with reason `stale` when the target is within full range but beyond `effRange`. Verify `go build ./...`.
- [x] 5.2 Add `sim/stale_test.go`, which verifies `go test ./sim -run Stale` passes, covering:
  - the 6th punch at distance 7 emitting `stale` and whiffing `stale`
  - the range sequence 85/70/55/40%
  - mixing attacks resetting the count
  - `jump_kick` not counting as `kick`
  - damage and frame data being unchanged

## 6. Log text, sensors and language

- [x] 6.1 Add `guard_break`, `stale` and the boosted-hit suffix to `sim/textlog.go` (design §7), and document the new events, fields, state/anim and whiff reason in `docs/RULES.md`'s event and frame tables. Extend `sim/textlog_test.go`. Verify `go test ./sim -run TextLog` passes.
- [x] 6.2 Add the sensors `guard`, `passive`, `enemy.passive` and `repeats` to `sim/dsl/vocab.go` (as integer sensors) and to the `view` snapshot. Cap `passive` at 999. Add validator tests (the spec's `passive > 90 or guard >= 3 or repeats >= 5` validates) and interpreter tests for each sensor's value, including `passive` reading 100 after 100 idle frames. Verify `go test ./sim/...` passes.
- [x] 6.3 Document the new rules in `docs/RULES.md`: guard break, the passivity boost, stale attacks, combo growth and the new defaults table. Verify by reading the updated sections against the specs.

## 7. Samples, goldens and balance

- [x] 7.1 Retune `scripts/*.pf` minimally per design §8, keeping each archetype and its header comment accurate. Run `go run ./cmd/pf selfplay scripts/` until every sample has at least one win and one loss, and record the final table in this task's notes. Verify `go test ./sim -run 'Balance|Samples'` passes.
  - Notes: changed `ducker.pf` (jabs from the duck instead of a 6th stale sweep), `thrower.pf` (mixes in a punch after 5 throws) and `turtle.pf` (backs off at `guard >= 3`, kicks back at kick range). `jumper.pf`, `lazy.pf` and `rushdown.pf` are unchanged. Final selfplay table (row vs column, W-D-L): ducker beats rushdown and thrower, loses to jumper and lazy. jumper beats ducker, lazy, thrower and turtle, loses to rushdown. lazy beats ducker and turtle, loses to jumper, rushdown and thrower. rushdown beats jumper, lazy, thrower and turtle, loses to ducker. thrower beats lazy, loses to the other four. turtle beats thrower, draws ducker, loses to the rest.
- [x] 7.2 Regenerate the golden logs (`go test ./sim -run Golden -update`) and the `pf sim` text snapshots in `cmd/pf/testdata/text`. Review the `rushdown_vs_turtle` and one `lazy` pair diff by hand for correct guard-break and boost events. Verify `go test ./...` passes.

## 8. Player guide and MCP

- [x] 8.1 Update `docs/PLAYER_GUIDE.md` and name config `2026.11-3`:
  - the guard-break, passivity, stale-attack and combo-growth rules
  - the four sensors
  - strategy notes on mixing attacks, not overstaying in block, and punishing a passive opponent
  - at least one valid `pf` example using the new sensors

  Verify `go test ./docs` passes (sync test and examples), and `go test ./mcpserver` passes, since it serves the guide.

## 9. Replay viewer

- [x] 9.1 Port the new event lines to `web/src/replay.ts`, add `boost` to `web/src/types.ts`, and extend `web/test/replay.test.ts` so the feed still equals the regenerated text snapshots. Verify `npm test` in `web/` passes.
- [x] 9.2 Add the `GB` guard-break pose and the `guard_break` mapping to `web/src/fighterart.ts`. Extend `fighterart.test.ts` with the mapping, and confirm the golden-log coverage test still passes with the new logs. Verify `npm test` passes.
- [x] 9.3 Add the `guard_break`, `boost` (`×1.3`) and `stale` callouts with an `alpha` fade to `web/src/callouts.ts`, and draw them in `web/src/stage.ts`: per-glyph alpha, with callouts on the same fighter stacked. Extend `callouts.test.ts` with the spec scenarios, including the fade at 0 and 20 frames. Verify `npm test` passes.
- [x] 9.4 Rebuild `web/dist`, rebuild `pf`, and check in headless Chrome (puppeteer) that there are no console errors. Take screenshots of a guard break (pose and callout), a boosted hit (`×1.x`), a `STALE` callout and a fading callout mid-life, then review them.

## 10. Wrap-up

- [x] 10.1 Run `go vet ./...` and `go test ./...` and confirm both pass. Then run `.github/scripts/mcp-smoke.sh` against a local `pf serve`, and add the new rules to the README's one-line game description if needed. In `docs/ISSUES.md`, add a section for this change with follow-ups found during the work (for example, tuning the thresholds after real matches).
