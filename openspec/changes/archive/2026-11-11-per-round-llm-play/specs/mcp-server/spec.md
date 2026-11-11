# Spec Delta

## MODIFIED Requirements

### Requirement: Match tools
The match tools SHALL call the HTTP API of the `pf serve` server at the configured URL, mapping to it as follows:
- `create_match {name, script}` → `POST /matches`
- `join_match {match_id, name, script}` → `POST /matches/{id}/join`. `match_id` MAY be a bare id or a game link (any URL containing `/play/matches/{id}` or `/watch/matches/{id}`). For a game link, the tool SHALL call the `pf serve` server at the link's origin. For a bare id, it calls the configured server. If there is no configured server, a bare id fails with an explanation asking for the game link. `join_match`'s `script` is optional: without it, the seat starts with nothing submitted.
- `submit_script {match_id, token, script}` → `POST /matches/{id}/scripts`
- `get_match {match_id}` → `GET /matches/{id}`
- `get_round_replay {match_id, round, include_checks?}` → the replay summary of `GET /matches/{id}/rounds/{n}`, plus which seat played p1

`create_match` and `join_match` SHALL return the `match_id`, `seat` and `token`, along with the match state. `join_match`'s description SHALL say that the joiner gets the first free seat (a in a hosted match nobody has joined yet, otherwise b), and its result SHALL name the seat given. Their descriptions SHALL tell the model that the token is shown only once, that it must be passed to `submit_script`, and that matches do not expire. Every match tool's text content SHALL summarise the state: status, wins per seat, whether each seat has submitted, the round results, and what to do next.

#### Scenario: Two models play a match
- **WHEN** one MCP client calls `create_match` and another calls `join_match` with that `match_id`
- **THEN** the join result shows round 1 played, and `get_match` from either client shows the same state

#### Scenario: Replay of a match round
- **WHEN** `get_round_replay` is called for round 2 of a match where seat b played p1
- **THEN** the summary reports that seat b was p1, and its events equal the `pf sim` lines for that round's stored log

#### Scenario: Wrong token
- **WHEN** `submit_script` is called with a token that does not belong to the match
- **THEN** the result has `isError` true and includes the API code `unauthorized`

#### Scenario: Match tools disabled
- **WHEN** `pf mcp` was started with `--server ""` and `create_match` is called
- **THEN** the result has `isError` true and explains that match tools need a `pf serve` URL

#### Scenario: Join by game link
- **WHEN** `join_match` is called with `match_id` `http://127.0.0.1:8080/play/matches/0123456789abcdef` and `pf mcp` uses `http://127.0.0.1:8080`
- **THEN** it joins match `0123456789abcdef` and reports the seat it was given

#### Scenario: Link for another server
- **WHEN** `join_match` is given a link to a match on `http://192.168.1.20:8080` while `pf mcp` was started with the default server
- **THEN** it joins the match on `http://192.168.1.20:8080`, and later calls with the same link go there too
