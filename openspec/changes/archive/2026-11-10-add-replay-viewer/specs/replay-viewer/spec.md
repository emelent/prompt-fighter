# Spec Delta

## Purpose

Defines the browser replay viewer served by `pf serve`. It renders round event logs frame by frame (stage, fighters, HP, events and the executing script lines), so players can see what their scripts did and why, and it follows live matches as rounds are played.

## ADDED Requirements

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
For the current frame, the stage SHALL draw each fighter from that frame's entry alone, with no game logic:
- Its x position on a 0–100 stage, and its height above the ground from `y`.
- Its facing.
- A pose for its state: standing, walking, crouching for `duck`, airborne, landing, lying for `knockdown` and `ko`, and distinct looks for `hitstun`, `blockstun` and `block`.
- A colour per state.
- During an attack's `anim`, a limb for that attack whose appearance differs between startup, active and recovery.
- A translucent look while `invulnerable`.

It SHALL also draw:
- a brief flash at the fighters on frames with `hit`, `block` or `throw` events
- each player's name and an HP bar
- a combo counter while a fighter's `combo` is 2 or more
- the frame counter (`f N / last`)
- on the last frame, a result banner with the winner, reason and final HP

#### Scenario: Jump kick active
- **WHEN** a frame has p1 `state` `airborne`, `anim` `jump_kick`, `phase` `active` and `y` 10
- **THEN** p1 is drawn raised above the ground, with the jump-kick limb in its active appearance

#### Scenario: Knockdown
- **WHEN** a frame has p2 `state` `knockdown` and `invulnerable` true
- **THEN** p2 is drawn lying down and translucent

#### Scenario: Combo counter
- **WHEN** a frame has p2 `combo` 3
- **THEN** a "3 hits" combo counter is shown for p2

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
