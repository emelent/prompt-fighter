# Tasks

## 1. Build fixes and Makefile

- [x] 1.1 Fix `.github/workflows/ci.yml`: build the web app (`npm ci`, `npm run build`) before `go vet`/`go test` in the test job and before `go build` in the MCP smoke job; remove the stale-dist check. Verify on a clean clone (`git clone` into the scratchpad, no `web/dist`) that the same steps succeed.
- [x] 1.2 Add `var version = "dev"` and a `version` subcommand (`--version`/`-v` too) to `cmd/pf`, with a test that prints `dev`. Verify `go test ./cmd/pf` passes.
- [x] 1.3 Add the root `Makefile` (design §2: web, build, test, release, smoke, assets, clean; VERSION/LDFLAGS/PLATFORMS). Verify on a clean clone: `make` produces `bin/pf` and `bin/pf version` prints the git-describe version; `make test` passes; `make release VERSION=v0.0.0-test` produces six archives and `SHA256SUMS`; `make smoke` passes; `make clean` removes the outputs.

## 2. Automatic releases

- [x] 2.1 Add `tools/nextversion` (design §4) with table tests for every spec scenario plus scopes, merge commits and non-conventional subjects. Verify `go test ./tools/...` passes and `go run ./tools/nextversion` in this repo (no tags) prints `v0.1.0`.
- [x] 2.2 Rewrite `.github/workflows/release.yml` (design §5: test → version → web → build matrix via `make release` → publish with `target_commitish`; manual `v*` tags; concurrency). Validate the YAML parses and every `make`/script path it calls exists; dry-run the version and build steps locally with the same commands.

## 3. Join without a script

- [x] 3.1 Make the join script optional in `server/matches.go` and `server/match.go` (design §6). Extend `server/hosted_test.go`: two name-only joins → `awaiting_scripts`, no rounds; round 1 runs on the second submission; join with a script still counts as round 1. Verify `go test ./server` passes.
- [x] 3.2 MCP (design §7): `parseMatchRef` returns the base from the link; per-call base in the API client; optional `join_match` script; descriptions and texts; match tools usable with links when no `--server`. Update `matchref_test.go` (link to another server now joins there, using two test servers), add a name-only join test, and export `ToolNames()`. Verify `go test ./mcpserver` passes.

## 4. Plugin

- [x] 4.1 Add `.claude-plugin/marketplace.json` and `plugins/pf/` (plugin manifest, `.mcp.json`, `commands/join.md`, `commands/submit.md`) per design §8, checking the format against the current Claude Code plugin docs. Add `plugins/plugin_test.go` (manifests parse, marketplace source exists, every referenced MCP tool exists). Verify `go test ./plugins` passes, and `claude plugin validate` if the CLI offers it.
- [x] 4.2 Exercise the commands' intended tool sequences against a local `pf serve` with a scripted MCP client (join by link with no script → files written; submit valid script; submit invalid script → no submission), so the instructions in join.md/submit.md describe calls that actually work. Record the transcript in the task notes.
  - Notes (scripted MCP client against `pf serve`, with `pf mcp` started as the plugin does, without `--server`):
    - Two `join_match` calls using the link and no script → seats a and b, "Nothing submitted yet", `awaiting_scripts`, no rounds. Both `.prompt-fighter/match.json` files were written, and both rushdown starters validated.
    - Invalid `fighter.pf` (`kik`) → `validate_script` reports `1:8: unknown action "kik" — did you mean "kick"?`, and seat b stays unsubmitted.
    - Submitting both with the saved link and token → round 1 runs on the second submission.
    - `get_round_replay` by link works.
    - Fixed along the way: join.md now names the sample bot's `source` field.

## 5. Web

- [x] 5.1 Name-only join form on the play page; the waiting-for-opponent state offers the editor with "Submit for round 1" (design §9). Extend `play.test.ts` if state logic changes. Verify `npm run check` passes.
- [x] 5.2 Setup view instructions: plugin install, `/pf:join <game link>`, per-round tweak + `/pf:submit`, PATH note (design §9). Update `spectate.test.ts`. Verify `npm test` passes.

## 6. Docs and end-to-end check

- [x] 6.1 Update README (install `pf` from Releases, `make`, plugin install, `/pf:join`/`/pf:submit` per round, conventional commits for contributors), `docs/API.md` (join without script, MCP links), `docs/PLAYER_GUIDE.md` §11, `docs/ISSUES.md` (remove "dist is committed" wording; follow-ups: HTTP MCP, bundled binaries, path-filtered releases). Verify `go test ./docs` passes.
- [x] 6.2 End to end on a fresh build (`make`): host creates a match in headless Chrome; player 1 joins by name in the web UI and submits round 1 from the editor; player 2 joins with the MCP `join_match` (link, no script) then `submit_script`; round 1 runs only after both submissions and the host plays it. Confirm the setup view shows the plugin instructions. Then `make test` and `make smoke` pass.
