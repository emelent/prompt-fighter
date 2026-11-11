# play-ui Specification

## Purpose

Defines the browser pages players use to play Prompt Fighter without `curl` or an MCP client: a lobby to create or join best-of-3 matches, and a play page with a CodeMirror script editor (live validation), a waiting state, and automatic replay of each round as it resolves.

## Requirements

### Requirement: Play pages
`pf serve` SHALL serve the play pages from the same embedded web app as the viewer, with the same security headers:
- `GET /play` is the lobby.
- `GET /play/matches/{id}` is the play page for a match.

`GET /` SHALL redirect (302) to `/play`. The pages SHALL use only the existing match API: `POST /matches`, `POST /matches/{id}/join`, `POST /matches/{id}/scripts`, `GET /matches/{id}` and `POST /validate`.

#### Scenario: Root opens the lobby
- **WHEN** a browser requests `/`
- **THEN** it is redirected to `/play`, which returns the web app's HTML with the viewer's Content-Security-Policy

#### Scenario: Play page for any id
- **WHEN** a browser requests `/play/matches/0123456789abcdef`
- **THEN** the response is 200 HTML, and the page itself reports a missing match

### Requirement: Cover screen
The first screen (`/play`) SHALL show the Prompt Fighter cover art and two choices, **Create match** and **Join match**, along with the menu music's sound toggle.
- **Create match** SHALL create a hosted match (`POST /matches` with an empty object) and open its spectator waiting screen, `/watch/matches/{id}`. The creator takes no seat.
- **Join match** SHALL ask for a match id or any link containing one, then load the match and offer **Join as a player**, which opens `/play/matches/{id}`, when a seat is free, and **Watch**, which opens `/watch/matches/{id}`. It reports a clear error for input that is not a match id or link, and for a match that does not exist.

The cover art SHALL be served from the same origin like the other viewer assets, with a text alternative.

#### Scenario: Host a match from the first screen
- **WHEN** a visitor opens `/play` and selects "Create match"
- **THEN** a hosted match is created and the browser shows `/watch/matches/{id}`, waiting for players, with the game link

#### Scenario: Join from a link
- **WHEN** a visitor selects "Join match", pastes `http://host:8080/play/matches/0123456789abcdef` for a match with a free seat, and selects "Join as a player"
- **THEN** the browser opens `/play/matches/0123456789abcdef` in the join state

#### Scenario: Watch from the first screen
- **WHEN** a visitor selects "Join match", enters a match id, and selects "Watch"
- **THEN** the browser opens `/watch/matches/{id}`

#### Scenario: Full match
- **WHEN** the entered match has both seats taken
- **THEN** only "Watch" is offered, with a note that both seats are taken

### Requirement: Play page states
The play page SHALL load `GET /matches/{id}` and show exactly one state:

| state | when | what the page offers |
|---|---|---|
| **join** | this browser holds no seat and a seat is free | a name field; joining calls `POST /matches/{id}/join` with the name only and stores the seat it was given (a or b) |
| **spectator** | this browser holds no seat and both seats are taken | a link to watch the match at `/watch/matches/{id}` |
| **waiting for opponent** | this browser holds a seat and the other seat is empty | the invite link (`/play/matches/{id}`) with a copy button, and the editor with a "Submit for round 1" button (or to replace a round-1 script already submitted) |
| **your move** | the match is awaiting scripts and this seat has not submitted for the next round | the editor, prefilled with this seat's draft (or last submitted script), and a "Submit for round N" button |
| **waiting for their script** | this seat has submitted and the opponent has not | a waiting message naming the round; the editor stays available, and resubmitting replaces the pending script |
| **finished** | the match is finished | the winner and wins per seat, and links to every round's replay |
| **not found** | the match does not exist | a not-found message with a link back to the lobby |

The page SHALL also show both seats' names and wins, a list of played rounds with links to their replays, and to seated players a shareable watch link (`/watch/matches/{id}`) with a copy button.

#### Scenario: Opponent opens the invite link
- **WHEN** a second browser with no stored seat opens the invite link of a match waiting for an opponent
- **THEN** it shows the join state; after joining, round 1 runs and that browser auto-plays the round 1 replay

#### Scenario: Waiting for the other script
- **WHEN** seat a submits its round 2 script and seat b has not
- **THEN** seat a's page shows that it is waiting for the opponent's round 2 script, and seat b's page shows the "your move" state

#### Scenario: Third visitor
- **WHEN** a browser with no stored seat opens the play page of a match whose seats are both taken
- **THEN** it shows the spectator state with a link to `/watch/matches/{id}`

#### Scenario: Share the watch link
- **WHEN** a seated player opens their play page
- **THEN** it shows the watch link `/watch/matches/{id}` with a copy button

#### Scenario: First player to open the game link
- **WHEN** a browser with no stored seat opens the game link of a hosted match with both seats empty and joins
- **THEN** it is given seat a and shows the "waiting for opponent" state

#### Scenario: Join with a name, then write round 1
- **WHEN** a visitor joins with the name `alice` from the game link
- **THEN** the page shows the editor with "Submit for round 1", and round 1 runs once both seats have submitted

### Requirement: Script editor
The editor SHALL be CodeMirror 6, with line numbers, undo history and `.pf` syntax highlighting. Highlighting distinguishes comments, numbers and counts, control keywords, actions, sensors and handler names. The editor SHALL validate the script through `POST /validate`, at most once per 300 ms pause in typing and whenever the script changes. It SHALL mark each error at its line and column with the validator's message. It SHALL show a status line: "valid", or the number of errors, or "checking…". It SHALL show the script's size against the 4096-byte limit. The submit, create and join actions SHALL be disabled until the current text has validated successfully. The editor SHALL work under the viewer's Content-Security-Policy, without loosening it.

#### Scenario: Live error marker
- **WHEN** the player types `punch\n\n    kik` and pauses
- **THEN** within about a second the editor marks line 3, column 5 with `unknown action "kik" — did you mean "kick"?`, and the status line shows 1 error

#### Scenario: Fixing the error enables submit
- **WHEN** the player then changes `kik` to `kick`
- **THEN** the marker disappears, the status shows "valid", and the submit action is enabled

#### Scenario: No CSP violations
- **WHEN** the editor is opened and used
- **THEN** the browser reports no Content-Security-Policy violations

### Requirement: Submitting and waiting
Submitting SHALL call `POST /matches/{id}/scripts` with the stored seat token. It SHALL show any API error as text: an invalid script with its errors, a finished match, or a lost seat. While the match is not finished, the play page SHALL poll `GET /matches/{id}` every 2 seconds without overlapping requests, backing off to 5 seconds on network errors. It SHALL stop polling once the match is finished.

#### Scenario: Submission rejected
- **WHEN** the server rejects a submission with `unauthorized`
- **THEN** the page says that this browser's seat token is not valid for the match, and offers to forget the stored seat

### Requirement: Auto-play on round resolution
When a poll or a submission response shows a round that wasn't there before (the highest new round number if several appear together), the play page SHALL open that round's replay at `/view/matches/{id}/rounds/{n}?from=play&autoplay=1`. There, the replay starts playing immediately, and its back link returns to the play page.

#### Scenario: Round resolves while waiting
- **WHEN** a player is on the "waiting for their script" state and the opponent submits
- **THEN** within about 2 seconds the browser opens the new round's replay, playing from frame 0

#### Scenario: Own submission completes the round
- **WHEN** a player's submission is the second of the round, so the response already contains the new round
- **THEN** the browser opens that round's replay immediately, without waiting for the next poll

### Requirement: Seat tokens and drafts in the browser
After a successful create or join, the page SHALL store `{seat, token, name}` in `localStorage` under a key derived from the match id. It SHALL store the editor's draft per match after each change. Both SHALL be read back on later visits to the same match. A "forget this seat" action SHALL remove the stored seat for that match. If `localStorage` is unavailable, the pages SHALL keep working for the current page view and SHALL say that the seat will not survive a reload.

#### Scenario: Reload keeps the seat
- **WHEN** a player who created a match reloads its play page
- **THEN** the page still knows the player holds seat a, and shows the draft they were editing

### Requirement: Sample bots as starting points
The lobby and the editor SHALL offer the bundled sample bots (`scripts/*.pf`, embedded at build time) as starting scripts. Loading one replaces the editor content, after confirmation if the editor is not empty.

#### Scenario: Load a sample
- **WHEN** a player picks `turtle` from the sample list on an empty editor
- **THEN** the editor contains exactly `scripts/turtle.pf`, and it validates as valid

### Requirement: Menu music
The lobby SHALL play the menu music track on a loop, starting at the first click or key press on the page (browsers block audio until then) and stopping when the visitor leaves the lobby. A labelled **sound** toggle on the lobby SHALL turn the music off and on. It SHALL share the sound preference with the simple round player, so turning sound off in one place turns it off in both. The music SHALL be served from the same origin like the other viewer assets.

#### Scenario: Music in the lobby
- **WHEN** a visitor opens `/play` with sound on and clicks anywhere on the page
- **THEN** the menu music starts and loops until they leave the lobby

#### Scenario: Sound off
- **WHEN** the visitor turns the lobby's sound toggle off
- **THEN** the music stops, and it stays off on the next visit and in simple replays
