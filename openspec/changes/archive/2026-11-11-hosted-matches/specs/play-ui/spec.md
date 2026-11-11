# Spec Delta

## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Play page states
The play page SHALL load `GET /matches/{id}` and show exactly one state:

| state | when | what the page offers |
|---|---|---|
| **join** | this browser holds no seat and a seat is free | a name field and an editor; joining calls `POST /matches/{id}/join` and stores the seat it was given (a or b) |
| **spectator** | this browser holds no seat and both seats are taken | a link to watch the match at `/watch/matches/{id}` |
| **waiting for opponent** | this browser holds a seat and the other seat is empty | the invite link (`/play/matches/{id}`) with a copy button; the editor stays available to replace the round-1 script |
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

## REMOVED Requirements

### Requirement: Lobby
**Reason**: The first screen becomes the cover screen with Create match (hosted, the creator only watches) and Join match. Creating a match as seat a with a name and script from the web page, and the separate watch box, are replaced.
**Migration**: See "Cover screen". Creating a match as seat a remains available through `POST /matches` with a name and script, and through the MCP `create_match` tool. Watching goes through Join match → Watch.
