# Proposal

## Why

Phase 1 delivered a deterministic engine that can only be driven from the `pf` CLI. Prompt Fighter's core loop is: write a script with your LLM, fight, watch the replay, revise, fight again. That loop needs a server two players can play a match through. The later phases need one too: the MCP server (Phase 3), the replay viewer (Phase 4) and the submission UI (Phase 5) all need an HTTP API for validating scripts, running rounds and fetching stored replays. This change is **Phase 2: the HTTP server**.

## What Changes

- **New `pf serve` command.** It runs an HTTP/JSON server. By default it binds to `127.0.0.1:8080` and stores data in `./data`.
- **Stateless endpoints:**
  - `GET /healthz` returns the server status and the engine's config and log-format versions.
  - `POST /validate` validates a script and returns the same errors as `pf validate`, as structured JSON.
  - `POST /sim` runs a single round between two scripts and returns the exact event log `sim.Simulate` produces, byte for byte. An optional `frames` sets the budget.
- **Best-of-3 matches with revision between rounds:**
  - `POST /matches` creates a match: the creator takes seat A with a name and a script and gets a seat token. `POST /matches/{id}/join` takes seat B.
  - Before each round, both seats submit a script (`POST /matches/{id}/scripts`, authorised by their seat token). Resubmitting the same script means "keep it".
  - The round runs as soon as both seats have submitted. Sides alternate: seat A is p1 in odd rounds and seat B is p1 in even rounds.
  - The first seat to win 2 rounds wins the match. Drawn rounds count for nobody. After at most 5 rounds, the seat with more wins takes the match, or the match is drawn if they're level.
  - `GET /matches/{id}` returns the match state. It never reveals the opponent's script for a round that hasn't been played. `GET /matches/{id}/rounds/{n}` returns a played round's event log.
- **Storage.** Matches and round logs are stored as JSON files on disk, one directory per match, written atomically. They survive restarts.
- **Protection for trusted, local use:** request-body size limit, a cap on concurrent simulations, per-request timeouts, and secret per-seat tokens (stored only as hashes). There are no user accounts; auth is deferred to Phase 5.
- **New `docs/API.md`** documenting every endpoint, request and response shape, and error code.
- The engine (`/sim`, `/sim/dsl`), the event log schema and the config are **unchanged**.

## Capabilities

### New Capabilities
- `http-api`: the HTTP/JSON surface. Covers `/healthz`, `/validate` and `/sim`, the JSON error envelope and codes, request limits and timeouts, and the guarantee that `/sim` returns the engine's log byte for byte.
- `match-play`: the best-of-3 match lifecycle over HTTP. Covers creating and joining matches, seat tokens, script submission per round, side alternation, round execution, win, draw and round-cap rules, hidden pending scripts, round-log retrieval, and on-disk persistence across restarts.

### Modified Capabilities
- `pf-cli`: adds the `pf serve` subcommand (flags `--addr` and `--data`). The existing `validate`, `sim` and `selfplay` requirements are unchanged.

## Impact

- New code: `/server` (handlers, match logic, file store) and a `serve` subcommand in `/cmd/pf`.
- Docs: new `docs/API.md`. `docs/ISSUES.md` gets a Phase 2 section updated with follow-ups.
- Dependencies: none new. Stdlib only (`net/http` with Go 1.22+ routing patterns, `crypto/rand`, `encoding/json`).
- Data: a new on-disk layout under the data directory, versioned with a `store_version` field.
- Downstream: Phases 3–5 build on these endpoints, and the replay viewer loads round logs from `GET /matches/{id}/rounds/{n}`.
- Out of scope: accounts and auth, public exposure, rate limiting per user, a leaderboard, match expiry or cleanup, websockets or push notifications (clients poll `GET /matches/{id}`), the MCP server and `PLAYER_GUIDE.md` (Phase 3).
