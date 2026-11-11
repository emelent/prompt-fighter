# Spec Delta

## MODIFIED Requirements

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
