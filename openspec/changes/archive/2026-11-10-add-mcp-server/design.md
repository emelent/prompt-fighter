# Design

## Context

- **Engine and server:** the engine (`sim`, `sim/dsl`) and the Phase 2 HTTP server (`server`) are stdlib-only.
- **CLI dispatch:** `cmd/pf/run.go` dispatches subcommands. Its `writeText`/`describe` functions produce the `pf sim` text log, but they live in `package main`, so other packages can't reuse them.
- **Embedding:** the sample bots are in `scripts/*.pf` and the docs in `docs/`. Neither is embedded anywhere.
- **SDK v1.8.0,** verified by downloading it and reading its source and docs:
  - It targets protocol `2026-07-28` (`latestProtocolVersion`) and also negotiates 2025-11-25, 2025-06-18, 2025-03-26 and 2024-11-05.
  - `Server.Connect` registers `server/discover` automatically, built from server info, capabilities and `ServerOptions.Instructions`.
  - `mcp.AddTool[In, Out]` infers input and output schemas from Go types, validates arguments, and turns a returned `error` into `isError: true`. It keeps any `Content` the handler supplies, and fills `StructuredContent` from `Out`.
  - Tools are listed sorted by name.
  - `ServerOptions.SetCacheable` sets `ttlMs` and `cacheScope` on list and read results.
  - `mcp.StdioTransport` serves stdin and stdout, and `mcp.NewInMemoryTransports` supports tests with an SDK client.
  - It requires Go ≥ 1.25; we're on 1.27.
- **The 2026-07-28 spec,** read from modelcontextprotocol.io:
  - It is stateless: no `initialize`, with per-request `_meta` and a required `server/discover`.
  - Every result carries `resultType`, and list results carry `ttlMs`/`cacheScope`. `tools/list` should be deterministic.
  - Logging, Sampling and Roots are deprecated; stdio servers log to stderr.
  - Stateful tools pass explicit handles, with their lifetime stated in the tool description.
  - Fixable problems are tool execution errors.

## Goals / Non-Goals

**Goals:**
- A `mcpserver` package with `New(Options) *mcp.Server`. It's transport-agnostic, so tests drive it over in-memory transports and `pf mcp` runs it over stdio.
- Tool output sized for a model's context: summaries, never raw frame logs.
- One source of truth for text replays: the CLI and MCP share the same formatter.
- Zero behaviour change in the engine, the HTTP API and `pf sim`.

**Non-Goals:**
- MCP over Streamable HTTP, prompts, subscriptions, completion, or the deprecated Logging, Sampling and Roots features.
- Authenticating MCP callers. Seat tokens remain the only secret, as in Phase 2.

## Decisions

### D1. Package layout
- `sim/textlog.go` adds `func (l *Log) EventLines(includeChecks bool) []string` and `func (l *Log) ResultLine() string`. `describe` moves here unchanged. `cmd/pf` `writeText` becomes a thin loop over these. The existing `TestSim*` CLI tests guard that `pf sim` output is unchanged.
- `scripts/embed.go` (package `scripts`) holds `//go:embed *.pf` plus `var FS embed.FS`. `docs/embed.go` (package `docs`) holds `//go:embed PLAYER_GUIDE.md RULES.md`. Adding a `.go` file to `scripts/` doesn't affect `pf selfplay`, which globs `*.pf`.
- `mcpserver/server.go`: `Options{ServerURL string; HTTPClient *http.Client; MaxSims int}` and `New`, which registers tools and resources. It sets `Instructions` and a `SetCacheable` policy (`ttlMs` 3,600,000, `cacheScope` `public`, since everything listed is static).
- `mcpserver/tools_local.go`: `validate_script`, `simulate_round`, `list_sample_bots` and `get_player_guide`.
- `mcpserver/tools_match.go` and `mcpserver/apiclient.go`: the match tools, plus a small client for the Phase 2 API.
- `mcpserver/summary.go`: the replay summary builder and its text rendering.
- `mcpserver/resources.go`: the guide, rules and bot resources.
- `cmd/pf/mcp.go`: `pf mcp` flag parsing and `server.Run(ctx, &mcp.StdioTransport{})`.

Alternative considered: putting the MCP server inside the `server` package so it calls the match logic directly. Rejected because the user chose to route match tools through a `pf serve` URL, so that two players on different clients share matches. A plain HTTP client also keeps the MCP layer decoupled from storage.

### D2. Tool contracts
Each tool is registered with `mcp.AddTool` using typed `In`/`Out` structs with `json` and `jsonschema` description tags, so the input and output schemas are inferred. The implementation task checks that the inferred input schemas set `additionalProperties: false`, and sets it explicitly if the SDK's inference doesn't. Handlers return `(&CallToolResult{Content: []Content{&TextContent{Text: readable}}}, out, nil)` on success. That gives the model prose to read, while `structuredContent` still carries `out` for clients that parse it. Tool errors are returned as `error` values, which the SDK turns into `isError: true`.

| tool | input | readOnly | openWorld |
|---|---|---|---|
| `validate_script` | `script` | ✓ | – |
| `simulate_round` | `script`, `opponent_script` \| `opponent_bot`, `side`?, `name`?, `opponent_name`?, `frames`?, `include_checks`? | ✓ | – |
| `list_sample_bots` | — | ✓ | – |
| `get_player_guide` | — | ✓ | – |
| `create_match` | `name`, `script` | – | ✓ |
| `join_match` | `match_id`, `name`, `script` | – | ✓ |
| `submit_script` | `match_id`, `token`, `script` | – | ✓ |
| `get_match` | `match_id` | ✓ | ✓ |
| `get_round_replay` | `match_id`, `round`, `include_checks`? | ✓ | ✓ |

`destructiveHint` is explicitly false for every tool, and `idempotentHint` is true for the read-only tools. Default names are `you` and `opponent`, or the bot's name.

Descriptions are written for the model. They say what the tool does, when to use it, and the next step: "after `create_match`, share the `match_id` with your opponent; keep the `token`, it is shown only once; matches do not expire".

### D3. Replay summaries
`summarise(l *sim.Log, includeChecks bool) Summary` builds:
- `players`
- `result` (with `winner` as `p1`, `p2` or null)
- `stats[p1|p2]`, counted in one pass over the events:
  - `hits`: hit events by the actor
  - `damage_dealt`: the sum of their damage, plus successful-throw damage
  - `blocks`: block events where the fighter was the defender
  - `whiffs`
  - `throws`: successful throws
  - `knockdowns`: knockdown events on the fighter
  - `checks`
  - `max_combo`: the largest combo value among the actor's hits
- `events`: `l.EventLines(includeChecks)`, capped at 200 lines, with `events_truncated` and `events_total`

`simulate_round` adds `you`. `get_round_replay` adds `round`, `p1_seat` and the seat names. The text rendering is the result line (`l.ResultLine()`, so it's identical to `pf sim`), then one stats line per player, then the events.

### D4. Sample bots
At startup the server reads `scripts.FS`, sorts the bots by name, and validates each one. A bot that fails to validate is a build bug, and a test catches it. `summary` is the leading run of `#` lines, with `# ` stripped and the lines joined by spaces. `opponent_bot` looks a bot up by name, and an unknown name gives an error that lists the valid names.

### D5. Match tools over HTTP
`apiclient` holds the base URL and an `http.Client` with a 30-second timeout. Each call JSON-encodes the request, sets `Authorization: Bearer` for `submit_script`, and reads at most 4 MiB.
- **2xx** responses decode into `server.View` (imported from the `server` package for the types only, so the wire shapes can't drift) or into the raw round log.
- **Non-2xx** responses decode the error envelope into an error. Its text is the API code and message, plus each script error's `text` prefixed by `line:col` (`error code invalid_script: script is invalid\n1:1: unknown action ...`).
- **Connection failures** become `cannot reach pf serve at <url> (<err>). Start it with "pf serve" or run pf mcp with --server <url>.`
- **An empty `ServerURL`** makes every match tool return the "match tools are disabled" error before any I/O.

`get_round_replay` fetches the match state, to learn `p1_seat` and the names, and then the round log, and summarises the log. The text rendering of `View` is shared by every match tool:
- the status
- each seat's name, wins and submitted flag
- one line per round
- a "next:" hint, such as "waiting for opponent to join (share match_id …)", "submit your script for round N", or "match finished: seat b won"

### D6. Resources
- `AddResource` for `promptfighter://docs/player-guide` and `promptfighter://docs/rules` (`text/markdown`).
- One `AddResource` per bot, `promptfighter://bots/<name>` (`text/plain`). A fixed list beats a template here, because `resources/list` then shows every bot.

Handlers return embedded bytes, and an unknown URI returns `mcp.ResourceNotFoundError(uri)`. The SDK maps that to the 2026-07-28 not-found error code.

### D7. Concurrency and limits
A semaphore (`chan struct{}`, size `MaxSims`, default `runtime.NumCPU()`) wraps local `sim.Simulate` calls, and a call waits on it until its context is done. Script size is already bounded by the language limit (4096 bytes). HTTP calls are bounded by the client timeout and the 4 MiB response limit.

### D8. `pf mcp`
- **Flags:** `--server`, default `http://127.0.0.1:8080`. An empty value disables match tools. Otherwise the value must parse with `url.Parse` and have an `http` or `https` scheme and a host, or the command exits 2.
- **stderr:** the startup line is `pf mcp: serving MCP on stdio (pf serve: <url|disabled>)`.
- **Running:** `server.Run(ctx, &mcp.StdioTransport{})` with a signal-aware context. When stdin hits EOF, `Run` returns. A nil, EOF or closed-connection error then exits 0, and any other error exits 1 with the message on stderr.
- **stdout:** nothing in the `pf mcp` path writes to it.

### D9. Tests
- **`mcpserver`, through SDK client sessions over `mcp.NewInMemoryTransports`:**
  - the tool list (names, order, annotations, schemas rejecting unknown properties, `ttlMs > 0`)
  - every tool scenario in the spec
  - resources, including read and unknown URIs
  - error texts
  - the match tools against an `httptest` server built from `server.New(TempDir)`, with two client sessions playing a match
  - the unreachable server, using a closed port
  - the slow server, using a handler that blocks past a shortened timeout
- **`sim`:** `EventLines`/`ResultLine` equal the old CLI output for the golden pairs.
- **`docs`:** the guide sync test (vocabulary from `sim/dsl` and `sim.DefaultConfig()`) and the `pf` code-block validity test.
- **`cmd/pf`, as wire-level tests against the built binary over real pipes:**
  - `server/discover` at `2026-07-28`, checking `resultType`, `supportedVersions`, capabilities and instructions
  - a legacy `initialize` at `2025-11-25` followed by `tools/list`
  - every stdout line being valid JSON
  - exit code 0 after stdin closes
  - `--server ://bad` exiting 2

## Risks / Trade-offs

- **[First third-party dependency, with a transitive tail including `golang.org/x/oauth2` and `jwt`, which we don't use]** → The version is pinned at v1.8.0 in `go.mod`/`go.sum`. Only `pf mcp` imports it, so the engine and HTTP server stay stdlib-only. Recorded in `ISSUES.md`.
- **[MCP 2026-07-28 is new; some clients still speak 2025-11-25]** → The SDK negotiates both eras, and a wire test covers the legacy `initialize` path.
- **[Seat tokens travel through the model's context and the client's transcripts]** → The token is a per-match bearer handle, as the stateful-tools guidance describes. It only lets someone submit for that seat, which is acceptable for trusted, local play. Recorded as a Phase 5 item, alongside accounts.
- **[Replay summaries drop frame-level detail a model might want]** → Event lines carry every hit, block, whiff and throw with its frame. `include_checks` adds sensor reads. The full log remains available over HTTP for the Phase 4 viewer.
- **[Guide drift as rules change]** → The sync and example tests fail CI whenever vocabulary or frame data changes without a guide update.

## Migration Plan

This is additive. `go.mod` gains the SDK requirement, and running `go mod tidy` records the transitive modules. To roll back, remove `mcpserver`, `cmd/pf/mcp.go` and the requirement.

## Open Questions

- Whether to add MCP prompts (for example "write a counter to bot X"), or a `tasks` extension, for long matches. Either can be added later without changing these tools.
