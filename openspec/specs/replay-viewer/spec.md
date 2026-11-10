# replay-viewer Specification

## Purpose

Defines the browser replay viewer served by `pf serve`. It renders round event logs frame by frame (stage, fighters, HP, events and the executing script lines), so players can see what their scripts did and why, and it follows live matches as rounds are played.

## Requirements

### Requirement: Viewer pages
`pf serve` SHALL serve the viewer at these paths:
- `GET /view` is the home page. It can open a local log file or a match by id.
- `GET /view/matches/{id}` is the live match page.
- `GET /view/matches/{id}/rounds/{n}` is the round player for a stored round.

Each path SHALL return 200 with the viewer HTML (`text/html; charset=utf-8`), whether or not the match exists; the page itself reports missing matches and rounds. Static assets SHALL be served from `GET /view/static/{file}` with correct content types. An unknown asset SHALL return 404. The viewer SHALL load only same-origin resources and SHALL work without network access to any third party.

#### Scenario: Open a round page
- **WHEN** a browser requests `/view/matches/0123456789abcdef/rounds/2`
- **THEN** the response is 200 HTML, and the page fetches `/matches/0123456789abcdef/rounds/2` and `/matches/0123456789abcdef` from the same origin

#### Scenario: Static asset types
- **WHEN** the browser loads the viewer's JavaScript and CSS
- **THEN** they are served with `text/javascript` and `text/css` content types

#### Scenario: Unknown asset
- **WHEN** a browser requests `/view/static/nope.js`
- **THEN** the response is 404

### Requirement: Security of player content
Script source, fighter names and every other value taken from a log or match state SHALL be inserted into the page as text only, never as HTML. Viewer responses SHALL carry a Content-Security-Policy that allows scripts, styles and connections only from the same origin, forbids inline scripts, objects and framing, and sets `base-uri 'none'`. They SHALL also carry `X-Content-Type-Options: nosniff` and `Referrer-Policy: no-referrer`.

#### Scenario: Script containing markup
- **WHEN** a round is viewed whose script contains `# <img src=x onerror=alert(1)>`
- **THEN** the script panel shows that text literally, and no element or handler is created from it

#### Scenario: Headers present
- **WHEN** any `/view` page or asset is requested
- **THEN** the response includes the Content-Security-Policy, `X-Content-Type-Options` and `Referrer-Policy` headers

### Requirement: Loading logs
The round player SHALL load a log from a stored round, or from a local file the user selects or drops onto the page. Local files never leave the browser. Before playing, the viewer SHALL check that the data is a JSON object with `format_version` `"1"`, two players and at least one frame. Otherwise it SHALL show a plain error message that names the problem, such as an unsupported format version or invalid JSON, instead of a blank or broken stage.

#### Scenario: Drop a local log
- **WHEN** the user drops a file produced by `pf sim --json` onto the home page
- **THEN** the round player opens with that log, showing both players' names and scripts

#### Scenario: Unsupported format
- **WHEN** the loaded JSON has `format_version` `"2"`
- **THEN** the viewer shows an error that says format version 2 is not supported by this viewer, which reads version 1

#### Scenario: Round not played
- **WHEN** the round page is opened for a round the server reports as not found
- **THEN** the page says the round has not been played and links to the match page

### Requirement: Playback controls
The round player SHALL show one frame at a time, from frame 0 to the log's last frame. It SHALL offer:
- play/pause
- step back and forward by 1 frame and by 10 frames
- jump to the start and the end
- a scrub bar over every frame
- playback speeds 0.25×, 0.5×, 1×, 2× and 4×, where 1× plays 60 frames per second

Playback SHALL stop on the last frame. Keyboard shortcuts SHALL be:

| key | action |
|---|---|
| Space | play/pause |
| ← / → | 1 frame back / forward |
| Shift+← / Shift+→ | 10 frames back / forward |
| Home / End | first / last frame |
| `[` / `]` | slower / faster |

Every control SHALL have an accessible name.

#### Scenario: Step and scrub
- **WHEN** the user presses → three times from frame 0, then drags the scrub bar to frame 300
- **THEN** the stage shows frame 3, and then frame 300, with the frame counter matching

#### Scenario: Stops at the end
- **WHEN** playback reaches the last frame
- **THEN** it pauses there and the result banner is shown

### Requirement: Frame links
The round player SHALL honour a URL fragment `#f=N` by opening at frame N, clamped to the valid range. While paused, it SHALL keep the fragment equal to the current frame, so the address can be shared.

#### Scenario: Open at a frame
- **WHEN** a round page is opened with `#f=412`
- **THEN** the player starts paused at frame 412

#### Scenario: Out of range
- **WHEN** a round page for a round ending at frame 300 is opened with `#f=999`
- **THEN** the player starts at frame 300

### Requirement: Stage rendering
For the current frame, the stage SHALL draw each fighter from that frame's entry alone, with no game logic, as an animated pixel-art character:
- Its x position on a 0–100 stage, and its height above the ground from `y`.
- Its facing. The sprite is mirrored to face the opponent.
- One character design shared by both players, with distinct colour palettes for p1 and p2 that stay the same in every state.
- A sprite frame chosen from the frame's `state`, `anim`, `phase` and `anim_frame`:
  - every `anim` in format "1" logs (idle, check, wait, walk_forward, walk_back, punch, kick, sweep, throw, jump, jump_kick, dive_punch, landing, duck, low_jab, high_block, low_block, hitstun, blockstun, knockdown, ko) has its own animation
  - an attack's startup, active and recovery phases show different sprite frames, and the active frame shows the attacking limb extended
  - walk and idle animations cycle while the fighter stays in that anim
  - `duck` and `low_jab` are drawn crouched, airborne states raised and tucked, `knockdown` and `ko` lying down, and `hitstun`, `blockstun` and `block` each with their own recognisable pose
  - an unknown `anim` falls back to the standing pose for its `state`, and never fails to draw
- A translucent look while `invulnerable`.
- On the last frame of a round won by KO or timeout, the winner in a win pose.

The stage SHALL have a 16:9 aspect ratio. The art SHALL be pixel art rendered without smoothing at a whole-number scale. All text drawn on the stage SHALL use the viewer's pixel font. The art SHALL be generated by the viewer itself, with no image files or third-party requests.

It SHALL also draw:
- a pixel-art stage backdrop behind the fighters
- a hit spark at the target on frames with `hit` or successful `throw` events, and a different block spark on `block` events, each lasting a few frames
- each player's name and an HP bar
- a combo counter while a fighter's `combo` is 2 or more
- the frame counter (`f N / last`)
- a frame countdown at the top centre showing the frames left in the round, `frame_budget − 1 − f`, as at least three digits. The log's `frame_budget` sets where it starts.
- on the last frame, a result banner with the winner, reason and final HP
- when attack ranges are shown, each attack's reach from the fighter in its facing direction

#### Scenario: Jump kick active
- **WHEN** a frame has p1 `state` `airborne`, `anim` `jump_kick`, `phase` `active` and `y` 10
- **THEN** p1 is drawn raised above the ground, in the jump-kick sprite frame with the leg extended

#### Scenario: Attack phases look different
- **WHEN** p1 throws a kick and the player steps from a `startup` frame to an `active` frame and then to a `recovery` frame
- **THEN** each of the three frames shows a different p1 sprite, and only the active one has the kicking leg extended

#### Scenario: Knockdown
- **WHEN** a frame has p2 `state` `knockdown` and `invulnerable` true
- **THEN** p2 is drawn lying down and translucent

#### Scenario: Players are told apart by palette
- **WHEN** both fighters are in the same state and anim
- **THEN** they are drawn with the same pose, in their own p1 and p2 colours

#### Scenario: Frame countdown
- **WHEN** a log with `frame_budget` 600 is shown at frame 0, at frame 250, and at frame 599 of a round that timed out
- **THEN** the countdown reads 599, 349 and 000 respectively

#### Scenario: Countdown follows a custom budget
- **WHEN** a log produced with `pf sim --frames 300` is shown at frame 0
- **THEN** the countdown reads 299

#### Scenario: Combo counter
- **WHEN** a frame has p2 `combo` 3
- **THEN** a "3 hits" combo counter is shown for p2

#### Scenario: No external art
- **WHEN** a round is played back in a browser with network access to other hosts blocked
- **THEN** the fighters and backdrop draw fully, and the viewer makes no requests for images

### Requirement: Event callouts
The stage SHALL show a short pixel-font callout above the fighters for these events, starting on the event's frame and lasting about half a second (30 frames) of log time:

| event | callout | where |
|---|---|---|
| `block` | `BLOCKED` | above the defender |
| `throw` with outcome `success` | `THROW` | above the thrown fighter |
| `throw` with outcome `tech` | `TECH` | once, between the fighters |
| `trade` | `TRADE` | once, between the fighters |
| `ko` | `K.O.` | large, centred above the result banner, from the event's frame to the end of the log |

A callout SHALL be determined by the current frame alone, so stepping, scrubbing or opening a frame link shows the same callouts as playing up to that frame. Callouts SHALL NOT overlap the result banner, the HP bars or the frame counter. (In format "1" logs the `ko` event is on the last frame, so `K.O.` and the result banner appear together.)

#### Scenario: Blocked punch
- **WHEN** the current frame is 5 frames after a `block` event whose target is p2
- **THEN** `BLOCKED` is shown above p2

#### Scenario: Callout has expired
- **WHEN** the current frame is 40 frames after a `block` event and no other callout event has happened since
- **THEN** no callout is shown

#### Scenario: Scrubbing shows the same callout
- **WHEN** the user drags the scrub bar straight to 10 frames after a `trade` event
- **THEN** `TRADE` is shown, as it would be when playing through that frame

#### Scenario: Knockout
- **WHEN** the player reaches the last frame of a round that ended by KO
- **THEN** a large `K.O.` callout is shown above the result banner, and both are fully readable

### Requirement: Round intro
When playback starts from frame 0 (pressing play at frame 0, replaying from the end, or auto-play), the round player SHALL first show a round intro lasting about 1.5 seconds of wall-clock time at any playback speed:
- both fighters stand in their idle pose at their frame-0 positions
- the round callout is shown first: `ROUND n` for a stored round n, `FINAL ROUND` for round 3, and nothing for a log with no round number, such as a local file
- then `FIGHT!` is shown

Frame 0 then plays as usual. The intro SHALL NOT be part of the timeline: during it the frame counter, scrub bar, script highlights and event feed show frame 0. Seeking, stepping, jumping to the start or end, and pausing SHALL end the intro immediately. Opening a `#f=N` frame link SHALL NOT show the intro.

#### Scenario: Auto-play shows the intro
- **WHEN** the browser opens `/view/matches/{id}/rounds/2?autoplay=1`
- **THEN** `ROUND 2` and then `FIGHT!` are shown over the stage at frame 0, after which frame 1 onwards plays

#### Scenario: Final round
- **WHEN** round 3 of a match is played from frame 0
- **THEN** the intro shows `FINAL ROUND` before `FIGHT!`

#### Scenario: Local file
- **WHEN** a local log file is played from frame 0
- **THEN** the intro shows only `FIGHT!`

#### Scenario: Skipping the intro
- **WHEN** the user presses → during the intro
- **THEN** the intro ends, and the player is paused at frame 1

#### Scenario: Frame link
- **WHEN** a round page is opened with `#f=120`
- **THEN** no intro is shown, and the player starts paused at frame 120

### Requirement: Event feed
The round player SHALL list the round's events, one line per event, with exactly the text `pf sim` prints for that event. `check` events SHALL be hidden unless the user turns them on. Events up to the current frame SHALL be marked, and the feed SHALL keep the latest one in view. Selecting an event line SHALL jump to its frame.

#### Scenario: Lines match pf sim
- **WHEN** a golden-pair log is loaded with checks hidden
- **THEN** the feed's lines equal the event lines of `pf sim` for that pair

#### Scenario: Jump from the feed
- **WHEN** the user selects the line `[f 086] p1 kick hits p2 for 9 (combo 1, hp 91)`
- **THEN** the player moves to frame 86

### Requirement: Script panels
The round player SHALL show both players' scripts with line numbers, labelled with the player's id and name. For the current frame it SHALL highlight each player's `script_line`. When that is 0 (stunned, KO'd or script ended), nothing is highlighted, and the panel shows the fighter's state instead.

#### Scenario: Highlight follows execution
- **WHEN** the current frame has p1 `script_line` 4 and p2 `script_line` 0 with `state` `hitstun`
- **THEN** line 4 of p1's script is highlighted, and p2's panel highlights no line and shows `hitstun`

### Requirement: Live match page
The match page SHALL show the match id and status, each seat's name, wins and whether it has submitted, and every played round with its p1 seat, winner, reason and a link to its round player. While the match is not finished, it SHALL poll `GET /matches/{id}` every 2 seconds. When a new round appears and auto-play is on (the default), it SHALL open that round's player. Polling SHALL stop once the match is finished. An unknown match SHALL show a not-found message.

#### Scenario: Follow a live match
- **WHEN** the match page is open and the opponent's submission causes round 2 to be played
- **THEN** within about 2 seconds the page lists round 2 and, with auto-play on, opens the round 2 player

#### Scenario: Finished match
- **WHEN** the match page loads a finished match
- **THEN** it shows the winner, lists every round, and does not poll

### Requirement: Auto-play and return links
A round player URL with the query parameter `autoplay=1` SHALL start playing from frame 0 as soon as the log has loaded, unless a `#f=N` frame link is present, which takes precedence and opens paused. With `from=play`, the round player's back link SHALL go to `/play/matches/{id}` and read "back to the match". Without it, the back link goes to the viewer's match page, as before.

#### Scenario: Auto-play from the play page
- **WHEN** the browser opens `/view/matches/{id}/rounds/2?from=play&autoplay=1`
- **THEN** playback starts immediately, and the back link points to `/play/matches/{id}`

#### Scenario: Frame link wins over autoplay
- **WHEN** the browser opens `/view/matches/{id}/rounds/2?autoplay=1#f=300`
- **THEN** the player opens paused at frame 300
