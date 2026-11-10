# Tasks

## 1. Config

- [x] 1.1 Add `AirZoneKind` (JSON-encoded by name) and the `AirZones` table with defaults 1–6 rising overhead, 7–13 air-to-air, 14–16 falling overhead and 17–20 landing low. Set the jump_kick and dive_punch `Height` to `HeightNone`, hitstun to 13, hit pushback to 2 and throw recovery to 18. Bump `DefaultVersion` to `2026.11-2` and re-pin the hash. Update `TestConfigBuildSpecFrameData` and the overlay test to the new spec values, and add a test that the zones cover airborne frames 1–20 contiguously. Verify that `go test ./sim -run Config` passes.

## 2. Air-attack zones and connection

- [x] 2.1 Record `act.zone` and the effective `act.height` at attack start (design D2), and switch `connect()` and `guarding()` to `act.height`. Implement the ordered whiff chain from design D3, including the new `grounded` reason and `ducked` applying only to grounded high attacks. Add tests for each new combat-rules scenario: rising overhead hits a ducker and the jump continues, overhead guarded by `high_block`, air-to-air whiffs `grounded` against a grounded target, air-to-air hits and knocks down an airborne target, landing low guarded by `low_block` but not `high_block`, and the zone fixed at start. Verify that `go test ./sim -run 'Zone|Whiff'` passes.
- [x] 2.2 Implement the landing-low carry-through in `preThink()`, and display an in-progress attack ahead of `landing` in `frame()` (design D4). Replace `TestStanceAirAttackEndsOnLanding` with the falling-ends-on-landing and landing-low-carries-through scenarios from the script-execution spec. Each asserts the anim row, the 5 active frames after touchdown, and when the following `wait` starts. Verify that `go test ./sim -run Stance` passes.

## 3. Hitstun, pushback, throws and combos

- [x] 3.1 Update the tests whose expectations depend on hitstun 5 or pushback 3 so they match the new spec values: hit hitstun and distance 8, wall transfer to x=10, the frame numbers in the `handler_test.go` scenarios, and `TestComboResetsWhenActionable`, which should pin a short hitstun explicitly. Update `TestKnockdownInvulnerable` for the throw's 24-frame total. Verify that `go test ./sim -run 'Hit|Handler|Knockdown|Combo'` passes.
- [x] 3.2 Add the new throw and combo scenario tests: a wake-up punch beats an immediate re-throw, an immediate re-throw beats a wake-up `high_block`, `punch punch kick` gives combos 1/2/3 with damage 5/4/5 and 86 HP at the defaults, and in `punch punch punch` the third punch whiffs `out_of_range`. Verify that `go test ./sim -run 'Throw|Combo'` passes.

## 4. Samples, balance and goldens

- [x] 4.1 Add `scripts/ducker.pf` (duck hold, `low_jab` when in range, sweep against approaching enemies) with strategy header comments. Add the "ducker ducks" test asserting that the ducker is in the `duck` state on at least one frame against every other sample. Verify that `go run ./cmd/pf validate scripts/*.pf` reports `ok` for all six and that `go test ./sim -run Ducker` passes.
- [x] 4.2 Run `go run ./cmd/pf selfplay scripts/` and retune the sample scripts (not the config) until every script beats at least one other and loses to at least one other. Verify that `go test ./... -run Balance` passes and record the final matrix.
- [x] 4.3 Regenerate the golden logs with `go test ./sim -run Golden -update`, review the `pf sim` text summaries of the golden pairs for sanity, and adjust `goldenPairs` if needed so that at least one KO and one timeout remain. Verify that `go test ./sim -run Golden` passes.

## 5. Docs and integration

- [x] 5.1 Update `docs/RULES.md`:
  - Header: the config version.
  - Frame data: throw 5/1/18, hitstun 13, pushback 2, and the air-zone table.
  - Connection and whiffs: the whiff chain including `grounded`.
  - Carry-through.
  - Throw section: the wake-up triangle replaces the "balance note".
  - Combos: no longer dormant.
  - Blocks: the D7 interpretation is now confirmed.
  - Reaction windows: punch being unreactable is now intended.
  - The `knockdown`/`whiff` detail tables.

  Verify that every whiff reason the code emits is listed.
- [x] 5.2 Update `docs/ISSUES.md`: mark Phase 1 issues 1–6 resolved, with a one-line decision each and the config version (`2026.11-2`) where values changed. Verify that no unresolved item remains in the Phase 1 section.
- [x] 5.3 Run `go vet ./...`, `go test ./... -count=1` and `go test -race ./sim`, and confirm all pass, including the determinism and symmetry tests. Report the new selfplay matrix and any balance observations to the user.
