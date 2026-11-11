# Spec Delta

## ADDED Requirements

### Requirement: Menu music
The lobby SHALL play the menu music track on a loop, starting at the first click or key press on the page (browsers block audio until then) and stopping when the visitor leaves the lobby. A labelled **sound** toggle on the lobby SHALL turn the music off and on. It SHALL share the sound preference with the simple round player, so turning sound off in one place turns it off in both. The music SHALL be served from the same origin like the other viewer assets.

#### Scenario: Music in the lobby
- **WHEN** a visitor opens `/play` with sound on and clicks anywhere on the page
- **THEN** the menu music starts and loops until they leave the lobby

#### Scenario: Sound off
- **WHEN** the visitor turns the lobby's sound toggle off
- **THEN** the music stops, and it stays off on the next visit and in simple replays

## MODIFIED Requirements

### Requirement: Lobby
The lobby SHALL let a player create a match by entering a fighter name (1–40 bytes) and a round-1 script in the script editor, optionally starting from one of the bundled sample bots. After `POST /matches` succeeds, it SHALL store the seat and token and open the play page. It SHALL also let a player open a match by id, which goes to that match's play page, and SHALL have a **Watch a match** box that accepts a match id or any link containing one and opens `/watch/matches/{id}`.

#### Scenario: Create a match
- **WHEN** a player enters the name `alice`, picks the sample bot `rushdown` as a starting script, and selects "Create match"
- **THEN** the match is created with seat a, and the browser shows `/play/matches/{id}` in the "waiting for opponent" state, with an invite link

#### Scenario: Invalid script blocks creation
- **WHEN** the script in the lobby editor contains `kik`
- **THEN** "Create match" is disabled and the editor marks the error at its line and column

#### Scenario: Watch from the lobby
- **WHEN** a visitor pastes `http://host:8080/play/matches/0123456789abcdef` into "Watch a match"
- **THEN** the browser opens `/watch/matches/0123456789abcdef`

### Requirement: Play page states
The play page SHALL load `GET /matches/{id}` and show exactly one state:

| state | when | what the page offers |
|---|---|---|
| **join** | this browser holds no seat and seat b is free | a name field and an editor; joining calls `POST /matches/{id}/join` and stores the seat |
| **spectator** | this browser holds no seat and both seats are taken | a link to watch the match at `/watch/matches/{id}` |
| **waiting for opponent** | this browser holds seat a and seat b is empty | the invite link (`/play/matches/{id}`) with a copy button; the editor stays available to replace the round-1 script |
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
