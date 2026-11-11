# Tasks

## 1. Server routes

- [x] 1.1 Add `GET /watch/matches/{id}` and `GET /watch/matches/{id}/rounds/{n}` on `handleViewerPage` in `server/api.go`. Extend `server/viewer_test.go`: both return 200 HTML with the viewer headers for any id. Verify `go test ./server` passes.

## 2. Spectator logic (pure)

- [x] 2.1 Create `web/src/spectate.ts` with `waitingState`, `nextToWatch`, `endButton` and `Countdown` (injectable timers) per design §2. Add `web/test/spectate.test.ts`, which verifies `npm test` in `web/` passes. It covers:
  - the waiting-on-one-player scenario (round 2, score 1–0, "waiting for bob")
  - a missing seat b ("waiting for an opponent to join")
  - the finished state
  - `nextToWatch` with and without `after`, including several new rounds at once (lowest first) and joining mid-match (no countdown)
  - `endButton` choosing Next when n+1 is played, Done (to `?after=n`) otherwise, and Done after the deciding round
  - the countdown ticking 5 → 1, firing at 0, and `cancel()` stopping it
- [x] 2.2 Add simple-player settings to `spectate.ts`: `loadSimpleSettings`/`saveSimpleSettings` with the defaults (all panels off, sound on, 0.5×), field-by-field validation, and storage failures falling back to the defaults. Add tests for the defaults, a round trip, invalid stored JSON, and a throwing `localStorage`. Verify `npm test` passes.

## 3. Simple round player

- [x] 3.1 Refactor the controls and panels in `renderPlayer` (`web/src/app.ts`) into the groups from design §3 (`ctl-core`, `ctl-timeline`, `ctl-speed`, `ctl-toggles`, `panel-scripts`, `panel-feed`) without changing the full mode. Verify the existing web tests pass and a `/view` round page looks and works as before in headless Chrome.
- [x] 3.2 Add `ctx.mode` (`'full' | 'simple' | 'spectator'`). In the simple modes:
  - hide the optional groups with the `hidden` attribute
  - add a rewind button (seek to 0, keep playing) next to play/pause
  - add the ⚙ settings popover (labelled checkboxes; closes on Escape or an outside click; returns focus)
  - apply and save the settings and the speed
  - start at the saved speed (0.5× by default) with the sound setting

  Use `'simple'` for `?from=play` round pages. Verify `npm run check` passes.
- [x] 3.3 Add the end-of-replay bar (design §4):
  - from-play mode shows Done → `/play/matches/{id}`
  - spectator mode uses `endButton` with a 5-second countdown to Next round, polls while at the last frame, and cancels on seek or play
  - a "click for sound" hint shows until audio is unlocked

  Verify `npm run check` passes.

## 4. Watch pages

- [x] 4.1 Create `web/src/watch.ts` with the waiting screen (design §5): round, score, seat statuses, played-round links (`/watch/.../rounds/n`), 2-second polling with backoff, the countdown card with Watch now, the finished result, and not found. Route `/watch/matches/{id}` and `/watch/matches/{id}/rounds/{n}` (mode `'spectator'`) in `app.ts`, and add styles in `style.css` (no inline styles). Verify `npm run check` passes.

## 5. Lobby and play page

- [x] 5.1 In `web/src/play.ts`:
  - add the lobby's "Watch a match" card, reusing the id-or-link parser
  - add the watch link and copy button for seated players
  - point the spectator state to `/watch/matches/{id}`

  Extend `web/test/play.test.ts` or `playstate.test.ts` for the parser and the spectator link. Verify `npm test` passes.

- [x] 5.2 Add the lobby menu music (design §6). Extend `web/convert-audio.sh` to produce `src/audio/menu.mp3` and run it. Play the track on a loop from the first gesture, pause it when the lobby is left, and add a lobby sound toggle that shares the simple player's `sound` setting. Add tests for the shared preference, and a server test that the track is served as `audio/mpeg`. Verify `npm run check` and `go test ./server` pass.

## 6. Build, browser check and docs

- [x] 6.1 Rebuild `web/dist` and `pf`, then run a headless-Chrome walkthrough against a local `pf serve` with a fresh data dir. Two players create, join and submit through the API, and a third browser watches. Check each of these and take screenshots:
  - the waiting screen shows "waiting for <name>" and updates when a script arrives
  - the countdown opens round N
  - the simple player shows only play/pause, rewind and settings, at 0.5×
  - turning on "scripts" in settings shows them, and they're still on after a reload
  - at the end, Done returns to the waiting screen, and Next round appears with its countdown when the next round has been played
  - the finished screen
  - a `?from=play` replay ends with Done to the play page
  - `/view` round pages still have the full controls
  - no console errors, and no `POST` from watch pages
  - the lobby starts the menu music after a click, loops it, and stops it when sound is toggled off or the lobby is left
- [x] 6.2 Document the watch pages, joining as a spectator and the simple player's settings in `docs/API.md` (page table and viewer section) and the README. Add follow-ups to `docs/ISSUES.md` (for example, a lobby list of live matches, and spectator counts). Verify `go vet ./...` and `go test ./...` pass.
