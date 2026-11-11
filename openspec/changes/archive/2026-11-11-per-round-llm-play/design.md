# Design

## Context

- `web/embed.go` has `//go:embed dist`. `web/.gitignore` ignores `dist/`, so a clean checkout can't compile any package that imports `promptfighter/web` (`server`, `cmd/pf`) until `npm run build` has run. `.github/workflows/ci.yml` runs `npm ci` but never builds the web app before `go vet` and `go test`. It also has a "web/dist is stale" step that only made sense if `web/dist` were committed. `release.yml` builds Go directly on a tag push. Both fail on GitHub.
- The repo is `github.com/emelent/prompt-fighter`, branch `main`. It has no tags yet.
- `cmd/pf/run.go` dispatches subcommands and has no version.
- Server: `handleJoin` decodes `seatReq{Name, Script *string}` and `check()` requires the script. `Match.join(name, script, hash)` stores `Pending: &script`. `readyToPlay()` already needs both pending scripts, and `submit()` works for any seat that isn't finished.
- MCP: `apiClient{base}` is fixed from `--server`. `parseMatchRef(ref, server)` errors when a link's origin differs from `--server`. `join_match` requires `script`.
- Web: the play page's join state shows a name field and the editor, and posts both. The setup view's LLM text uses `claude mcp add … --server`.

## Goals / Non-Goals

**Goals:**
- A clean clone builds with `make`, and GitHub builds pass, without committing generated files.
- Hands-off semver releases from conventional commits, using only first-party GitHub actions and in-repo code.
- Per-round, human-driven LLM play through two slash commands. The LLM never submits on its own.

**Non-Goals:**
- Serving MCP over HTTP from `pf serve`, or bundling binaries in the plugin. Both are noted as follow-ups.
- Changing the API for players who still join with a script.
- Commit-message linting or changelog files. Release notes come from GitHub's generator.

## Decisions

### 1. `web/dist` is a build product everywhere
Keep the gitignore. The Makefile's `build` and `test` depend on `web`.
- CI does `npm ci` and `npm run build` before `go vet` and `go test`, and drops the stale-dist step.
- The release workflow builds the web app once in a `web` job, uploads `web/dist` as an artifact, and each platform build downloads it. This avoids running npm six times.
- `web/web_test.go` keeps its consistency check, which now runs against the freshly built dist.

- **Alternative:** committing `web/dist`. Rejected: generated churn in every diff, and the user's gitignore says no.

### 2. Makefile
A POSIX `make` file with no GNU-only functions beyond `$(shell)`.

| variable | default |
|---|---|
| `VERSION` | `$(shell git describe --tags --always --dirty 2>/dev/null \|\| echo dev)` |
| `LDFLAGS` | `-s -w -X main.version=$(VERSION)` |
| `PLATFORMS` | `linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64` |

Targets:
- `web` runs `npm ci`, only when `web/node_modules` is missing or older than `package-lock.json`, then `npm run build`.
- `build` produces `bin/pf` with `CGO_ENABLED=0 go build -trimpath -ldflags`.
- `test` runs `web`, then `go vet ./...` and `go test ./...` (which includes the web tests through `go test ./web`).
- `release` loops over `PLATFORMS`, writes `dist/pf-$(VERSION)-$os-$arch/{pf[.exe],README.md}`, archives it as zip or tar.gz, and writes `SHA256SUMS`, using `shasum -a 256` or `sha256sum`, whichever exists.
- `smoke` builds `bin/pf`, starts `bin/pf serve` on a free port with a temp data dir, runs `.github/scripts/mcp-smoke.sh`, and kills the server.
- `assets` runs `npm run assets` in `web/`.
- `clean`.

The release workflow calls `make release` per platform with `PLATFORMS` set to one entry, so local and CI archives come from the same code.

### 3. Version stamping
`cmd/pf/main.go` gets `var version = "dev"`. The `version` subcommand prints it, and `pf --version` and `pf -v` do too. The `usage` text lists `pf version`.

### 4. `tools/nextversion` (Go)
- **Pure core:** `Next(lastTag string, commits []Commit) (next string, ok bool)`, with `Commit{Subject, Body string}`.
- **Header parsing:** headers match `^(\w+)(\([^)]*\))?(!)?:`. `BREAKING CHANGE:` or `BREAKING-CHANGE:` at a line start in the body counts as breaking.
- **Bump rules:**
  - major bump when breaking and the major version is ≥ 1
  - minor when breaking and the major version is 0, or when any commit is `feat`
  - otherwise patch
  - no commits → `ok=false`
  - with no last tag, the next version is `v0.1.0`
- **`main`:** runs `git describe --tags --abbrev=0 --match 'v[0-9]*'` (no tag → empty), then `git log <tag>..HEAD --format=%s%x1f%b%x1e` (or the full history), and prints the next version or nothing. Its exit code is 0 either way.
- **Tests:** a table test covers the spec scenarios, plus scopes (`feat(web):`), case, merge commits (patch), and a non-conventional subject (patch).
- **Why Go:** `go test ./...` already runs everywhere, so there's no bats or shell test harness, and no third-party action.

### 5. Release workflow (`.github/workflows/release.yml`)
Triggers: `push` to `main` and `push` of tags `v*`, with `permissions: contents: write` and a `concurrency: release` group (no cancelling).

Jobs:
- **`test`:** checkout, setup-go and node, `make test`.
- **`version`** (needs test):
  - On a tag push, the version is the tag name.
  - On main, checkout with `fetch-depth: 0` and `fetch-tags`, then run `go run ./tools/nextversion`.
  - Output `version`, empty when there's nothing to release.
- **`web`** (if there's a version): `make web`, then upload `web/dist`.
- **`build`** (matrix, if there's a version): download `web/dist`, then `make release VERSION=… PLATFORMS=os/arch`, then upload the archive.
- **`publish`:** download the archives, write `SHA256SUMS`, then `softprops/action-gh-release@v2` with `tag_name: <version>`, `target_commitish: ${{ github.sha }}` and `generate_release_notes`. Creating the release creates the tag on that commit, so no separate tag push is needed.

`ci.yml` keeps running on PRs and pushes. Its fixed MCP smoke job now builds the web app first too.

### 6. Server: optional join script
- `seatReq.Script` stays a pointer.
- In `handleJoin`, the name is validated as before. The script is validated only when present.
- `Match.join(name string, script *string, hash string)` sets `Pending = script`, which may be nil.
- The status after filling the second seat becomes `awaiting_scripts` whether or not scripts are present. That's the same transition as today. `playIfReady` then runs round 1 only if both are pending.
- `handleCreateMatch` is unchanged: a player who creates still needs both name and script, and a host sends neither.

### 7. MCP: the server comes from the game link
- `parseMatchRef(ref, server)` returns `(id, base string, err)`. A link yields its origin as the base. A bare id yields the configured server, or an error when there isn't one: "this pf mcp has no server configured; pass the game link instead of the bare id".
- `apiClient.doJSON` and `do` take the base per call. The client keeps `http` only, and `base` becomes the default. Match tools are no longer disabled outright when `--server ""`: they work with links.
- `create_match` uses the configured server only.
- `JoinIn.Script` becomes `omitempty` and optional in the schema. The description says to join with just a name when the player will submit later.
- The result text for a join with no script says: "nothing submitted yet; submit round 1 with submit_script".

- **Trust:** the tool contacts whatever origin the model passes. That's acceptable because the user gives the link. The tool only ever calls `/matches/...` paths and returns their JSON, and the risk is noted.

### 8. The `pf` plugin
Layout:

```
.claude-plugin/marketplace.json    {"name":"prompt-fighter","owner":{"name":"emelent"},"plugins":[{"name":"pf","source":"./plugins/pf","description":…}]}
plugins/pf/.claude-plugin/plugin.json   {"name":"pf","description":…,"version":…}
plugins/pf/.mcp.json               {"mcpServers":{"prompt-fighter":{"command":"pf","args":["mcp"]}}}
plugins/pf/commands/join.md
plugins/pf/commands/submit.md
```

Each command has frontmatter (`description`, `argument-hint`, and `allowed-tools` listing the plugin's MCP tools plus Read/Write). The body is precise instructions for the model:

- **join.md:**
  1. Parse `$ARGUMENTS` into a link and an optional name; ask if the link is missing.
  2. Call `get_player_guide` if the model has not read it this session.
  3. Call `join_match` with no script.
  4. Write `.prompt-fighter/match.json`.
  5. Create `fighter.pf` if missing, from `list_sample_bots`' rushdown or a short starter, and run `validate_script`.
  6. Report, and finish with explicit instructions: "Don't submit. Wait for the player to ask for changes, then for /pf:submit."
- **submit.md:**
  1. Read both files.
  2. If `match.json` is missing, tell the player to run `/pf:join` and stop.
  3. Run `validate_script`. On errors, list them and stop without editing.
  4. Run `submit_script` with the link and token.
  5. Run `get_match`, and report who has submitted. If a round just ran, give its result and suggest the player ask for an analysis with `get_round_replay`.
  6. Never modify `fighter.pf`.

MCP tool names in a plugin are namespaced as `mcp__plugin_pf_prompt-fighter__<tool>`. A Go test (`plugins/plugin_test.go` in package `plugins`) parses both manifests and the frontmatter. It checks that every `mcp__plugin_pf_prompt-fighter__*` name is a tool in `mcpserver`'s catalogue, using a small exported `mcpserver.ToolNames()`.

The exact plugin schema is checked against the current Claude Code docs during implementation (`claude plugin validate`, if available).

### 9. Web
- **Join state:** a name field and a Join button that posts `{name}`. After joining, the page shows the waiting-for-opponent or your-move state with the editor and "Submit for round 1".
- **`maybeAutoplay` and submissions:** unchanged.
- **Setup view:** `setupInfo()` drops `mcpCommand` and `llmPrompt` and adds:
  - `pluginInstall`, two lines: `/plugin marketplace add emelent/prompt-fighter` and `/plugin install pf@prompt-fighter`
  - `joinCommand`: `/pf:join <game link>`
  - `perRound`: the tweak-then-`/pf:submit` explanation
  - a note that the player needs `pf` on PATH (from GitHub Releases)

## Risks / Trade-offs

- **[`pf` must be on PATH for the plugin]** → Documented, with the `claude mcp add … /abs/path/pf mcp` fallback. Removing it needs HTTP MCP or bundled binaries (follow-ups).
- **[Commit messages that aren't conventional]** → They count as patch bumps, so releases still happen. Contributors are told the rules in the README.
- **[Release on every push to main]** → Intended by the user. Docs-only commits also release a patch (no path filter). That's fine for now, and listed as a follow-up.
- **[Plugin command instructions are prompts, not code]** → Models may deviate. The commands are explicit and short, and `/pf:submit`'s "never edit" rule is repeated. The local files make the state inspectable.
- **[The MCP server talks to any origin in a link]** → Only `/matches/...` endpoints, initiated by the user's own prompt. Noted in docs.

## Migration Plan

- After merge, the first push to main releases `v0.1.0`.
- Existing matches and clients are unaffected.
- Players install the plugin to get the new commands. The old `claude mcp add` setup keeps working.
