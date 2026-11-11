# Spec Delta

## MODIFIED Requirements

### Requirement: Create and join
`POST /matches` with `{"name", "script"}` SHALL create a match with the caller in seat `a`. It SHALL return 201 with `{"match_id", "seat": "a", "token", "match": <state>}`. The match starts in status `waiting_for_opponent`, and seat a's script counts as its submission for round 1.

`POST /matches` with an empty JSON object (no `name` and no `script`) SHALL create a **hosted** match with both seats empty, for a host who only watches. It SHALL return 201 with `{"match_id", "match": <state>}` and no seat or token. A hosted match starts in status `waiting_for_players`. Sending only one of `name` and `script` SHALL fail with 400 `bad_request`.

`POST /matches/{id}/join` with `{"name"}` and an optional `"script"` SHALL put the caller in the first free seat, `a` if it is empty and otherwise `b`, and return 201 with `{"match_id", "seat", "token", "match": <state>}`. A script given on join counts as the joiner's round-1 submission. Without one, the seat starts with nothing submitted, and the player submits round 1 with `POST /matches/{id}/scripts`. When the join fills seat a of a hosted match, the status becomes `waiting_for_opponent`. When it fills the last seat, the status becomes `awaiting_scripts`. Every round, round 1 included, runs as soon as both seats have a submission, which may be inside the join request when both scripts are already in. Joining a match whose seats are both taken SHALL fail with 409 `conflict`. Names are 1–40 bytes. Scripts MUST validate, or the request fails with 422 `invalid_script`.

Match ids SHALL be 16 lowercase hex characters, and seat tokens 32 lowercase hex characters, both from a cryptographically secure random source. Tokens are returned only once, in the create or join response.

#### Scenario: Create a match
- **WHEN** a client posts a valid name and script to `/matches`
- **THEN** the response is 201 with seat `a`, a 32-hex-character token, and a match whose status is `waiting_for_opponent` with seat b empty

#### Scenario: Join runs round 1
- **WHEN** a second client posts a valid name and script to `/matches/{id}/join`
- **THEN** the response is 201 with seat `b`, and the returned match has one played round with seat a as p1

#### Scenario: Seat already taken
- **WHEN** a client joins a match whose seats a and b are both filled
- **THEN** the response is 409 `conflict`

#### Scenario: Invalid script on create
- **WHEN** a client creates a match with a script containing `kik`
- **THEN** the response is 422 `invalid_script` and no match is created

#### Scenario: Host a match
- **WHEN** a client posts `{}` to `/matches`
- **THEN** the response is 201 with a `match_id`, no `seat` and no `token`, and a match whose status is `waiting_for_players` with both seats empty

#### Scenario: Players join a hosted match
- **WHEN** one client joins a hosted match, and then another
- **THEN** the first join returns seat `a` and leaves the match `waiting_for_opponent`, and the second returns seat `b` with round 1 already played, seat a as p1

#### Scenario: Half a create request
- **WHEN** a client posts `{"name": "alice"}` to `/matches` without a script
- **THEN** the response is 400 `bad_request` and no match is created

#### Scenario: Join without a script
- **WHEN** two players join a hosted match with only their names
- **THEN** both joins succeed, the match is `awaiting_scripts` with no rounds played, and round 1 runs when the second of them submits a script
