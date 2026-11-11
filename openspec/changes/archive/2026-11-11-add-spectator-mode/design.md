# Design

## Context

- One embedded web app serves `/view`, `/play` and their sub-pages, all through `handleViewerPage`. `web/src/app.ts` routes on `location.pathname`.
- `renderPlayer(log, ctx)` builds the round player:
  - header and back link
  - the stage
  - a controls row: ⏮ −10 −1 ▶ +1 +10 ⏭, a scrub bar, the frame counter, speed, and the checks, ranges and sound toggles
  - three panels: both scripts and the event feed

  `ctx` already carries `autoplay`, `backLabel`, `matchLink` and `round`.
- `Playback` (`web/src/playback.ts`) owns frame, speed and the intro. Its speed is set through `setSpeed`.
- `GET /matches/{id}` is public and contains no secrets: seat names, wins, `submitted`, rounds and winner. Tokens are never returned. So spectators need no new API.
- The play page's `spectator` state (`web/src/playstate.ts`) currently links to `/view/matches/{id}`.
- Match polling helpers already exist in `web/src/match.ts`: `POLL_MS` 2000, with backoff on errors.

## Goals / Non-Goals

**Goals:**
- A spectator flow (waiting screen → countdown → simple replay → Next/Done) built from pure, unit-tested state functions, as `playstate.ts` is for the play page.
- One simple round player shared by spectators and by `?from=play` player replays.

**Non-Goals:**
- Listing live matches in the lobby. That needs a new endpoint and stays an open follow-up in ISSUES.
- Spectator counts, chat, or showing players who is watching.
- Changing the `/view` analysis pages or the live match page.
- Server-side push. Polling stays at 2 s.

## Decisions

### 1. Routes reuse the viewer page handler
`server/api.go` adds `GET /watch/matches/{id}` and `GET /watch/matches/{id}/rounds/{n}` on `handleViewerPage`, with the same headers and CSP. `app.ts` routes `/watch/matches/{id}` to `watchPage(id)` in a new `web/src/watch.ts`, and `/watch/matches/{id}/rounds/{n}` to the round loader with `ctx.mode = 'spectator'`.

- **Why:** there's no new server logic, and the CSP and asset handling stay in one place.

### 2. Pure spectator logic in `web/src/spectate.ts`
- `waitingState(view)` returns:
  - `{ kind: 'not_found' }`
  - `{ kind: 'waiting', round, score, seats: [{ seat, name, wins, submitted }], waitingFor: string[], rounds }`
  - `{ kind: 'finished', winner, score, rounds }`
- `nextToWatch(view, watched)` returns the lowest played round number greater than `watched`, or null.
  - On the waiting screen, `watched` comes from `?after=N`, which Done passes as the round just watched.
  - A spectator opening the waiting screen fresh, with `after` absent, is treated as having watched every played round, so it waits for the next one. It shows links to past rounds instead of replaying the whole history.
- `endButton(view, round)` returns `{ kind: 'next', round: n+1 }` if round n+1 is played, `{ kind: 'done', to: '/watch/matches/{id}?after=n' }` otherwise. When the match is finished, Done leads to the result screen.
- `class Countdown`: `start(seconds, onTick, onDone)`, `cancel()`, injectable timers. It counts 5 → 1 and fires `onDone` at 0.
- Settings: `loadSimpleSettings()` / `saveSimpleSettings()` use `{ scripts, feed, timeline, speed: boolean, ranges: boolean, sound: boolean, playbackSpeed: Speed }`. Defaults: all panels off, sound on, `playbackSpeed` 0.5. Storage goes in `localStorage['pf.simple']` as JSON, wrapped in try/catch and validated field by field. Invalid values fall back to the defaults.
- The existing `pf.sound` key (from the sound change) is merged into `sound`: the simple player reads `pf.simple.sound`, and the full player keeps `pf.sound`.

- **Why:** every timing and branch decision is testable without a browser. The DOM code only renders.

### 3. The simple player is a mode of `renderPlayer`, not a second player
`ctx.mode: 'full' | 'simple' | 'spectator'`. `'simple'` is used for `from=play`, and `'spectator'` for watch rounds, which is simple plus the Next/Done logic. In the simple modes:
- The controls row is split into groups, each a container with its own class:
  - `ctl-core`: ⏮ rewind and ▶/⏸, always visible
  - `ctl-timeline`: −10 −1 +1 +10 ⏭, the scrub bar and the frame counter
  - `ctl-speed`
  - `ctl-toggles`: checks and ranges
- Panels: `panel-scripts` (both script panels) and `panel-feed`.
- A ⚙ settings button toggles a popover. It's a `<div role="dialog">` with a checkbox per part, labelled, closes on Escape and on an outside click, and returns focus to the gear.
- A group is hidden with the `hidden` attribute, so no inline styles are needed under the CSP. Hidden groups still work, and their state keeps syncing.
- Rewind is `pb.seek(0)`, and play resumes if it was playing. Seeking to 0 cancels the intro, which is fine: the intro only plays from a fresh `play()`.
- Speed: `pb.setSpeed(settings.playbackSpeed)` before auto-play. Changing the speed select in the simple modes saves `playbackSpeed`.
- Sound: the `soundOn` getter reads the settings' sound toggle.

- **Why:** one player keeps behaviour, such as frame links, the intro and keyboard shortcuts, identical between modes. The analysis viewer path (`mode 'full'`) is untouched.

### 4. End-of-replay overlay
A DOM bar under the stage, `role="status"`, `aria-live="polite"`, appears when `pb.frame === last` and playback has stopped. It contains the Next round or Done button and the countdown text.
- Spectator mode polls `GET /matches/{id}` every 2 s while the bar is shown and recomputes `endButton`. Showing the last frame of a finished match's final round leads to Done.
- From-play mode always shows Done, linking to `/play/matches/{id}`.
- The bar hides and cancels the countdown on any seek or play.

The Next countdown navigates with `location.assign('/watch/matches/{id}/rounds/{n+1}?autoplay=1')`, so each round is its own page load. That gives a fresh stage, a fresh random music track and the intro.

### 5. Waiting screen
`watchPage(id)` renders the round heading, a score line, a seat list ("✓ submitted" or "waiting for …"), and the played-rounds links (to `/watch/.../rounds/n`). It polls with the existing backoff helpers. On `nextToWatch !== null` it shows a countdown card ("Round N starting in 5…") with a **Watch now** button, then navigates. On `finished` it shows the result and stops polling. The page uses the same dark styles as the play page.

### 6. Menu music
The lobby has no Phaser game, so it plays the menu track with a plain `HTMLAudioElement`:
- `loop = true` and volume 0.4.
- Started on the first `pointerdown` or `keydown`, then retried on later gestures if `play()` was rejected.
- Paused when the lobby is left. The lobby is replaced by in-app navigation, so the element is paused and released in the page-change path.

The track comes from `web/src/audio/menu.mp3` (imported as a URL, so Vite fingerprints it and won't inline it). `convert-audio.sh` re-encodes it from `audio/menu.mp3` at 96 kbps. The CSP already allows `media-src 'self'`.

The lobby's sound toggle reads and writes the same `sound` field as the simple player's settings (`pf.simple`).

### 7. Play page and lobby
- **Lobby:** a "Watch a match" card. It reuses the id parser that "Join a match" already uses, which accepts an id or any link containing one.
- **Seated players:** a "Watch link" row (readonly input and a copy button) on the play page, below the score. The existing invite-copy pattern is reused.
- **Spectator state:** links to `/watch/matches/{id}`.

## Risks / Trade-offs

- **[Auto-navigation surprises]** → The countdown is visible and offers Watch now / Next round. Rewinding or playing cancels it on the replay page. The waiting screen's countdown can't be cancelled, since watching is the point of the page, but it is 5 seconds and shown clearly.
- **[Audio needs a user gesture]** → A spectator who opens a watch link and never clicks hears nothing. Autoplay starts silently and the music begins on the first click or key press (Phaser unlocks on the first gesture). The waiting screen's "Watch now" button counts as a gesture, but navigating to the round loads a new page, which needs its own. The round page shows a small "click for sound" hint until audio unlocks.
- **[Settings shared between players and spectators]** → That's intended: they're one viewer's preferences. The `/view` analysis viewer ignores them.
- **[Polling load]** → One `GET` every 2 s per open watch page, the same as the live match page. That's fine for a trusted local server. Public hosting follow-ups already note rate limits.

## Open Questions

- None that block implementation. The exact wording and layout of the waiting screen can be refined in review.
