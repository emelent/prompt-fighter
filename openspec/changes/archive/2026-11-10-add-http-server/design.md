# Design

## Context

The engine exposes `sim.Simulate(cfg, a, b Entrant) (*Log, error)`. It is a pure function: it has no shared state and runs safely in parallel (verified by `TestConcurrentSimulate` under `-race`). It returns `*sim.ScriptError` with `[]dsl.Error` for invalid scripts, and `Log.JSON()` gives the canonical compact bytes. `dsl.Validate(src, cfg.Limits)` gives the same errors `pf validate` prints. `cmd/pf/run.go` dispatches subcommands as `func(args, stdout, stderr) int`. The module is stdlib-only on Go 1.27, so the method-and-path routing patterns `net/http` gained in Go 1.22 are available. The user's decisions are recorded in proposal.md.

## Goals / Non-Goals

**Goals:**
- A `server` package exposing `New(Options) (http.Handler, error)`. Tests and `pf serve` share it, and it needs no global state.
- Match rules as a pure state machine, separate from HTTP and disk, so the spec scenarios can be tested as plain unit tests.
- Crash-safe, inspectable storage using only the stdlib.
- `/sim` and the round logs stay byte-identical to `sim.Simulate`, which is the contract the replay viewer depends on.

**Non-Goals:**
- Multi-process or multi-host deployment. Per-match locking is in-process only.
- Caching, match expiry or cleanup, pagination, or listing matches.
- Streaming or push. Clients poll `GET /matches/{id}`.

## Decisions

### D1. Package layout
- `server/api.go`: `New`, routing, the JSON envelope, body decoding, limits and the simulation semaphore.
- `server/match.go`: the pure match model. It defines `Match`, `Seat` and `Round`, and the methods `Create`, `Join`, `Submit` and `readyToPlay`. It also has `applyRound(n, log)` and the result rules (2 wins, draws don't count, 5-round cap), and `View()`, which builds the public state with pending scripts stripped. It does no I/O and has no clock.
- `server/store.go`: the file store, with `Load(id)`, `Save(m)`, `SaveRound(id, n, bytes)`, `LoadRound(id, n)` and `Exists(id)`.
- `cmd/pf/serve.go`: flag parsing, `http.Server` with timeouts, and signal-driven shutdown.

Alternative considered: putting everything in `cmd/pf`. Rejected because Phase 3 (MCP) will reuse the match logic directly, without going through HTTP.

### D2. Routing and the error envelope
Patterns are registered **without** methods (`/sim`, `/matches/{id}/join`, …). A small route table maps each pattern to an allowed set of methods, and a disallowed method gets a JSON 405 with an `Allow` header. A catch-all `/` returns JSON 404. Together these make every error use the envelope. The stdlib mux would otherwise write plain-text 404 and 405 responses.

The envelope helper writes `{"error": {"code", "message"}}`, plus `errors` for `invalid_script`. The `dsl.Error` entries are mapped to `{player?, line, col, message, text: e.String()}`.

Request decoding is done in one helper. It applies `http.MaxBytesReader(64 KiB)`; a `*http.MaxBytesError` becomes 413. It uses `json.Decoder.DisallowUnknownFields` and then checks for trailing data. The required fields and ranges (name 1–40 bytes, frames 1–600) are checked explicitly, and failures return 400.

### D3. Simulation concurrency
A semaphore (a `chan struct{}` of size `Options.MaxSims`, default `runtime.NumCPU()`) wraps every `Simulate` call, both `/sim` and match rounds. Acquiring it selects on the slot, a `time.After(Options.SimWait)` (default 5 s) and the request context. On timeout it returns 503 `busy`. A simulation takes milliseconds, so this mainly guards against bursts. Tests inject `MaxSims: 1` and a short `SimWait` to hit the 503 path deterministically, by holding the slot through a test hook.

### D4. Match model
- **Seats:** `a` and `b`. Each has `Name`, `TokenHash` (hex SHA-256), `Pending *string` and `Wins`.
- **Status:** `waiting_for_opponent` → `awaiting_scripts` → `finished`. `next_round` is `len(Rounds)+1`.
- **Create** sets seat a's `Pending`. **Join** fills seat b and sets its `Pending`. Then the caller checks `readyToPlay()`, which is true when both seats have a pending script and the match isn't finished.
- **Playing round n:** p1 is seat a if n is odd and seat b otherwise. The entrants are `{Name: seat.Name, Source: *seat.Pending}`. The engine winner `p1`/`p2` is mapped back to a seat. `applyRound` records `Round{N, P1Seat, WinnerSeat, Reason, EndFrame, HP{a,b}, ConfigVersion}`, adds the win, clears both `Pending`, and then sets the status: `finished` if a seat has 2 wins or n == 5 (taking the higher wins or a draw), otherwise `awaiting_scripts`.
- **Submit** checks that the match isn't finished (409 otherwise), validates the script (422) and replaces `Pending`. That allows seat a to replace its submission during `waiting_for_opponent`.
- **`View()`** builds the API state. It never includes `Pending` or `TokenHash`, only `submitted: Pending != nil`.

Tokens and ids come from `crypto/rand` and are hex-encoded: 16 bytes for a token (32 characters) and 8 bytes for an id (16 characters). Tokens are compared by hashing the presented bearer token and comparing it in constant time with `subtle.ConstantTimeCompare`.

### D5. Request flow and locking
Every match-mutating request runs under a per-match `sync.Mutex`, kept in a `map[string]*sync.Mutex` that is itself guarded by a mutex. Entries are never deleted, which is fine at Phase 2 scale. The sequence is:
1. Lock.
2. Load from the store.
3. Authorise.
4. Mutate.
5. If ready, acquire a simulation slot, `Simulate`, `SaveRound` and then `applyRound`.
6. `Save`.
7. Unlock.
8. Respond with `View()`.

Serialising per match makes "a round runs exactly once" hold even when both seats submit concurrently: the second request sees no pending pair. If simulation hits `busy`, the submission is still saved and the round runs on the next request that finds the match ready. Concretely, `GET` doesn't play rounds, but the next `Submit` re-checks `readyToPlay`. **Assumption, recorded:** with a 5-second wait on a millisecond workload, this path is practically unreachable, and it is covered by one test.

### D6. File store
The layout is `<data>/matches/<id>/match.json` (`store_version: 1` plus the match model) and `<data>/matches/<id>/rounds/<n>.json`, which holds the exact `Log.JSON()` bytes.
- **Atomic writes:** create a temp file in the same directory, write it, `fsync`, close, `rename` over the target, then `fsync` the directory.
- **Write order on a round:** the round log first, then `match.json`. `match.json` is the source of truth, so a crash between the two leaves at worst an orphan round file, which the replay of the same round overwrites.
- **Loading:** the store loads by id on every request. There is no in-memory cache, so a restart needs no recovery step.
- **Ids:** an id is validated as 16 hex characters before it touches the filesystem, which prevents path traversal.
- **Version check:** an unknown `store_version` fails loudly on load.

Alternative considered: one file per match with the logs embedded. Rejected because the logs are about 150–380 KB each, so every submit would rewrite megabytes.

### D7. `pf serve`
The flags are `--addr` (default `127.0.0.1:8080`) and `--data` (default `./data`). The command runs `os.MkdirAll(data/matches)` and listens with `net.Listen` first, so a bad address fails fast with exit 2. It serves with `ReadHeaderTimeout` 5 s, `ReadTimeout` 10 s, `WriteTimeout` 30 s and `IdleTimeout` 60 s. Startup logs one line to stderr. `signal.NotifyContext(SIGINT, SIGTERM)` triggers `srv.Shutdown`, with a 10-second grace period, and exit 0. For tests, the subcommand takes a context, so it can be stopped without sending real signals.

### D8. Config versions across upgrades
Rounds always use `sim.DefaultConfig()` of the running binary. Each `Round` records the `config_version` taken from its log. The match's top-level `config_version` is the version at creation. If the server is upgraded mid-match, later rounds show the newer version, and the state makes that visible rather than hiding it.

### D9. Tests
- **`server/match_test.go`:** table tests of the state machine for every match-play scenario (side alternation, seat mapping, two straight wins, draws extending the match, the 5-round cap, replace-before-round, submit after finish, hidden pending scripts). It uses a fake round result so no simulation runs.
- **`server/api_test.go`:** `httptest` tests of every http-api and match-play scenario:
  - envelope codes, unknown field, 404 and 405
  - 413 on a 65 KiB body
  - busy (with `MaxSims: 1` and a held slot)
  - `/sim` and round logs byte-equal to `sim.Simulate`
  - 401 without a token
  - the token absent from all stored files
  - concurrent final submissions producing exactly one round (run under `-race`)
  - a restart mid-match (a new handler on the same `TempDir`)
- **`cmd/pf/serve_test.go`:** the default flag values, a bad `--addr` exiting 2, and a cancelled context shutting down cleanly with exit 0.

## Risks / Trade-offs

- [Per-match locks are in-process only: two `pf serve` processes on one data directory could double-play a round] → This is documented in `docs/API.md` and `ISSUES.md` as a single-process limitation. A lock file is a Phase 5 item if it's ever needed.
- [Unbounded disk growth, since matches are never deleted] → Acceptable for local play. Recorded as a Phase 5 follow-up (expiry and cleanup).
- [Anyone who learns a match id can read its state and played-round logs] → This is intended: replays are meant to be shareable. Pending scripts and tokens are never exposed. Ids are 64-bit random, so they can't be guessed.
- [Polling clients add load] → It's trivial at this scale. Push notifications are out of scope.
- [A seat that never submits stalls the match forever] → There's no timeout or forfeit in Phase 2. Recorded as a follow-up.

## Migration Plan

This is new functionality with no existing data. `store_version` 1 is the baseline, and a later format change will bump it and add a migration or refuse to load old data.

## Open Questions

- Whether a forfeit or timeout rule for idle seats is wanted. It can be added later as a new requirement without changing these endpoints.
