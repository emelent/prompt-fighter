# match-play Specification

## Purpose

Defines best-of-3 matches played over HTTP between two seats that may revise their fight scripts between rounds, including seat tokens, round execution, the win, draw and round-cap rules, what each seat may see, and persistence across server restarts.

## Requirements

### Requirement: Create and join
`POST /matches` with `{"name", "script"}` SHALL create a match with the caller in seat `a`. It SHALL return 201 with `{"match_id", "seat": "a", "token", "match": <state>}`. The match starts in status `waiting_for_opponent`, and seat a's script counts as its submission for round 1. `POST /matches/{id}/join` with `{"name", "script"}` SHALL put the caller in seat `b` and return 201 with `{"match_id", "seat": "b", "token", "match": <state>}`. Both seats then have a submission, so round 1 runs before the join response is returned. Names are 1–40 bytes. Scripts MUST validate, or the request fails with 422 `invalid_script`.

Match ids SHALL be 16 lowercase hex characters, and seat tokens 32 lowercase hex characters, both from a cryptographically secure random source. Tokens are returned only once, in the create or join response.

#### Scenario: Create a match
- **WHEN** a client posts a valid name and script to `/matches`
- **THEN** the response is 201 with seat `a`, a 32-hex-character token, and a match whose status is `waiting_for_opponent` with seat b empty

#### Scenario: Join runs round 1
- **WHEN** a second client posts a valid name and script to `/matches/{id}/join`
- **THEN** the response is 201 with seat `b`, and the returned match has one played round with seat a as p1

#### Scenario: Seat already taken
- **WHEN** a client joins a match whose seat b is already filled
- **THEN** the response is 409 `conflict`

#### Scenario: Invalid script on create
- **WHEN** a client creates a match with a script containing `kik`
- **THEN** the response is 422 `invalid_script` and no match is created

### Requirement: Seat tokens
Requests that act for a seat SHALL carry `Authorization: Bearer <token>`. A missing or unrecognised token SHALL be rejected with 401 `unauthorized`. The server SHALL persist only a SHA-256 hash of each token, never the token itself.

#### Scenario: Missing token
- **WHEN** a client posts to `/matches/{id}/scripts` without an `Authorization` header
- **THEN** the response is 401 `unauthorized`

#### Scenario: Token not stored in plain text
- **WHEN** a match is created and its stored files are read
- **THEN** no file contains the returned token

### Requirement: Script submission per round
While a match is `awaiting_scripts`, each seat SHALL submit its script for the next round with `POST /matches/{id}/scripts` and the body `{"script"}`. Submitting the same script as before is how a seat keeps it. A seat MAY resubmit before the round runs, and the latest submission replaces the earlier one. Seat a MAY also resubmit its round-1 script while waiting for an opponent. The response SHALL be 200 with `{"match": <state>}`. Submitting to a finished match SHALL be rejected with 409 `conflict`. An invalid script SHALL be rejected with 422 `invalid_script` and leave the earlier submission in place.

#### Scenario: Revise between rounds
- **WHEN** after round 1 seat a submits a revised script and then seat b resubmits its old script
- **THEN** round 2 runs with seat a's revised script and seat b's unchanged script

#### Scenario: Replace before the round runs
- **WHEN** seat a submits script X and then script Y before seat b submits
- **THEN** the next round uses script Y for seat a

#### Scenario: Submit after the match finished
- **WHEN** a seat submits a script to a finished match
- **THEN** the response is 409 `conflict`

### Requirement: Round execution and sides
A round SHALL run as soon as both seats have a submission for it, inside the request that completed the pair, and exactly once even when both seats submit at the same moment. Round n SHALL put seat a as p1 when n is odd and seat b as p1 when n is even. It SHALL use the default config and the seat names as the player names. Each played round SHALL record its number, which seat was p1, the winning seat (or null), the reason, the end frame, and each seat's final HP. Its full event log SHALL be stored.

#### Scenario: Sides alternate
- **WHEN** rounds 1, 2 and 3 are played
- **THEN** seat a is p1 in rounds 1 and 3 and seat b is p1 in round 2

#### Scenario: Round result maps to seats
- **WHEN** seat b is p1 in round 2 and the log's winner is `p1`
- **THEN** the round's winning seat is `b`

#### Scenario: Concurrent final submissions
- **WHEN** both seats' submissions for a round arrive concurrently
- **THEN** exactly one round with that number is recorded

### Requirement: Match result
A seat that wins 2 rounds SHALL win the match, and the match SHALL become `finished` immediately. Drawn rounds SHALL count for neither seat. If 5 rounds have been played without a seat reaching 2 wins, the match SHALL finish: the seat with more round wins takes it, or it is a draw (winner null) when the wins are equal. Otherwise, after a round the status SHALL be `awaiting_scripts`, and neither seat has a submission for the next round.

#### Scenario: Two straight wins
- **WHEN** seat a wins rounds 1 and 2
- **THEN** the match is `finished` with winner `a` after 2 rounds

#### Scenario: Draws extend the match
- **WHEN** round 1 is a draw, seat b wins round 2 and seat a wins round 3
- **THEN** the match is still `awaiting_scripts`, with 1 win each, and round 4 is next

#### Scenario: Round cap
- **WHEN** 5 rounds are played with results draw, a, draw, b, draw
- **THEN** the match is `finished` with winner null

### Requirement: Match state and visibility
`GET /matches/{id}` SHALL return 200 with the state `{"match_id", "status", "config_version", "next_round", "seats": {"a": {"name", "wins", "submitted"}, "b": <same> | null}, "rounds": [...], "winner"}`. `submitted` reports whether that seat has a pending submission for the next round. The state SHALL NOT include any pending, unplayed script. Scripts become visible only through the logs of played rounds. Reading state needs no token. An unknown id SHALL return 404 `not_found`.

#### Scenario: Pending script hidden
- **WHEN** seat a has submitted a revised script for round 2 and seat b has not
- **THEN** `GET /matches/{id}` shows seat a `submitted` true and does not contain seat a's revised script anywhere

#### Scenario: Unknown match
- **WHEN** a client requests `GET /matches/0000000000000000` for a match that does not exist
- **THEN** the response is 404 `not_found`

### Requirement: Round logs
`GET /matches/{id}/rounds/{n}` SHALL return 200 with the stored event log of played round n. Its bytes SHALL be identical to the engine's log for that round's scripts, names and sides. A round that has not been played SHALL return 404 `not_found`.

#### Scenario: Fetch a replay
- **WHEN** round 1 has been played and a client requests `/matches/{id}/rounds/1`
- **THEN** the body equals the engine's log for round 1's p1 and p2 scripts and names

#### Scenario: Unplayed round
- **WHEN** a client requests round 3 of a match that has played 1 round
- **THEN** the response is 404 `not_found`

### Requirement: Persistence
Matches, pending submissions and round logs SHALL be stored under the server's data directory and SHALL survive a server restart. Each write SHALL be atomic, so a crash never leaves a partially written file in place. The stored format SHALL carry a `store_version`.

#### Scenario: Restart mid-match
- **WHEN** round 1 is played, seat a submits for round 2, and the server is restarted with the same data directory
- **THEN** `GET /matches/{id}` returns the same state, and once seat b submits, round 2 runs with seat a's earlier submission
