# Spec Delta

## MODIFIED Requirements

### Requirement: Waiting screen
The waiting screen SHALL show:
- the round being waited for (`next_round`)
- the score: each seat's name and wins
- for each seat, whether it has submitted for that round; seats that have not are listed as "waiting for <name>", and a missing seat b as "waiting for an opponent to join"
- links to the replays of rounds already played

While any seat is free, it SHALL instead show the setup view:
- "Waiting for players (n/2 joined)", with the name of each player who has joined
- the game link (`/play/matches/{id}` on this server) with a copy button
- copy-ready instructions for joining with an LLM through the `pf` Claude Code plugin: the two install commands (`/plugin marketplace add …` and `/plugin install pf@prompt-fighter`), the join command `/pf:join <game link>`, and the per-round commands `/pf:script <how you want to fight>` (writes `roundN.pf`) and `/pf:submit`, each command in its own copy field

While the match is not finished, it SHALL poll `GET /matches/{id}` every 2 seconds and update in place.

#### Scenario: Waiting on one player
- **WHEN** the match is waiting for round 2, seat a (alice, 1 win) has submitted and seat b (bob, 0 wins) has not
- **THEN** the screen shows round 2, the score alice 1 – bob 0, alice as submitted, and "waiting for bob"

#### Scenario: Updates while open
- **WHEN** the waiting screen is open and seat b submits its script
- **THEN** within about 2 seconds the screen shows seat b as submitted

#### Scenario: Hosted match waiting for players
- **WHEN** a host has just created a hosted match and one player, alice, has joined
- **THEN** the waiting screen shows "Waiting for players (1/2 joined)" with alice, the game link with a copy button, and the plugin instructions including `/pf:join <the game link>`

#### Scenario: Both players joined
- **WHEN** the second player joins with only a name
- **THEN** the setup view is replaced by the round 1 waiting view, showing who has submitted, and the countdown starts once round 1 is played
