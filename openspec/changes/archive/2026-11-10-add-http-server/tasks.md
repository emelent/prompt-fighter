# Tasks

## 1. API core

- [x] 1.1 Create the `server` package with `Options` (DataDir, MaxSims, SimWait, MaxBody) and `New(Options) (http.Handler, error)`. Implement the route table with method checks (JSON 405 with an `Allow` header), the catch-all JSON 404, the error-envelope helper (including `invalid_script` errors with `player`/`line`/`col`/`message`/`text`), and the strict decode helper: a 64 KiB `MaxBytesReader` returning 413, `DisallowUnknownFields`, and a trailing-data check. Test malformed JSON, the unknown field, unknown route, wrong method and oversized body. Verify that `go test ./server -run 'Envelope|Route|Decode'` passes.
- [x] 1.2 Implement `GET /healthz` and `POST /validate`. Test the version fields, a valid script, and an invalid script whose error `text` matches `pf validate`'s line exactly. Verify that `go test ./server -run 'Health|Validate'` passes.
- [x] 1.3 Implement the simulation semaphore (design D3) and `POST /sim`: name defaults and the 40-byte cap, `frames` 1–600, and 422 with player-tagged errors. Test byte-equality with `sim.Simulate().JSON()`, `frames` 120, `frames` 0 and 601 returning 400, an invalid p2, and 503 `busy` with `MaxSims: 1` and a held slot. Verify that `go test ./server -run Sim` passes.

## 2. Match model

- [x] 2.1 Implement the pure match model in `server/match.go` (design D4): seats, status transitions, Create/Join/Submit, `readyToPlay`, `applyRound` with side alternation and seat mapping, the 2-win, draw and 5-round-cap rules, and `View()` without pending scripts or token hashes. Add id and token generation with `crypto/rand`, and constant-time token checking. Add table tests for every match-play scenario that needs no HTTP, using fake round results. Verify that `go test ./server -run Match` passes.

## 3. Storage

- [x] 3.1 Implement the file store (design D6): validated ids, atomic temp-write + fsync + rename + directory fsync, `match.json` with `store_version` 1, `rounds/<n>.json` holding exact log bytes, and a loud failure on an unknown `store_version`. Test the round trip, a path-traversal id being rejected, an unknown `store_version` being rejected, and a leftover temp file from a simulated crash not being treated as data. Verify that `go test ./server -run Store` passes.

## 4. Match endpoints

- [x] 4.1 Implement `POST /matches`, `POST /matches/{id}/join`, `POST /matches/{id}/scripts` (bearer token), `GET /matches/{id}` and `GET /matches/{id}/rounds/{n}`, with per-match locking and the request flow from design D5. Write the round log before `match.json`. Add `httptest` tests for each match-play scenario:
  - create
  - join runs round 1
  - seat already taken returns 409
  - an invalid script on create returns 422 and creates nothing
  - a missing token returns 401
  - the token is not stored in plain text (scan every file)
  - revise between rounds
  - replace before the round runs
  - submit after finish returns 409
  - the pending script is hidden
  - an unknown match returns 404
  - a round log equals the engine output
  - an unplayed round returns 404
  - concurrent final submissions record exactly one round

  Verify that `go test -race ./server -run Matches` passes.
- [x] 4.2 Add the restart test: play round 1, have seat a submit, build a new handler on the same data directory, check the state is unchanged, then have seat b submit and check round 2 uses seat a's earlier script. Add the full-match test: two seats play to a finished match with a winner through HTTP only. Verify that `go test ./server -run 'Restart|FullMatch'` passes.

## 5. CLI and docs

- [x] 5.1 Add the `pf serve` subcommand (design D7), taking a context for tests: `--addr` and `--data` defaults, creating the data directory, a fast failure (exit 2) on a bad address, a one-line startup log, and a graceful shutdown with exit 0. Add it to the usage text. Test the defaults, `--addr nonsense` exiting 2, and a cancelled context exiting 0 after serving `/healthz`. Verify that `go test ./cmd/pf -run Serve` passes.
- [x] 5.2 Write `docs/API.md`:
  - every endpoint, with its request and response JSON and example `curl` calls
  - the error codes table
  - the limits
  - the match rules (sides, wins, draws, the 5-round cap, revision, visibility)
  - the on-disk layout
  - the single-process limitation

  Update `docs/ISSUES.md`: add these Phase 2 follow-ups to the Phase 2 section, and move the `--config` item to wherever it still applies.
  - idle-seat forfeit or timeout
  - match cleanup
  - multi-process locking

  Verify by cross-checking every route registered in `server/api.go` against `API.md`.

## 6. Integration

- [x] 6.1 Run `go vet ./...`, `go test ./... -count=1` and `go test -race ./server ./sim -count=1`, and confirm all pass. Then do a manual smoke run: start `pf serve` against a temporary data directory, play one match between two sample scripts with `curl`, fetch a round log, and stop the server with SIGINT. Report the transcript.
