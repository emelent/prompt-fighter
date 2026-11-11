# Proposal

## Why

Today a third person can only open a full match in the analysis viewer (`/view/matches/{id}`). That viewer is built for debugging scripts: scrub bar, step buttons, speeds, event feed and both scripts. Nothing in it is designed for simply watching a match unfold round by round. Players watching their own rounds from the play page get the same busy view. The user wants a proper spectator experience: join without a seat, wait between rounds on a clear status screen, and watch each round in a simple arcade-style player. Players should get the same simple player.

## What Changes

- **Watch pages:** `GET /watch/matches/{id}` (the waiting screen) and `GET /watch/matches/{id}/rounds/{n}` (a round replay for spectators). They're served by the same embedded app with the same security headers. They need no seat or token, use only `GET /matches/{id}` and `GET /matches/{id}/rounds/{n}`, and can't change the match.
- **Joining as a spectator:**
  - The lobby gets a "Watch a match" box that accepts a match id or a link.
  - The play page shows players a shareable watch link (`/watch/matches/{id}`) with a copy button.
  - A third visitor to a full match's play page is sent to the watch link, not to the analysis viewer.
- **Waiting screen:**
  - It shows the round number, the score (wins per seat with names), each seat's submission status ("submitted" / "waiting for …"), and links to rounds already played. It polls the match every 2 s.
  - When a new round has been played, it shows a countdown, "Round N starting in 5… 4… 3… 2… 1", then opens that round.
  - When the match is finished, it shows the final result.
- **End-of-replay button (spectators):**
  - When the replay reaches its last frame and the next round has already been played, a **Next round** button appears with the same 5-second countdown, then opens that round. Pressing the button skips the countdown.
  - Otherwise a **Done** button returns to the waiting screen.
  - While a spectator sits on a finished replay, the page keeps polling, so the button switches from Done to Next round as soon as the next round is played.
- **Simple round player** for spectators and for players arriving from the play page (`?from=play`):
  - Only play/pause and rewind (back to the start) are visible, plus the end-of-replay button. Players get a **Done** button that returns to their play page.
  - Playback starts at 0.5× with sound on. The scripts, event feed, step buttons, scrub bar, speed control and the checks/ranges toggles are hidden.
  - A **settings** button opens a menu that toggles each hidden part back on (scripts, event feed, timeline and step controls, speed, checks and ranges) plus sound.
  - Choices, including the chosen speed, are remembered per browser.
- **`/view` analysis pages keep their full controls.** The live match page there is unchanged.
- **Menu music:** the lobby (`/play`) plays `audio/menu.mp3` on a loop. It starts on the first click or key press, since browsers block audio until then. A sound toggle on the lobby turns it off, using the same sound preference as the replays.

## Capabilities

### New Capabilities
- `spectator-mode`: watch pages, joining as a spectator, the waiting screen, round-to-round transitions with the countdown, and the finished-match screen.

### Modified Capabilities
- `replay-viewer`: adds the simple round player and its settings menu, used by watch pages and by `?from=play` replays. The return-link requirement changes for `from=play`: it now ends in a Done button.
- `play-ui`: the lobby gets a "Watch a match" box and looping menu music, the play page shows the watch link, and the spectator state points to `/watch/matches/{id}`.

## Impact

- **Server:** `server/api.go` gets two routes on the existing viewer-page handler, plus server tests. No API changes.
- **Web:**
  - `web/src/app.ts`: routing for `/watch/...`; `renderPlayer` gets a `simple` mode with a settings menu and an end-of-replay button.
  - A new `web/src/watch.ts` for the waiting screen.
  - A new pure `web/src/spectate.ts` for the waiting-screen state, countdown, next-round logic and settings persistence, with tests.
  - `web/src/play.ts` for the lobby box, watch link and menu music, and `web/src/style.css`.
  - `web/convert-audio.sh` also converts `menu.mp3`.
  - `web/dist` is rebuilt.
- **Docs:** `docs/API.md` (the page table, plus watching and the simple player) and the README's "Host the game" section. `docs/ISSUES.md` gets follow-ups.
- **No engine, log or MCP changes.**
