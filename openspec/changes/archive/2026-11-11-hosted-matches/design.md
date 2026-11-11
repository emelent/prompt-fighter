# Design

## Context

- `server/match.go`:
  - `newMatch(id, cfg, name, script, hash)` always fills seat a.
  - `join()` always fills seat b and errors with `errSeatTaken` if b is set.
  - Statuses are `waiting_for_opponent`, `awaiting_scripts` and `finished`.
  - `ready()` needs both seats with a pending script.
  - `sides(n)` makes seat a p1 in odd rounds.
- `server/matches.go`: `handleCreateMatch` decodes `seatReq{name, script}` and runs `check(req, "a")`. `handleJoin` checks `m.B != nil` and responds with seat `"b"`.
- MCP `join_match` calls `POST /matches/{id}/join` with the given `match_id` and the server URL from `--server`.
- Web:
  - `lobbyPage` in `play.ts` has the create form (editor), a join-by-id box and the Watch box (from add-spectator-mode), plus the menu music toggle.
  - `playstate.ts` returns `join` only when seat b is free.
  - `watch.ts` and `spectate.ts` (`waitingState`) treat a missing seat b as "waiting for an opponent to join".
  - `match.ts` has the status texts.
  - `matchIdFrom()` already parses an id or link.
- Assets: `web/convert-audio.sh` re-encodes `audio/` into `web/src/audio/`. The cover is a 1696×2528 JPEG of 2.6 MB in `graphics/`.

## Goals / Non-Goals

**Goals:**
- Hosted matches with no API break: player-created matches, existing stored matches and `create_match` behave exactly as before.
- One link for both players, usable by people (web) and by LLMs (MCP).

**Non-Goals:**
- Host privileges, such as kicking players, restarting or a host token. The host is just a spectator who created the match.
- Reserving seats per player, which would need separate links.
- A public list of hosted matches (still a follow-up).
- Changing round timing: round 1 still runs on the second join.

## Decisions

### 1. Hosted creation is "an empty create request"
`handleCreateMatch` decodes into `seatReq` with pointer fields.
- **Both absent:** a hosted match. `newHostedMatch(id, cfg)` creates it with `Status = "waiting_for_players"` and `A = B = nil`, and the response is `{match_id, match}`.
- **Exactly one present:** 400 `bad_request` ("send both name and script to take seat a, or neither to host a match").
- **Both present:** today's path.

- **Why:** no new endpoint or flag, and old clients that always send both are unaffected.
- **Alternative:** `{"host": true}`. This was rejected because it adds an invariant to validate, and an empty create is unambiguous.

### 2. Join takes the first free seat
`join(name, script, hash) (Seat, error)`:
- if `A == nil`, it fills a and the status becomes `waiting_for_opponent`
- otherwise, if `B == nil`, it fills b, and the status becomes `awaiting_scripts` as today, after which `playIfReady` runs round 1
- otherwise it returns `errSeatTaken` ("both seats are taken")

The handler validates the name and script before choosing, with the error field named after the seat it would get, and responds with the assigned seat.

`sides(n)`, `ready()`, `submit` and `view()` already handle nil seats or need no change. `view()` reports `null` seats as today.

Stored matches need no migration: an old match always has seat a set, so join behaves exactly as before.

### 3. MCP link handling
`join_match`'s `match_id` argument goes through `parseMatchRef(ref, serverURL)`. The result:
- **A bare 16-hex id:** used as is.
- **A URL with `/play/matches/{id}` or `/watch/matches/{id}`:** returns the id, plus the link's origin.
- **A link origin that differs from the configured server's origin:** a tool error, "This game link is for `X`, but this pf mcp talks to `Y`. Restart it with `pf mcp --server X` (for Claude Code: `claude mcp add prompt-fighter -- pf mcp --server X`)."
- **Anything else:** "`match_id` must be a 16-character match id or a game link".

The same parser applies to `get_match`, `submit_script` and `get_round_replay`, since models will pass the link around. Descriptions mention links. The state summary's "what to do next" line covers `waiting_for_players`.

### 4. Web: cover screen and join screen
`lobbyPage` becomes:
- A two-column layout (one column below 800 px): the cover `<img alt="Prompt Fighter: two martial artists clash on a rooftop at sunset">` on one side.
- On the other: the title, Create match (primary) and Join match buttons, and the menu-music toggle.
- **Create match:** `postJSON('/matches', {})`, then `location.assign('/watch/matches/{id}')`. Errors show in an alert under the buttons.
- **Join match:** swaps the button area for a form (an id-or-link input, Continue, and Back). Continue runs `matchIdFrom()` and `GET /matches/{id}`, then shows the match's two seats and the buttons **Join as a player** (only if a seat is free) and **Watch**. 404 and bad input show inline errors.

The editor-based create form, the old join box and the Watch box are removed from the lobby. `editorPanel` stays, because the play page uses it.

`playstate.ts`:
- `join` when this browser holds no seat and either seat is free.
- `waitingOpponent` when this browser holds a seat and the other is empty.
- `waiting_for_players` is handled as "no seats yet".

### 5. Web: setup view on the waiting screen
`waitingState()` gains `kind: 'setup'` when `status` is `waiting_for_players` or `waiting_for_opponent`. It carries `{ joined: [names], gameLink, origin }`, with `gameLink = origin + '/play/matches/' + id` and `origin` from `location.origin`, passed in so the function stays pure.

`watch.ts` renders "Waiting for players (n/2 joined)", the joined names, the game link and a Copy button. It also renders an "Add the MCP server" block (a code line and a Copy button) and a "Prompt for your LLM" block with a Copy button, reading:

> Join the Prompt Fighter match at <game link> using the prompt-fighter MCP tools. Read the player guide first, then join with a name and a round-1 script, and keep playing until the match is over.

Once both seats are filled, round 1 has been played, so the existing countdown takes over. `match.ts` status texts add `waiting_for_players`, "waiting for players to join".

### 6. Cover image pipeline
`web/convert-audio.sh` becomes `web/convert-assets.sh`. It runs the same audio steps, and also uses `sips` (macOS) or ImageMagick `magick` (if present) to resize `graphics/prompt_fighter_cover.jpeg` to 800 px wide, JPEG quality 80, into `web/src/images/cover.jpg` (about 150 KB).
- `npm run assets` runs it. `npm run audio` stays as an alias.
- The resized file is committed, so builds don't need the tools.
- The image is imported with `?url`, so Vite fingerprints it. Images aren't fetched by script, so the CSP's `img-src 'self'` covers it, and inlining is fine either way.

## Risks / Trade-offs

- **[Anyone with the link can take a seat]** → Expected for trusted local use, and the same as today's invite link. The setup view shows who joined, so a host notices an unexpected player.
- **[The LLM's MCP server points elsewhere]** → `join_match` explains the mismatch and the exact restart command, and the waiting screen's instructions already include the right `--server`.
- **[The host closes the tab]** → Nothing breaks. The match carries on between the players, and the host can reopen the watch link.
- **[The first screen loses the web "create and play" path]** → A web player creates a hosted match and opens the game link themselves. One extra click, recorded as an assumption for review.

## Migration Plan

No data migration is needed: stored matches always have seat a set. Rollback is a code revert. Hosted matches created meanwhile have `A == nil`, which an older server would mishandle, so roll back only on a fresh data dir or after those matches finish.
