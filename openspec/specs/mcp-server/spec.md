# mcp-server Specification

## Purpose

Defines `pf mcp`, the stdio Model Context Protocol server that lets a player's LLM client validate fight scripts, spar against sample bots, play best-of-3 matches on a `pf serve` server and study replays, with tool results sized and worded so a model can act on them.

## Requirements

### Requirement: Protocol and transport
`pf mcp` SHALL speak MCP over stdio: newline-delimited JSON-RPC messages on stdin and stdout. It SHALL support protocol revision `2026-07-28`, including `server/discover`, which advertises its supported versions, its capabilities (`tools` and `resources`), its identity (name `prompt-fighter` plus a version) and short usage instructions. It SHALL also remain usable by clients that negotiate the earlier revisions supported by the MCP Go SDK v1.8.0, through the `initialize` handshake. It SHALL write nothing to stdout except MCP messages; all diagnostics go to stderr. It SHALL exit with code 0 when stdin is closed.

#### Scenario: Discovery at 2026-07-28
- **WHEN** a client sends `server/discover` with `_meta` protocol version `2026-07-28` over stdio
- **THEN** the result has `resultType` `complete`, lists `2026-07-28` in `supportedVersions`, declares the `tools` and `resources` capabilities, and includes non-empty `instructions`

#### Scenario: Clean stdout
- **WHEN** a client lists tools and calls several of them, some of which fail
- **THEN** every line the server wrote to stdout parses as a JSON-RPC message

#### Scenario: Exit on stdin close
- **WHEN** the client closes the server's stdin
- **THEN** the server process exits with code 0

### Requirement: Tool catalogue
The server SHALL expose exactly these tools: `validate_script`, `simulate_round`, `list_sample_bots`, `get_player_guide`, `create_match`, `join_match`, `submit_script`, `get_match` and `get_round_replay`. `tools/list` SHALL return them in a deterministic order, each with a title, a description written for a model, an input schema that rejects unknown properties, and an output schema. Annotations:
- `readOnlyHint` true for `validate_script`, `simulate_round`, `list_sample_bots`, `get_player_guide`, `get_match` and `get_round_replay`.
- `destructiveHint` false for every tool.
- `openWorldHint` true only for the tools that contact `pf serve`.

List results SHALL carry a non-zero `ttlMs` freshness hint, because the catalogue never changes while the process runs.

#### Scenario: Stable tool list
- **WHEN** a client calls `tools/list` twice
- **THEN** both results contain the same nine tools in the same order

#### Scenario: Unknown argument rejected
- **WHEN** a client calls `validate_script` with `{"script": "kick", "scirpt": "x"}`
- **THEN** the call fails, and nothing is validated

### Requirement: Error reporting
Mistakes a model can fix SHALL be returned as tool execution errors (`isError: true`) with plain-language text. That covers invalid scripts, out-of-range values, unknown sample bots or matches, wrong seat tokens, rejected match actions and an unreachable server. Script validation failures SHALL list each error in the same `line:col: message` form `pf validate` prints. HTTP API errors SHALL include the API's error code and message. Protocol errors (JSON-RPC `error`) SHALL be used only for malformed requests and unknown tools.

#### Scenario: Invalid script in a simulation
- **WHEN** `simulate_round` is called with a script containing `kik` on line 3, column 5
- **THEN** the result has `isError` true, and its text includes `3:5: unknown action "kik" — did you mean "kick"?`

#### Scenario: Server not running
- **WHEN** `get_match` is called while nothing listens at the configured server URL
- **THEN** the result has `isError` true, and its text names the URL and says to start `pf serve` or pass `--server`

### Requirement: validate_script
`validate_script` with `{"script"}` SHALL return `{"valid", "errors": [{"line", "col", "message", "text"}]}`, matching `pf validate`. An invalid script is a successful call, not a tool error: the model asked for validation and got it.

#### Scenario: Valid script
- **WHEN** `validate_script` is called with `loop { kick }`
- **THEN** the result is not an error and its structured content is `{"valid": true, "errors": []}`

#### Scenario: Invalid script
- **WHEN** `validate_script` is called with a script containing `kik`
- **THEN** the result is not an error, `valid` is false, and the error text matches `pf validate`

### Requirement: Sample bots
The server SHALL bundle the repository's sample bots (`scripts/*.pf`). `list_sample_bots` SHALL return `{"bots": [{"name", "summary", "source"}]}` sorted by name. `name` is the file name without `.pf`, and `summary` is the bot's leading comment block with the `#` markers removed.

#### Scenario: List bots
- **WHEN** `list_sample_bots` is called
- **THEN** it returns `ducker`, `jumper`, `lazy`, `rushdown`, `thrower` and `turtle` in that order, each with a non-empty summary and its full source

### Requirement: simulate_round
`simulate_round` SHALL run one round with the default config. Its input is:
- `script`: the caller's script.
- Exactly one of `opponent_script` or `opponent_bot`.
- Optional `side`: `p1` (the default) or `p2`, for the caller.
- Optional `name` and `opponent_name`.
- Optional `frames`: 1–600, default 600.
- Optional `include_checks`: default false.

It SHALL return a replay summary (see "Replay summaries") identifying which side the caller played. It needs no `pf serve` server.

#### Scenario: Spar against a bot
- **WHEN** `simulate_round` is called with a script and `"opponent_bot": "turtle"`
- **THEN** the result's players are the caller as p1 and `turtle` as p2, and its result equals the engine result for those two scripts

#### Scenario: Play the other side
- **WHEN** `simulate_round` is called with `"side": "p2"`
- **THEN** the caller's script plays as p2 and the summary reports `"you": "p2"`

#### Scenario: Ambiguous opponent
- **WHEN** `simulate_round` is called with both `opponent_script` and `opponent_bot`, or with neither
- **THEN** the result has `isError` true and says to give exactly one

#### Scenario: Unknown bot
- **WHEN** `simulate_round` is called with `"opponent_bot": "ninja"`
- **THEN** the result has `isError` true and lists the available bot names

### Requirement: Replay summaries
`simulate_round` and `get_round_replay` SHALL return a replay summary instead of the full frame log. The summary SHALL contain:
- `players`: `[{id, name}]` for p1 and p2.
- `result`: winner, reason, end frame and final HP.
- `stats` per player: hits landed, damage dealt, blocks, whiffs, throws landed, knockdowns suffered, sensor checks, and the largest combo.
- `events`: the event lines exactly as `pf sim` prints them, where `check` events appear only when `include_checks` is true.

At most 200 event lines SHALL be included. When more exist, the summary SHALL include the first 200, set `events_truncated` true, and report `events_total`. The text content SHALL present the same information readably: the result line first, then the stats, then the events.

#### Scenario: Events match pf sim
- **WHEN** a round is summarised without checks
- **THEN** its `events` equal the event lines of `pf sim` for the same scripts, excluding the final result line

#### Scenario: Long replays are capped
- **WHEN** a round produces more than 200 event lines
- **THEN** the summary contains exactly 200 lines, `events_truncated` is true, and `events_total` is the full count

### Requirement: Match tools
The match tools SHALL call the HTTP API of the `pf serve` server at the configured URL, mapping to it as follows:
- `create_match {name, script}` → `POST /matches`
- `join_match {match_id, name, script}` → `POST /matches/{id}/join`
- `submit_script {match_id, token, script}` → `POST /matches/{id}/scripts`
- `get_match {match_id}` → `GET /matches/{id}`
- `get_round_replay {match_id, round, include_checks?}` → the replay summary of `GET /matches/{id}/rounds/{n}`, plus which seat played p1

`create_match` and `join_match` SHALL return the `match_id`, `seat` and `token`, along with the match state. Their descriptions SHALL tell the model that the token is shown only once, that it must be passed to `submit_script`, and that matches do not expire. Every match tool's text content SHALL summarise the state: status, wins per seat, whether each seat has submitted, the round results, and what to do next.

#### Scenario: Two models play a match
- **WHEN** one MCP client calls `create_match` and another calls `join_match` with that `match_id`
- **THEN** the join result shows round 1 played, and `get_match` from either client shows the same state

#### Scenario: Replay of a match round
- **WHEN** `get_round_replay` is called for round 2 of a match where seat b played p1
- **THEN** the summary reports that seat b was p1, and its events equal the `pf sim` lines for that round's stored log

#### Scenario: Wrong token
- **WHEN** `submit_script` is called with a token that does not belong to the match
- **THEN** the result has `isError` true and includes the API code `unauthorized`

#### Scenario: Match tools disabled
- **WHEN** `pf mcp` was started with `--server ""` and `create_match` is called
- **THEN** the result has `isError` true and explains that match tools need a `pf serve` URL

### Requirement: Resources
The server SHALL expose these resources:
- `promptfighter://docs/player-guide`, the player guide, as `text/markdown`.
- `promptfighter://docs/rules`, `RULES.md`, as `text/markdown`.
- One `promptfighter://bots/{name}` resource per sample bot, with its source as `text/plain`.

Reading an unknown resource URI SHALL fail with a resource-not-found protocol error.

#### Scenario: Read the guide
- **WHEN** a client reads `promptfighter://docs/player-guide`
- **THEN** it receives the same text that `get_player_guide` returns

#### Scenario: Read a bot
- **WHEN** a client reads `promptfighter://bots/turtle`
- **THEN** it receives the exact contents of `scripts/turtle.pf`

### Requirement: Local resource limits
Local simulations SHALL run at most N at a time (default: the number of CPUs), and a call SHALL wait for a free slot. Calls to `pf serve` SHALL use a 30-second HTTP timeout, and response bodies SHALL be limited to 4 MiB.

#### Scenario: Slow server
- **WHEN** the configured server accepts a connection but never responds
- **THEN** the match tool returns a tool execution error after the timeout instead of hanging indefinitely
