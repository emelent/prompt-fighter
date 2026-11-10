# Open Issues

Decisions and known problems, grouped by the phase they block. When an issue is resolved, record the decision, and the config version bump if there is one, then remove it from this list.

## Phase 1: Simulation core (resolved)

All six review issues were decided and implemented in the OpenSpec change `phase1-balance-decisions`, config version `2026.11-2`. The rules are described in `docs/RULES.md`.

| # | issue | decision |
|---|---|---|
| 1 | Throws could be looped on wake-up | **Fixed:** throw recovery 15 → 18. An attack on wake-up now beats an immediate re-throw, while the re-throw still beats a wake-up block (`2026.11-2`). |
| 2 | Combos were dormant | **Fixed:** hitstun 5 → 13, hit pushback 3 → 2. `punch kick` and `punch punch kick` link, and pushback ends punch strings (`2026.11-2`). |
| 3 | Ducking hard-countered every air attack | **Fixed:** air-attack zones by start frame. Rising and falling overheads hit duckers, the peak is air-to-air only, and the landing low carries through touchdown. Ducking now dodges only grounded high attacks (`2026.11-2`). |
| 4 | Blocked-attacker stun interpretation | **Confirmed:** the attacker is stunned for the attack's recovery, then `on_block` runs. No change. |
| 5 | Punch can't be reacted to | **Accepted** as intended: punch is the fast poke you have to predict. No change. |
| 6 | Sample bot coverage | **Done:** added `ducker.pf`. jumper now kicks as a landing low. All six samples satisfy the balance test. |

### Follow-ups (not blocking Phase 2)
- **Weak samples:** at `2026.11-2`, `lazy` and `thrower` each win only one matchup (2-0-8). That satisfies the balance rule, but they are the first candidates for script tuning.
- **Untested-in-play rules:** no sample attacks air-to-air, so the `grounded` whiff never happens in selfplay. Unit tests cover it. Consider an air-to-air bot when the sample set is next revised.

## Phase 2: HTTP server

- **Optional `--config` override for `pf sim`:** a JSON override for balance experiments. This is CLI-only, so it needs no spec change and can land in any phase (carried over from the Phase 1 design's open questions).

## Phase 3: MCP server and player guide

- `docs/PLAYER_GUIDE.md` should be written from `docs/RULES.md`. The Phase 1 balance decisions are now made (config `2026.11-2`), so the rules are stable to document.

## Phase 4: Replay viewer

- None yet.

## Phase 5: Submission UI

- None yet.
