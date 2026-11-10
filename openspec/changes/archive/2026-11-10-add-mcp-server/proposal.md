# Proposal

## Why

Prompt Fighter is played by asking an LLM to write and revise fight scripts. Today the LLM can only take part if the player copies scripts, validation errors and replays between the chat and `pf` or `curl` by hand. An MCP server lets the player's own LLM client (Claude Code, Claude Desktop and similar) do the whole loop itself: read the rules, write a script, validate it, spar against the sample bots, play a best-of-3 match on a `pf serve` server, and study the replays.

Players and their LLMs also need a single document that teaches the `.pf` language and the combat rules, written for script authors rather than engine reviewers. That is the player guide. This change is **Phase 3: the MCP server and `PLAYER_GUIDE.md`**.

## What Changes

- **New `pf mcp` command.** It runs a Model Context Protocol server over **stdio**, so the player's MCP client launches it as a subprocess. It's built on the official Go SDK `github.com/modelcontextprotocol/go-sdk` **v1.8.0** and targets MCP protocol revision **2026-07-28**, which is stateless: no `initialize` handshake, and `server/discover` instead. Through the SDK's protocol negotiation, clients on earlier revisions (2025-11-25 and older) keep working. stdout carries only MCP messages; diagnostics go to stderr. The server exits when stdin closes.
- **Tools that work locally, with no server needed:**
  - `validate_script` returns the same errors as `pf validate`.
  - `simulate_round` runs one round, either between two scripts or against a named **sample bot**. It returns a compact, LLM-sized replay summary, not the 150–380 KB frame log.
  - `list_sample_bots` lists the bundled bots with their strategy comments.
  - `get_player_guide` returns the player guide.
- **Match tools that call `pf serve`** (`--server URL`, default `http://127.0.0.1:8080`): `create_match`, `join_match`, `submit_script`, `get_match` and `get_round_replay`. Following the 2026-07-28 guidance on stateful tools, the match id and seat token are explicit handles that the model carries between calls.
- **Resources:** the player guide, the rules reference, and each sample bot's source, for clients that support resources.
- **Errors:** input problems, invalid scripts and server errors are returned as tool execution errors (`isError: true`) whose text the model can act on. An unreachable server gets a message telling the player how to start `pf serve`.
- **New `docs/PLAYER_GUIDE.md`,** covering:
  - the language and the costs of each action and sensor
  - stances, overlays and air zones
  - hits, blocks and throws
  - combos and handlers
  - limits and common validation errors
  - strategy notes drawn from the sample bots
  - how to play through MCP or HTTP

  The guide, `RULES.md` and the sample bots are embedded in the `pf` binary.
- **Refactor:** the `pf sim` text log formatter moves from `cmd/pf` into the `sim` package, so the CLI and MCP produce identical summaries. `pf sim` output doesn't change.

## Capabilities

### New Capabilities
- `mcp-server`: the `pf mcp` stdio MCP server. Covers the protocol revision and compatibility, the tool set with input and output contracts and annotations, replay summaries and their size limits, sample bots, resources, error reporting, the match tools' use of `pf serve`, and stdout/stderr hygiene and shutdown.
- `player-guide`: the content requirements for `docs/PLAYER_GUIDE.md`. It must stay complete (every action, sensor and rule a script author needs) and in sync with the engine vocabulary, which is checked by a test.

### Modified Capabilities
- `pf-cli`: adds the `pf mcp` subcommand (with a `--server` flag). The existing requirements are unchanged.

## Impact

- New code:
  - `/mcpserver` (the MCP server, tools, resources and HTTP client for `pf serve`)
  - `cmd/pf/mcp.go`
  - `sim/textlog.go` (the moved formatter)
  - small embed packages so the docs and sample bots ship inside the binary
- **Dependencies:** this is the first non-stdlib dependency. It adds `github.com/modelcontextprotocol/go-sdk v1.8.0`, and through it `github.com/google/jsonschema-go`, `golang.org/x/oauth2`, `github.com/golang-jwt/jwt/v5`, `github.com/yosida95/uritemplate/v3`, `github.com/segmentio/encoding` and a few `golang.org/x` modules. Only `pf mcp` uses them; the engine and HTTP server stay stdlib-only.
- Docs: new `docs/PLAYER_GUIDE.md`, updated `docs/ISSUES.md`.
- Unchanged: the engine rules, the event log format, the config version, the HTTP API and the store format.
- Out of scope: MCP over HTTP, MCP prompts, the deprecated Logging, Sampling and Roots features, subscriptions (match state is polled with `get_match`), and the replay viewer (Phase 4).
