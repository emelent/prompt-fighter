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

### Requirement: Lobby
The lobby SHALL let a player create a match by entering a fighter name (1–40 bytes) and a round-1 script in the script editor, optionally starting from one of the bundled sample bots. After `POST /matches` succeeds, it SHALL store the seat and token and open the play page. It SHALL also let a player open a match by id, which goes to that match's play page.

#### Scenario: Create a match
- **WHEN** a player enters the name `alice`, picks the sample bot `rushdown` as a starting script, and selects "Create match"
- **THEN** the match is created with seat a, and the browser shows `/play/matches/{id}` in the "waiting for opponent" state, with an invite link

#### Scenario: Invalid script blocks creation
- **WHEN** the script in the lobby editor contains `kik`
- **THEN** "Create match" is disabled and the editor marks the error at its line and column

### Requirement: Play page states
The play page SHALL load `GET /matches/{id}` and show exactly one state:

| state | when | what the page offers |
|---|---|---|
| **join** | this browser holds no seat and seat b is free | a name field and an editor; joining calls `POST /matches/{id}/join` and stores the seat |
| **spectator** | this browser holds no seat and both seats are taken | a link to the match in the replay viewer |
| **waiting for opponent** | this browser holds seat a and seat b is empty | the invite link (`/play/matches/{id}`) with a copy button; the editor stays available to replace the round-1 script |
| **your move** | the match is awaiting scripts and this seat has not submitted for the next round | the editor, prefilled with this seat's draft (or last submitted script), and a "Submit for round N" button |
| **waiting for their script** | this seat has submitted and the opponent has not | a waiting message naming the round; the editor stays available, and resubmitting replaces the pending script |
| **finished** | the match is finished | the winner and wins per seat, and links to every round's replay |
| **not found** | the match does not exist | a not-found message with a link back to the lobby |

The page SHALL also show both seats' names and wins, and a list of played rounds with links to their replays.

#### Scenario: Opponent opens the invite link
- **WHEN** a second browser with no stored seat opens the invite link of a match waiting for an opponent
- **THEN** it shows the join state; after joining, round 1 runs and that browser auto-plays the round 1 replay

#### Scenario: Waiting for the other script
- **WHEN** seat a submits its round 2 script and seat b has not
- **THEN** seat a's page shows that it is waiting for the opponent's round 2 script, and seat b's page shows the "your move" state

#### Scenario: Third visitor
- **WHEN** a browser with no stored seat opens the play page of a match whose seats are both taken
- **THEN** it shows the spectator state with a link to `/view/matches/{id}`

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
