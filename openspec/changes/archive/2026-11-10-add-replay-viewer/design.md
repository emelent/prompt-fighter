# Design

## Context

- **The event log** (replay-log spec, format `1`) has `frames[i].p1|p2` carrying `x`, `y`, `facing`, `state`, `anim`, `anim_frame`, `phase`, `hp`, `invulnerable`, `combo` and `script_line`. Its `events[]` carry a frame, a type, an actor, an optional target and the detail fields, and `players[]` carry the names and full scripts. A renderer needs no game logic.
- **The server:** `pf serve` (`server/api.go`) registers method-checked routes through `s.route(...)`, plus a JSON-404 catch-all on `/`. `Routes()` feeds the `docs/API.md` sync test, so new routes must be added to that doc's endpoint table.
- **Logs available today:** `GET /matches/{id}/rounds/{n}` returns the exact log bytes, and `GET /matches/{id}` returns the `View` state.
- **Event text:** `sim.EventLines`/`describeEvent` (`sim/textlog.go`) is the canonical event text. Byte-exact snapshots of it for the 6 golden pairs, with and without checks, live in `cmd/pf/testdata/text`, and the matching logs are in `sim/testdata/golden`.
- **Toolchain:** Go 1.27 and Node v24 (npm 11) are present locally. The Phase 4 plan requires **Phaser 3, TypeScript (strict) and Vite**. The latest releases on npm are `phaser` 3.90.0 (the newest 3.x; npm's `latest` tag is already 4.2.1, so the version must be pinned exactly), `typescript` 7.0.2, `vite` 8.3.7 and `vitest` 5.1.0. Node must not become a requirement for `go build` or `go test`, so the built output is committed and embedded. The chrome-devtools MCP tools are available for a visual check during implementation.

## Goals / Non-Goals

**Goals:**
- A viewer that ships inside the `pf` binary, built with Phaser 3, strict TypeScript and Vite, and making no third-party requests. `go build` works from the committed `dist/`, without Node.
- Rendering that is a pure function of (log, frame). The logic can be tested with Vitest without a browser or Phaser.
- Event text that is byte-identical to `pf sim`, enforced by tests.
- Player-supplied text that can never become markup.

**Non-Goals:**
- Sprites, sound, video export, editing scripts in the browser, mobile-specific UI, and a per-frame state inspector (a follow-up).

## Decisions

### D1. Layout and embedding
```
web/
  package.json       exact versions (no ^ or ~): phaser 3.90.0; dev: typescript 7.0.2, vite 8.3.7, vitest 5.1.0
  package-lock.json  committed
  tsconfig.json      "strict": true, noUncheckedIndexedAccess, noImplicitOverride, module/target ES2022, "types": ["vite/client"]
  vite.config.ts     base "/view/static/", build.outDir "dist", assetsDir "" (flat assets), no external CDN
  index.html         Vite entry; <script type="module" src="/src/app.ts">
  src/app.ts         routing by location.pathname, DOM wiring (controls, feed, script panels, match page)
  src/stage.ts       StageScene extends Phaser.Scene: draws buildScene() output, drives the frame clock
  src/replay.ts      pure: validateLog, formatEvent/eventLines, frame clamping, hash parse, speeds
  src/scene.ts       pure: buildScene(log, f, opts) → plain draw description (no Phaser import)
  src/match.ts       pure: matchRows(view), newRoundToOpen(prevView, view)
  src/types.ts       TypeScript types for the event log (format 1) and the match View
  src/style.css
  test/*.test.ts     Vitest suites for the pure modules
  dist/              committed `vite build` output: index.html + hashed .js/.css assets
  embed.go           package web: //go:embed dist
  web_test.go        checks dist; runs `npm test` when web/node_modules exists, else t.Skip
```
Scripts are `npm run dev` (Vite dev server, which proxies `/matches` and `/healthz` to a local `pf serve`), `npm run build` (`tsc --noEmit && vite build`), `npm test` (`tsc --noEmit && vitest run`) and `npm run check` (`tsc --noEmit && vitest run && vite build`). Only Phaser 3 APIs are used (`Phaser.Game`, `Phaser.Scene`, `GameObjects.Graphics`/`Text`, `Scale.FIT`), never Phaser 2/CE or Phaser 4 APIs. The pinned version keeps upgrades deliberate.

`server` imports `web` and serves `web.FS` (rooted at `dist`). The viewer lives in its own `web` package rather than in `server`, so it can be developed on its own with Vite.

Alternative considered: a separate static site served by any web server. Rejected because the user's default (served by `pf serve`) keeps match URLs same-origin, so there's no CORS, and keeps deployment to one binary.

### D2. Routes and headers
These routes are added with the existing `s.route` helper:

| route | serves |
|---|---|
| `GET /view` | `index.html` |
| `GET /view/matches/{id}` | `index.html` |
| `GET /view/matches/{id}/rounds/{n}` | `index.html` |
| `GET /view/static/{file}` | an embedded built asset from `dist/` (unknown file → the existing JSON 404) |

`app.ts` picks the page from `location.pathname`. Vite's `base` makes the built `index.html` refer to `/view/static/<hashed>.js|css`. Every viewer response goes through `viewerHeaders(w)`, which sets:
```
Content-Security-Policy: default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data: blob:; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'
X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
```
Phaser 3 needs no inline scripts or `eval`. `blob:` and `data:` in `img-src` cover its generated textures, and the manual check confirms there are no CSP violations. Content types come from `mime.TypeByExtension`, with `.js` → `text/javascript; charset=utf-8` and `.css` → `text/css; charset=utf-8`. `index.html` is `text/html; charset=utf-8`. The routes get rows in the `docs/API.md` endpoint table, so the routes test stays green.

### D3. Rendering as a pure scene
`buildScene(log, f, {showRanges})` returns plain data:
- the stage bounds
- per fighter: x and ground position, height, `pose` (`stand`/`walk`/`crouch`/`air`/`land`/`down`), `color` (by state), facing, an optional `limb {attack, phase, reach}`, an `alpha` (0.45 if invulnerable), `name`, `hp`, `combo`, `state` and `scriptLine`
- `flashes` for hit, block and throw events on frame `f` (and the next 3 frames, fading)
- `frameLabel` (`f N / last`)
- on the last frame, `banner`

Limb length comes from a static table of attack ranges (stage units), mirroring the default config, and phase styling is startup (thin), active (bright) and recovery (dim). `scene.ts` imports nothing from Phaser.

`StageScene` (in `stage.ts`) owns a `Phaser.GameObjects.Graphics` for shapes and a few `Phaser.GameObjects.Text` objects for names, combo counters, the frame label and the banner. Each frame it clears the graphics and redraws from `buildScene`, mapping stage units to scene pixels. The game is created with `type: Phaser.AUTO` (WebGL with a Canvas fallback), a fixed logical size and `scale: { mode: Phaser.Scale.FIT, autoCenter: Phaser.Scale.CENTER_BOTH }` inside the stage container. No DOM text is created from the scene.

Vitest tests assert scene properties for the spec scenarios: jump kick active is raised with an active limb, knockdown is `down` with alpha < 1, a combo of 3 shows "3 hits", and the last frame has a banner.

### D4. Event text parity
`formatEvent(e)` ports `describeEvent` from Go line for line, and `eventLines(log, includeChecks)` mirrors `EventLines`. A Vitest test loads each `sim/testdata/golden/*.json`, compares `eventLines(log, false)` with the snapshot lines in `cmd/pf/testdata/text/<pair>.txt` minus the final result line, and does the same with checks against `.v.txt`. A Go change to the formatter that skips the TypeScript update then fails the Vitest test, and the snapshot test already guards the Go side.

### D5. Playback
- **Loop:** state is `{frame, playing, speed}`, held in a small playback controller that the DOM controls and `StageScene` share. `StageScene.update(time, delta)` accumulates `delta/1000 × 60 × speed` and advances whole frames, then stops and pauses on the last frame. The DOM controls call the controller, and the scene redraws whenever the frame changes.
- **Speeds:** `[0.25, 0.5, 1, 2, 4]`.
- **Hash:** the `#f=N` fragment is parsed by `parseFrameHash`, clamped with `clampFrame`, and written with `history.replaceState` when paused or after a seek, never per frame while playing.
- **Keyboard:** handlers are ignored while focus is in an input.
- **Accessibility:** controls are `<button>` and `<input type=range>` elements with `aria-label`s.

### D6. Untrusted text
All DOM text goes through `textContent` or `createTextNode`. Script panels are built line by line as `<li>` elements, each with a `textContent`. A Vitest test scans `web/src/*.ts` and fails on `innerHTML`, `outerHTML`, `insertAdjacentHTML`, `document.write` and `eval`/`Function(`. The CSP backs this up in depth.

### D7. Loading and errors
- **Round page:** `fetch('/matches/{id}/rounds/{n}')` and `fetch('/matches/{id}')`, in parallel. The second supplies the seat names and which seat played p1. On a 404, the page shows "Round N has not been played" with a link to the match page.
- **Home page:** a file `<input>` and a drag-and-drop zone read the file with `FileReader`. `validateLog(obj)` returns either an error string or nothing; it checks `format_version === "1"`, two `players`, a non-empty `frames` array, and that `frames[i].f === i` for the first and last frame. Errors render in an alert region as text.

### D8. Live match page
`match.ts` holds the pure helpers, and `app.ts` polls `GET /matches/{id}` every 2 s with `setTimeout` chaining (no overlapping requests) until `status === "finished"`. `matchRows(view)` turns the state into rows of text. `newRoundToOpen(prev, cur)` returns the highest new round number when `cur.rounds.length > prev.rounds.length`. With auto-play on, the page navigates to that round's player, keeping a `?from=match` query so the player shows a "back to match" link. Network errors back off to 5 s and show a notice.

### D9. Tests and verification
- **Vitest (`web/test`):** `validateLog` cases, event parity with the snapshots, `clampFrame`/`parseFrameHash`/speed stepping, `buildScene` scenarios, `matchRows`/`newRoundToOpen`, and the no-HTML-sinks scan.
- **Type checking:** `tsc --noEmit` under `strict` is part of `npm run build`, `npm run check` and `npm test`. Compiler errors fail the build.
- **Go:**
  - `web_test.go` checks that the embedded `dist/index.html` exists, refers only to `/view/static/` assets, and that every asset it refers to is embedded. It runs `npm test` when `web/node_modules` exists, and skips otherwise.
  - `server` tests cover the viewer routes (200 HTML for all three pages, the asset content types, an unknown asset returning JSON 404, the headers on every viewer response, and `GET /view` not being JSON).
  - The `API.md` routes test covers the new routes.
- **Manual (implementation task):** run `pf serve`, play a match, open `/view/matches/{id}/rounds/1` in Chrome via the devtools MCP, take screenshots at a few frames, exercise keyboard and scrubbing, and confirm that the console has no errors and no CSP violations, and that Phaser reports version 3.90.0.

## Risks / Trade-offs

- **[The TypeScript event formatter duplicates Go logic]** → The parity test against the byte-exact snapshots makes any drift fail CI wherever the frontend tests run. RULES.md stays the single description of event detail fields.
- **[The frontend tests are skipped where `web/node_modules` is missing]** → This is documented in `docs/API.md` and `ISSUES.md`, and the developer machine has Node. A CI follow-up should run `npm ci && npm run check` so they always run.
- **[The committed `dist/` can go stale relative to `src/`]** → `npm run check` always rebuilds `dist`, and the final task runs it. The Go test checks that `dist` is self-consistent (every referenced asset is embedded). A CI follow-up should fail when `npm run build` changes `dist`.
- **[Bundle size: Phaser is about 1.2 MB minified]** → It's served from the local binary and cached by the browser. That's acceptable for a local viewer.
- **[Phaser 4 is npm's `latest`, so a careless `npm install phaser` would upgrade it]** → The exact pin in `package.json` and the committed lockfile prevent that, and the manual check confirms that Phaser reports 3.90.0.
- **[Geometric rendering is plain]** → It's intentional for readability. Sprites can be added later behind the same scene description.
- **[Large logs, up to 600 frames]** → Each frame's scene is built on demand, which is cheap. The event feed is built once, and only the current-row marker updates.

## Migration Plan

Additive. No data or API changes, and the only spec relaxation is the JSON-only wording in `http-api`. Developers changing the viewer need Node and npm (`cd web && npm ci`); everyone else builds `pf` from the committed `dist/`. To roll back, remove `web/` and the four routes.

## Open Questions

- Sprite art and a frame-inspector panel can follow without changing the spec's URLs or loading rules.
