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

Implemented by the OpenSpec change `add-http-server`: `pf serve`, the stateless `/validate` and `/sim` endpoints, and best-of-3 matches with revision between rounds, stored as JSON files on disk. The API is described in `docs/API.md`.

### Follow-ups (not blocking Phase 3)
- **Idle-seat forfeit or timeout:** a seat that never submits stalls its match forever. A rule could forfeit a round, or the match, after N minutes. It can be added as a new requirement without changing the existing endpoints.
- **Match cleanup:** matches are never deleted or expired, so disk use grows without bound. Needed before any public deployment (Phase 5).
- **Multi-process locking:** per-match locks are in-process only, so two `pf serve` processes on the same data directory could play a round twice. A lock file, or a database, would fix it. Only needed if the server is ever scaled out.

## Any phase

- **Optional `--config` override for `pf sim`:** a JSON override for balance experiments. It is CLI-only and needs no spec change (carried over from the Phase 1 design's open questions).

## Phase 3: MCP server and player guide

Implemented by the OpenSpec change `add-mcp-server`: `pf mcp` (a stdio MCP server built on `github.com/modelcontextprotocol/go-sdk` v1.8.0, targeting protocol 2026-07-28 and still negotiating older revisions) and `docs/PLAYER_GUIDE.md`. The tools are `validate_script`, `simulate_round`, `list_sample_bots`, `get_player_guide`, `create_match`, `join_match`, `submit_script`, `get_match` and `get_round_replay`, plus resources for the guide, the rules and the sample bots.

### Notes
- **First third-party dependency:** the MCP SDK and its transitive modules (`github.com/google/jsonschema-go`, `golang.org/x/oauth2`, `golang.org/x/sync`, `golang.org/x/sys`, `golang.org/x/time`, `github.com/segmentio/encoding`, `github.com/segmentio/asm`, `github.com/yosida95/uritemplate/v3`). Only `mcpserver` and `cmd/pf` import them; `sim`, `sim/dsl` and `server` stay stdlib-only.

### Follow-ups (not blocking Phase 4)
- **MCP over Streamable HTTP:** for remote or hosted play. It needs the Phase 5 auth decisions first.
- **MCP prompts:** for example "write a counter to bot X" or "review my last round". These are additive.
- **Push updates for match state:** clients poll `get_match` today. MCP `subscriptions/listen` or a tasks-extension handle could notify when the opponent submits.
- **Seat tokens in model context:** tokens pass through the LLM conversation and client transcripts. That's acceptable for trusted, local play, but should be revisited together with accounts and auth in Phase 5.

## Phase 4: Replay viewer

Implemented by the OpenSpec change `add-replay-viewer`: a browser viewer served by `pf serve` under `/view` (home and local files, a live match page, a round player with frame links). It's built with Phaser 3.90.0, strict TypeScript and Vite, with the build committed in `web/dist` and embedded in `pf`. The usage is described in `docs/API.md`, under Replay viewer.

### Follow-ups (not blocking Phase 5)
- **CI for the frontend:** run `cd web && npm ci && npm run check` in CI, and fail if `npm run build` changes `web/dist`. Locally, `go test ./web` skips the frontend tests when `node_modules` is missing.
- ~~**Sprites and animation:**~~ done by `pixel-art-fighters`: code-generated pixel-art fighters, backdrop, sparks and pixel-font HUD.
- **Frame inspector:** a panel showing the raw `frames[f]` entry and that frame's events, for debugging scripts.
- **Mobile layout:** the stage scales, but the controls and panels are designed for desktop widths.
- **Bundle size:** Phaser makes up most of the 1.2 MB JS bundle. Fine for a local viewer, but worth revisiting (for example a custom Phaser build) before public hosting.

## Phase 5: Submission UI

Implemented by the OpenSpec change `add-submission-ui`: the lobby at `/play` (with `GET /` redirecting to it) and the play page at `/play/matches/{id}`. They have a CodeMirror 6 editor with live validation, the waiting states, and auto-play of each round's replay. They're part of the same embedded web app, so `pf` still ships as one binary. The usage is described in `docs/API.md`, under Playing in the browser.

### Follow-ups
- **Listing open matches:** the lobby can only open a match by id or invite link. Listing matches waiting for an opponent needs a new endpoint (for example `GET /matches?status=waiting_for_opponent`).
- **Accounts and auth before public hosting:** seat tokens are bearer secrets in `localStorage` and in MCP model context. Public hosting also needs the Phase 2 follow-ups (rate limits, match cleanup, multi-process locking) and an idle-seat timeout.
- **Bundle size:** the JS bundle is now about 1.55 MB (Phaser about 1.2 MB, CodeMirror about 340 KB). Lazy-load the editor and the stage with dynamic `import()` if it matters.
- **Mobile layout:** the editor and panels are designed for desktop widths.
- **Remaining Phase 4 plan items:**
  - `/replay?match=…&round=…` URLs
  - ~~"BLOCKED/THROW/TRADE/KO" callouts~~ done by `pixel-art-fighters` (plus `TECH`)
  - per-line frame and check cost annotations in the script panels
  - a 0.25×–2× speed range
  - a Playwright smoke test (the browser checks so far were run by hand with `puppeteer-core`)

## Pixel-art fighters

Implemented by the OpenSpec change `pixel-art-fighters`: the replay stage is now 16:9 pixel art generated in code (`web/src/fighterart.ts`, `backdrop.ts`, `pixelfont.ts`), with event callouts, a frame countdown and a round intro (`callouts.ts`, `playback.ts`). There are no image files, and the CSP is unchanged.

### Follow-ups
- **Character select:** both players use the same character, told apart by palette. More characters would need a way to choose (for example a field in the match or the script header).
- **Sound:** hit, block and KO effects. The game is created with `noAudio` today.
- **Non-Latin names:** characters outside the 5×7 font draw as boxes on the stage. Full names still show in the script panels.
- **Custom configs:** the knockdown get-up pose assumes the default 20 knockdown frames, and the round word assumes best of 3. Logs don't carry these values.
