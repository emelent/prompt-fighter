# Proposal

## Why

Prompt Fighter's loop is: write a script, fight, **watch what happened**, revise. Players can only read replays today as event text (`pf sim`, `get_round_replay`) or as a 150–380 KB JSON log. That's fine for an LLM, but a human can't see spacing, timing or why a kick whiffed. The event log was designed from the start so that "a renderer never needs game logic" (the replay-log spec): every frame already carries position, state, animation, phase, HP, combo and the script line being executed. This change is **Phase 4: the replay viewer** that renders those logs.

## What Changes

- **New browser replay viewer**, served by `pf serve` under `/view`. No server-side rendering, and no change to the engine or to the log format.
  - `/view` is a home page. It opens a local log file (from `pf sim --json`, `POST /sim` or a saved round) by file picker or drag-and-drop, or a match by id.
  - `/view/matches/{id}/rounds/{n}` plays a stored match round.
  - `/view/matches/{id}` is a **live match page**. It polls the match state, lists the rounds with their results, and automatically opens each new round as it is played.
- **Playback:**
  - **Controls:** play/pause, step one frame, step 10 frames, a scrub bar and speed control (0.25×–4×; 1× = 60 frames per second).
  - **Keyboard shortcuts:** for all of the above.
  - **Shareable frame links:** `#f=412` opens at that frame.
- **On screen:**
  - **The stage:** fighters drawn as simple geometric figures. Each is colour-coded by state, shows facing, jump height, crouch and knockdown poses, and shows attack startup, active and recovery as a limb that grows, shines or fades.
  - **Combat feedback:** flashes for hits, blocks and throws, plus HP bars, combo counters, a frame counter and a result banner.
  - **Event feed:** the same lines as `pf sim`, scrolled to the current frame. Clicking a line jumps to its frame.
  - **Both scripts side by side,** with each fighter's currently executing line highlighted.
- **Security:** all script text and names come from players and are rendered strictly as text. The pages are sent with a restrictive Content-Security-Policy: same-origin scripts only, no inline scripts.
- **Implementation:** **Phaser 3** (pinned at exactly `3.90.0`, using only Phaser 3 APIs, never Phaser 2/CE or 4), **TypeScript in strict mode** and a **Vite** build. The built output (`web/dist`) is embedded in the `pf` binary through a new `web` package, so `go build` still needs no Node. Changing the viewer requires Node and npm.
- **Tests:** the viewer's pure logic is unit-tested with Vitest. `npm run check` type-checks, tests and rebuilds `dist`. The Go test suite runs the frontend tests when `web/node_modules` is installed and skips them otherwise. The viewer's event text is checked against the existing `pf sim` text snapshots, so the two formatters can't drift apart.
- **Docs:** `docs/API.md` gains a "Replay viewer" section, and `docs/ISSUES.md` gets Phase 4 follow-ups.

### Assumptions (defaults chosen without explicit confirmation; easy to revise before apply)
- **Stack:** Phaser 3.90.0, strict TypeScript and Vite, as the Phase 4 plan requires.
- **Hosting:** served by `pf serve`, plus opening local files. There's no standalone static site.
- **Visuals:** geometric figures, with no sprite art.
- **Scope:** core playback, the live match page and frame links. Debug overlays are limited to an optional attack-range display, and a per-frame state inspector is a follow-up.

## Capabilities

### New Capabilities
- `replay-viewer`: the browser viewer. Covers its pages and URLs, loading logs (from a match round or a local file, with format checking), playback controls and keyboard shortcuts, what's drawn per frame, the event feed, script line highlighting, the live match page, frame links, safe handling of player-supplied text, and the security headers.

### Modified Capabilities
- `http-api`: the "JSON transport and error envelope" requirement currently says every endpoint is JSON. It's narrowed to the API endpoints, so that `/view` pages and static assets can be served as HTML, CSS and JS. Unknown routes, `/view` paths included, still get the JSON 404.

## Impact

- New code: `/web`, a Vite + TypeScript project (`package.json` and `package-lock.json` with exact versions, a strict `tsconfig.json`, `vite.config.ts`, `src/*.ts`, Vitest tests, the committed build output `dist/` and `embed.go`), plus the viewer routes in `server/api.go`. A new Go test runs the frontend tests and checks the embedded build.
- Unchanged: the engine, the event log format, the match API, MCP and the config version.
- Dependencies: no new Go modules. The new npm packages are pinned exactly: `phaser` 3.90.0 as the only runtime dependency, bundled into `dist` (about 1.2 MB minified), and the dev dependencies `typescript` 7.0.2, `vite` 8.3.7 and `vitest` 5.1.0.
- Docs: `docs/API.md` and `docs/ISSUES.md`.
- Out of scope: sprite art, sound, editing scripts in the browser (Phase 5 submission UI), mobile-specific layouts beyond a responsive canvas, and recording or exporting video.
