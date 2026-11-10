# Tasks

## 1. Package and serving

- [x] 1.1 Scaffold the `web` Vite + TypeScript + Phaser project (design D1):
  - `package.json` with exact versions (`phaser` 3.90.0; dev: `typescript` 7.0.2, `vite` 8.3.7, `vitest` 5.1.0) and its `dev`/`build`/`test`/`check` scripts
  - a committed `package-lock.json` from `npm install`
  - a strict `tsconfig.json`, and a `vite.config.ts` with `base: '/view/static/'`, flat assets and a dev proxy to `pf serve`
  - an `index.html` entry, `src/app.ts`, `src/style.css`, `src/types.ts`, the built `dist/`, `embed.go` (`//go:embed dist`) and `web_test.go`. The test checks that `dist/index.html` refers only to embedded `/view/static/` assets, and runs `npm test` when `web/node_modules` exists, skipping otherwise.

  Add the four viewer routes and `viewerHeaders` to `server/api.go` (design D2), and add the routes to the `docs/API.md` endpoint table. Add server tests: all three pages return 200 `text/html; charset=utf-8` (not JSON), JS and CSS have the right content types, an unknown asset gives a JSON 404, every viewer response carries CSP, `nosniff` and `no-referrer`, and the round page refers to same-origin paths only. Verify that `npm run check` (in `web/`), `go test ./server -run 'View|Docs'` and `go test ./web` pass, and that `npm ls phaser` prints `phaser@3.90.0`.

## 2. Pure viewer logic (Vitest-tested)

- [x] 2.1 Implement `src/replay.ts`, typed against `src/types.ts`: `validateLog`, `formatEvent`/`eventLines` (a line-for-line port of `sim/textlog.go`), `clampFrame`, `parseFrameHash`/`frameHash`, and the speed list with `slower`/`faster`. Add Vitest tests: `validateLog` (valid, invalid JSON shape, format version 2 naming the version, empty frames, wrong player count); event parity against all 12 snapshots (`cmd/pf/testdata/text/*.txt` with the result line removed, and `*.v.txt` with checks) for the 6 golden logs; and the clamp, hash and speed edge cases. Verify that `npm test` passes.
- [x] 2.2 Implement `src/scene.ts` `buildScene(log, f, opts)` (design D3), with no Phaser import. It covers poses, state colours, limbs by attack and phase, alpha for invulnerability, flashes for hit/block/throw events and the 3 frames after, the combo label when combo ≥ 2, the frame label and the last-frame banner. Add Vitest tests for the spec's stage-rendering scenarios, using frames taken from golden logs (jump kick active, knockdown, combo 3, banner on the last frame), plus flash timing. Verify that `npm test` passes.
- [x] 2.3 Implement `src/match.ts` `matchRows(view)` and `newRoundToOpen(prev, cur)` (design D8), and add the no-HTML-sinks scan test over `src/*.ts` (design D6). Add Vitest tests for waiting, in-progress and finished views, and for new-round detection, including jumping several rounds between polls. Verify that `npm test` passes.

## 3. UI

- [x] 3.1 Implement `src/stage.ts`: a `StageScene extends Phaser.Scene` (Phaser 3 APIs only) that draws `buildScene` output with `Graphics` and `Text` objects, inside a `Phaser.Game` (`Phaser.AUTO`, `Scale.FIT`), with no DOM text. Its `update(time, delta)` drives the shared playback controller at 60 fps × speed and stops on the last frame. Then implement the round player in `app.ts`:
  - loading from a match round (both fetches in parallel, a "not played" message with a match link) and from a local file (picker and drag-and-drop with `validateLog` errors in an alert region)
  - controls with `aria-label`s
  - keyboard shortcuts per the spec table
  - `#f=N` handling
  - the event feed (checks toggle, current marker, click to jump)
  - the script panels (line numbers, highlighted `script_line`, state shown when the line is 0)

  All text goes through `textContent`. Run `npm run build` to refresh `dist/`. Verify that `npm run check` and `go test ./web ./server` pass.
- [x] 3.2 Implement the live match page in `app.ts` with `match.ts`: 2-second polling without overlap, an auto-play toggle (on by default), navigation to new rounds with `?from=match`, a stop once the match is finished, a not-found message, and a network-error backoff. Also implement the home page (open by match id, the local file zone). Rebuild `dist/`. Verify that `npm run check` passes.

## 4. Verification and docs

- [x] 4.1 Do a manual browser check with the chrome-devtools MCP:
  1. Start `pf serve` on a temp data directory and play a match (two seats, sample bots) with `curl`.
  2. Open `/view/matches/{id}/rounds/1` and take screenshots at frame 0, at a mid-fight frame with an attack active, and at the last frame.
  3. Check that the console has no errors or CSP violations, and that `Phaser.VERSION` is `3.90.0`.
  4. Exercise play, stepping, scrubbing, speed, `#f=` and clicking the event feed.
  5. Open a script containing `<img src=x onerror=alert(1)>` and confirm it's shown literally.
  6. Open the match page while submitting another round, and confirm that it auto-opens the new round.
  7. Drop a `pf sim --json` file on the home page.

  Fix anything found, rebuild `dist/`, and report the screenshots.
- [x] 4.2 Update `docs/API.md` with a "Replay viewer" section (the URLs, local files, keyboard shortcuts, frame links, and how to develop it with `cd web && npm ci && npm run dev`) and `docs/ISSUES.md` with a Phase 4 section (what shipped, and the follow-ups: sprites, a frame inspector, `npm ci && npm run check` plus a `dist` freshness check in CI, and a mobile layout). Then run `npm run check` (in `web/`), `go vet ./...` and `go test ./... -count=1`, and confirm all pass.
