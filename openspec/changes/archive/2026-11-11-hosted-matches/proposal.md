# Proposal

## Why

Today a match can only be created by a player, who takes seat a. The user wants to run matches like a tournament host:
- one person sets up a match on the server without playing
- two players, who may be LLMs driving the MCP tools or people using the web UI, join from a single game link
- the host watches each round as it is played

The first screen should also look like a game. It should show the Prompt Fighter cover art with just two choices, not the current form-heavy lobby.

## What Changes

- **Cover screen at `/play`:** the cover art (`graphics/prompt_fighter_cover.jpeg`, scaled down for the web) with two buttons, **Create match** and **Join match**. The menu music and its sound toggle stay. The lobby's create-with-script form and its separate join and watch boxes are replaced.
- **Create match (hosted):** creates a match with both seats empty. The creator takes no seat, and the browser goes straight to the spectator waiting screen for that match.
- **One game link for both players:** `/play/matches/{id}`. The first player to join takes seat a, and the second takes seat b. Each enters a name and a round-1 script, on the play page (web UI) or through the MCP `join_match` tool.
  - Round 1 runs when the second player joins, as today.
  - From then on, the existing flow applies: each round runs once both players submit, the host's waiting screen counts down and plays it, and Done returns to the waiting screen.
- **Waiting screen for setup:**
  - While seats are free, it shows "Waiting for players (n/2 joined)" with each joined player's name.
  - It shows the game link with a copy button, plus copy-ready instructions for joining with an LLM: the `claude mcp add` command for this server, and a prompt to give the model.
- **Join match screen:** takes a match id or link, then offers **Join as a player** (when a seat is free) and **Watch**.
- **HTTP API:**
  - `POST /matches` with an empty body (no name or script) creates a hosted match. It returns 201 `{match_id, match}`, with no seat or token, and the status `waiting_for_players`.
  - `POST /matches/{id}/join` fills the first free seat (a, then b) and reports which seat it gave.
  - Creating with `{name, script}` keeps today's behaviour (creator in seat a, status `waiting_for_opponent`), so existing MCP and API clients are unaffected.
- **MCP:**
  - `join_match` accepts a game link as well as a bare id.
  - If the link points at a different server than `pf mcp --server`, it fails with an explanation of how to point `pf mcp` at that server.
  - Its description and the state summaries mention hosted matches and which seat was assigned.
- **Assumptions (recorded, to confirm in review):**
  - The web cover screen's Create always makes a hosted match. A web player who wants to play opens the game link too.
  - One shared link, first come first served.
  - Round 1 still runs when the second player joins with their script.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `match-play`: "Create and join" gains hosted creation, the `waiting_for_players` status, and joining the first free seat.
- `play-ui`: "Lobby" becomes the cover screen with Create match and Join match. "Play page states" lets visitors join when either seat is free.
- `spectator-mode`: "Waiting screen" adds the setup view (players joined n/2), the game link and the LLM instructions.
- `mcp-server`: "Match tools" lets `join_match` take a link, with the server-mismatch error and the assigned-seat wording.

## Impact

- **Server:**
  - `server/match.go`: a hosted constructor, `waiting_for_players`, and join filling the first free seat.
  - `server/matches.go`: the create handler accepts an empty body, and join returns the assigned seat.
  - Tests for both, plus `docs/API.md`.
- **MCP:** `mcpserver/tools_match.go`: link parsing, the mismatch error, descriptions and summaries. Tests in `mcpserver`. The player guide's "How to play" section is updated.
- **Web:**
  - `web/src/play.ts`: the cover screen, the join screen, and joining either seat.
  - `web/src/playstate.ts`: the join state for any free seat, and the new status.
  - `web/src/spectate.ts` and `web/src/watch.ts`: the setup view, game link and instructions.
  - `web/src/match.ts`: status text.
  - `web/src/style.css`.
  - The cover image is resized into `web/src/images/` by an `npm run assets` script (the audio script renamed and extended).
  - `web/dist` is rebuilt.
- **Compatibility:** existing matches on disk and `create_match` by a player behave exactly as before. Clients that assumed `join` always returns seat `b` still get `b` for player-created matches.
