# Proposal

## Why

Phases 1–4 deliver the engine, the match API, the MCP server and a replay viewer. Playing a match in a browser, though, still means `curl` or an MCP client: there's no page to create or join a match, write a script and submit it. Phase 5 of the build plan closes that loop:

> Lobby page (create/join match), script editor (CodeMirror) with live validation, waiting state, auto-play the replay when the round resolves. The Go server serves the built `/web` assets so everything ships as one binary.

## What Changes

- **Lobby at `/play`.**
  - **Create a match:** a fighter name plus a round-1 script written in the editor. The script can start from one of the bundled sample bots.
  - **Join a match:** paste a match id, or open an invite link.
- **Play page at `/play/matches/{id}`.** It's the page each player keeps open during a match, and it shows one of these states:
  - **join:** you have no seat in this match and seat b is free.
  - **spectator:** you have no seat and the match is full. The page links to the replay viewer.
  - **waiting for opponent:** you created the match and no one has joined yet. The page shows an invite link to copy.
  - **your move:** submit your script for round N, revised or kept as it is.
  - **waiting for their script:** you've submitted, so the page says so. You can still revise until the round runs.
  - **finished:** the winner and links to every round's replay.
- **Script editor:** CodeMirror 6, with
  - syntax highlighting for `.pf`
  - **live validation**: as you type, the script is checked through `POST /validate` (debounced), and errors are shown inline at their line and column, with the same messages as `pf validate`
  - a byte counter against the 4096-byte limit
  - submission disabled while the script is invalid
- **Auto-play:** the play page polls the match. When a round resolves, whether through your own submission or your opponent's, it opens that round's replay and **starts playing it automatically**. The replay's back link returns to the play page, with your script still in the editor.
- **Seat tokens and drafts** are kept in the browser's `localStorage`, keyed by match id, so a refresh or a reopened tab keeps your seat. Drafts are also saved per match.
- **`GET /` redirects to `/play`,** so opening the server's root goes straight to the lobby.
- **One binary:** the lobby and play page are part of the existing `web/` app (Phaser 3 + strict TypeScript + Vite), built into `web/dist` and embedded in `pf`, which `pf serve` serves. No new server-side endpoints are needed; the pages use the existing match API.

## Capabilities

### New Capabilities
- `play-ui`: the browser lobby and play page. Covers creating and joining matches, the play-page states, the script editor with highlighting, live validation and limits, submission, the waiting state, auto-play on round resolution, storing seat tokens and drafts in the browser, and starting from sample bots.

### Modified Capabilities
- `http-api`: the "JSON transport and error envelope" requirement adds `/play` pages and the `GET /` redirect to the non-JSON exceptions, which so far were only the `/view` pages.
- `replay-viewer`: adds an "Auto-play and return links" requirement. A round URL carrying `?autoplay=1` starts playing on load, and `?from=play` points the back link at the play page.

## Impact

- New web code:
  - `web/src/play.ts`: the lobby and play-page controller
  - `web/src/editor.ts`: the CodeMirror setup in a shadow root
  - `web/src/pflang.ts`: the `.pf` tokenizer
  - `web/src/playstate.ts`: the pure state logic
  - `web/src/seats.ts`: token and draft storage

  Plus routing in `app.ts`, autoplay in the round player, and Vitest tests. `web/dist` is rebuilt.
- Server: two page routes (`/play` and `/play/matches/{id}`, serving the same embedded app) and an exact-root redirect in `server/api.go`, with tests and `docs/API.md` rows.
- **Dependencies:** new exact-pinned npm runtime packages `@codemirror/state` 6.7.6, `@codemirror/view` 6.43.15, `@codemirror/language` 6.13.1, `@codemirror/lint` 6.9.7, `@codemirror/commands` 6.11.1 and `@lezer/highlight` 1.2.5. No new Go modules.
- Docs: `docs/API.md` gets a "Playing in the browser" section, `docs/PLAYER_GUIDE.md` §11 mentions the lobby, and `docs/ISSUES.md` gets a Phase 5 section.
- Unchanged: the engine, the event log, the match API contract, MCP and the config version.
- **Out of scope:**
  - user accounts and authentication beyond seat tokens
  - public hosting and its hardening: rate limits, match cleanup, multi-process locking (already in `ISSUES.md`)
  - a leaderboard
  - autocompletion in the editor
  - mobile-specific layouts
