# Tasks

## 1. Shared building blocks

- [x] 1.1 Move the `pf sim` text formatter into `sim/textlog.go` as `(*Log).EventLines(includeChecks bool) []string` and `(*Log).ResultLine() string` (design D1), and make `cmd/pf`'s `writeText` use them. Add a test that, for every golden pair, `EventLines(false)` plus `ResultLine()` equals the existing `pf sim` text output. Verify that `go test ./sim -run TextLog` and `go test ./cmd/pf -run Sim` pass.
- [x] 1.2 Add the embed packages `scripts` (`*.pf`) and `docs` (`PLAYER_GUIDE.md`, `RULES.md`; a placeholder guide is acceptable until task 5.1). Add `github.com/modelcontextprotocol/go-sdk v1.8.0` to `go.mod` and run `go mod tidy`. Verify that `go build ./...` succeeds, that `go list -m github.com/modelcontextprotocol/go-sdk` prints `v1.8.0`, and that `go list -deps ./sim ./server` contains no non-stdlib packages.

## 2. MCP server and local tools

- [x] 2.1 Create `mcpserver` with `Options` and `New`: server info `prompt-fighter`, instructions, the `SetCacheable` policy, and the simulation semaphore. Add the sample-bot loader (sorted, validated, summaries from the leading comments) and the replay summary builder with its text rendering (design D3), including the 200-line cap. Unit-test the summary stats against a hand-checked log, the cap with a synthetic long event list, and bot summaries. Verify that `go test ./mcpserver -run 'Summary|Bots'` passes.
- [x] 2.2 Register `validate_script`, `simulate_round`, `list_sample_bots` and `get_player_guide` with typed In/Out, annotations and model-facing descriptions. Make sure the inferred input schemas reject unknown properties. Add SDK client tests over in-memory transports for each scenario:
  - the stable tool list with its annotations and `ttlMs`
  - an unknown argument being rejected
  - valid and invalid validation
  - listing the bots
  - sparring against a bot, with the result equal to the engine's
  - playing side p2
  - an ambiguous opponent and an unknown bot
  - an invalid script in a simulation returning an `isError` whose text matches `pf validate`
  - events equal to the `pf sim` lines

  Verify that `go test ./mcpserver -run 'Tools|Local'` passes.
- [x] 2.3 Register the resources (the guide, the rules and one per bot) with a not-found error for unknown URIs. Test reading the guide (equal to `get_player_guide`), a bot (equal to the file) and an unknown URI. Verify that `go test ./mcpserver -run Resource` passes.

## 3. Match tools

- [x] 3.1 Implement the API client (design D5): 30-second timeout, 4 MiB limit, error-envelope mapping, the unreachable-server message, and disabled mode. Then implement `create_match`, `join_match`, `submit_script`, `get_match` and `get_round_replay` with the shared match-state text rendering and its "next:" hint. Test against an `httptest` server built from `server.New(t.TempDir())`:
  - two client sessions play a match to the finish
  - a replay of round 2 reports seat b as p1, with events equal to the stored log's lines
  - a wrong token returns `isError` with `unauthorized`
  - disabled mode
  - an unreachable server (a closed port)
  - a slow server timing out, with a shortened client timeout

  Verify that `go test -race ./mcpserver -run Match` passes.

## 4. CLI

- [x] 4.1 Add `pf mcp` (design D8): the `--server` flag and its validation, the stderr startup line, stdio `Run`, and exit codes. Add it to the usage text. Add wire-level tests that build the binary and drive it over pipes:
  - `server/discover` at `2026-07-28` (`resultType`, `supportedVersions`, capabilities, instructions)
  - a legacy `initialize` at `2025-11-25` followed by `tools/list` returning nine tools
  - every stdout line parsing as JSON
  - exit 0 after stdin closes
  - `--server ://bad` exiting 2
  - the default server URL

  Verify that `go test ./cmd/pf -run Mcp` passes.

## 5. Player guide and docs

- [x] 5.1 Write `docs/PLAYER_GUIDE.md` covering every item in the player-guide spec, with a frame-data table whose rows are formatted `| name | startup | active | recovery | …`, `pf`-tagged examples that validate, and mistake examples using another tag. Add the tests in `docs/`: the guide sync test (actions, sensors, `in_range` attacks, frame data from `sim.DefaultConfig()`, and the limits) and the examples-validity test. Verify that `go test ./docs` passes.
- [x] 5.2 Update `docs/ISSUES.md` with a Phase 3 section: what shipped, the dependency note, and the follow-ups (MCP over HTTP, MCP prompts, push updates for match state, and seat tokens in model context, linked to Phase 5 auth). Add a short "Playing with an LLM" section to `docs/API.md`, pointing to `pf mcp` and the guide, with a Claude Code / Claude Desktop config snippet. Verify by reading both files for accuracy against the implemented flags and tool names.

## 6. Integration

- [x] 6.1 Run `go vet ./...`, `go test ./... -count=1` and `go test -race ./mcpserver ./server ./sim -count=1`, and confirm all pass. Then do a manual end-to-end smoke run:
  1. Start `pf serve` on a temp data directory.
  2. Run two `pf mcp` processes against it, driven by a small JSON-RPC script over pipes: discover, then `create_match` from one, `join_match` from the other, then submit scripts until the match finishes.
  3. Call `get_round_replay`.
  4. Close stdin on both and stop the server.

  Report the transcript.
