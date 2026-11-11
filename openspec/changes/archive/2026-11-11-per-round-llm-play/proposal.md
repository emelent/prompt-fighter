# Proposal

## Why

**The GitHub builds are broken.** `web/.gitignore` ignores `dist/`, so the built web app (`web/dist`) was never committed. `//go:embed dist` in `web/embed.go` then fails in every CI and release job (`pattern dist: no matching files found`), even though it works locally. The build pipeline and docs assumed `web/dist` was committed, and the repo says otherwise.

**Releases are manual.** There is no single command that builds everything, and releases need a hand-pushed tag. The user wants each push to main to be versioned and released automatically, with semver derived from the commit messages.

**LLM players play the whole match unattended.** The LLM is told to join and keep submitting scripts until the match is over. The user wants the human to stay in the loop: the LLM only joins at first, the player prompts it each round to study the replay and tweak the script, and an explicit `/pf:submit` sends that round's script.

## What Changes

- **The web build is generated, not committed:**
  - `web/dist` stays gitignored, as it is today.
  - Every Go build path (Makefile, CI, release) builds the web app first.
  - The CI step that checked for a stale committed `web/dist` is removed.
  - Docs that say "dist is committed" are corrected.
- **A `Makefile`** at the root. `make` builds everything into `bin/pf`, building the web app first. Targets:
  - `web`, `build`, `test` (go vet, go test, web tests)
  - `release` (cross-platform archives in `dist/`)
  - `smoke` (MCP smoke test)
  - `assets` (re-encode audio and the cover; needs ffmpeg)
  - `clean`
- **Version stamping:** `pf version` prints the build version, stamped at link time (`dev` when not stamped).
- **Automatic releases on GitHub:** on every push to main, after the tests pass, a tested Go tool (`tools/nextversion`) computes the next semver from conventional commits since the last `v*` tag:
  - `feat` bumps minor
  - `fix` and everything else bumps patch
  - `!` or `BREAKING CHANGE` bumps major, or minor while below 1.0
  - with no new commits, nothing is released

  The same workflow run then tags `vX.Y.Z`, builds the six platform archives with the version stamped in, and publishes the GitHub release with `SHA256SUMS`. Tags pushed by `GITHUB_TOKEN` don't trigger other workflows, so tagging and building happen in one run. Pushing a `v*` tag by hand still builds that release.
- **Joining takes no script:**
  - `POST /matches/{id}/join` with only a name takes a seat with nothing submitted.
  - Every round, including round 1, runs when both seats have submitted.
  - Joining with a script still works, and the script counts as the round-1 submission, so existing MCP and API clients work unchanged.
  - The web play page's join form asks for only a name, and the player then writes and submits round 1 in the editor.
- **A Claude Code plugin `pf`** in this repo (`plugins/pf`, with a marketplace manifest at `.claude-plugin/marketplace.json`). It bundles the prompt-fighter MCP server and adds two commands:
  - `/pf:join <game link> [name]`: joins only. It takes a seat with a name and saves the link, seat and token to `.prompt-fighter/match.json`. It creates `fighter.pf` to edit (unless one exists), validates it, and tells the player how to continue. It does not submit.
  - `/pf:submit`: validates `fighter.pf` and submits it for the next round, exactly as it is. It never edits the script. On validation errors it reports them and stops. It then reports the match state.

  Between rounds, the player prompts the LLM in plain words ("study round 1 and tweak fighter.pf to beat their sweeps").
- **MCP follows the game link:** match tools use the server named in a game link, rather than refusing links that differ from `--server`, so one installed MCP server works with any host. Bare ids still use `--server`. `join_match`'s `script` becomes optional.
- **The host's waiting screen** shows the new instructions: install the plugin (two commands), run `/pf:join <game link>`, then each round tweak the script and `/pf:submit`.
- **Docs:** README, API, player guide and ISSUES.

## Capabilities

### New Capabilities
- `build-and-release`: the Makefile targets, the web build as part of every Go build, version stamping, semver from conventional commits, and the automatic release workflow.
- `claude-plugin`: the `pf` plugin and marketplace manifest, and the behaviour of `/pf:join` and `/pf:submit`.

### Modified Capabilities
- `match-play`: "Create and join" changes so joining needs only a name and round 1 waits for both submissions.
- `mcp-server`: "Match tools" changes so game links use their own server and `join_match`'s script is optional.
- `play-ui`: "Play page states" changes so the join state asks for a name only.
- `spectator-mode`: "Waiting screen" changes so the setup view's LLM instructions use the plugin commands.
- `pf-cli`: adds `pf version`.

## Impact

- **Build:**
  - new `Makefile` and `tools/nextversion` (Go, with tests)
  - `.github/workflows/ci.yml`: builds web before Go; the stale-dist check is removed
  - `.github/workflows/release.yml`: rewritten
  - `cmd/pf`: `version` command and `-X` stamping
- **Server:**
  - `server/matches.go`: script optional on join
  - `server/match.go`: `join` with a nil pending script, and the status after the second join
  - tests and `docs/API.md`
- **MCP:**
  - `mcpserver`: per-call server from game links, optional join script, descriptions
  - tests
- **Plugin:** new `plugins/pf/` (`.claude-plugin/plugin.json`, `.mcp.json`, `commands/join.md`, `commands/submit.md`) and `.claude-plugin/marketplace.json`, with a Go test checking the manifests and commands.
- **Web:**
  - `web/src/play.ts`: the join form asks for a name only, then the editor
  - `web/src/spectate.ts` and `web/src/watch.ts`: the plugin instructions
  - tests
- **Docs:** README, `docs/API.md`, `docs/PLAYER_GUIDE.md`, `docs/ISSUES.md`.
- **Compatibility:**
  - Joining with a script behaves as before.
  - Matches already stored are unaffected.
  - A web player who joins by name and closes the tab still has their seat, as with any stored seat.
