# spectator-mode Specification

## Purpose

Lets anyone watch a match on a `pf serve` server without taking a seat: a waiting screen between rounds, and each round played back in the simple round player, moving from round to round with a short countdown.

## Requirements

### Requirement: Watch pages
`pf serve` SHALL serve these pages from the same embedded web app as the viewer, with the same security headers:
- `GET /watch/matches/{id}` is the waiting screen for a match.
- `GET /watch/matches/{id}/rounds/{n}` is the spectator replay of a played round.

Each SHALL return 200 HTML whether or not the match exists; the page itself reports a missing match or round. Watch pages SHALL use only `GET /matches/{id}` and `GET /matches/{id}/rounds/{n}`. They need no seat or token, never send scripts or tokens, and offer no way to change the match.

#### Scenario: Open a watch link
- **WHEN** a browser with no stored seat opens `/watch/matches/0123456789abcdef`
- **THEN** the response is 200 HTML with the viewer's Content-Security-Policy, and the page loads `/matches/0123456789abcdef`

#### Scenario: Unknown match
- **WHEN** a watch link names a match the server reports as not found
- **THEN** the page says the match does not exist and links to the lobby

#### Scenario: Spectators cannot act
- **WHEN** a spectator uses every control on a watch page
- **THEN** no `POST` request is sent to the server

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

### Requirement: Round transitions
When the waiting screen sees a round that has been played but not yet watched on this page, it SHALL show "Round N starting in 5", count down once per second to 1, and then open `/watch/matches/{id}/rounds/{N}`. If several new rounds appear at once, it SHALL start with the lowest one not yet watched. The countdown SHALL also offer a **Watch now** button that opens the round at once. A waiting screen opened directly (from a watch link or the lobby) SHALL treat the rounds already played as watched, listing them as links, so it waits for the next round. When it is reached from a replay's Done button, the rounds up to the one just watched count as watched.

#### Scenario: Joining mid-match
- **WHEN** a spectator opens the watch link of a match in which round 1 has been played and round 2 is awaiting scripts
- **THEN** the waiting screen shows round 2 with a link to the round 1 replay, and no countdown

#### Scenario: Both scripts are in
- **WHEN** the waiting screen is open and the second script for round 2 arrives, so round 2 is played
- **THEN** within about 2 seconds the screen shows "Round 2 starting in 5", counts down to 1, and opens the round 2 replay

### Requirement: End of a spectator replay
When a spectator replay reaches its last frame:
- If the next round has already been played, a **Next round** button SHALL appear with a countdown "Round N+1 starting in 5" that opens the next round's spectator replay when it reaches 0. Pressing the button opens it at once.
- Otherwise a **Done** button SHALL appear that returns to the waiting screen.

While the last frame is shown, the page SHALL poll `GET /matches/{id}` every 2 seconds, and switch from Done to Next round (with its countdown) as soon as the next round is played. Rewinding or playing again SHALL cancel a running countdown.

#### Scenario: Next round already played
- **WHEN** a spectator finishes watching round 1 and round 2 has already been played
- **THEN** a Next round button and a 5-second countdown appear, and round 2's replay opens when it ends

#### Scenario: Next round not played yet
- **WHEN** a spectator finishes watching round 2 and round 3 has not been played
- **THEN** a Done button appears, and pressing it opens the waiting screen for round 3

#### Scenario: Round arrives while on the last frame
- **WHEN** a spectator is on the last frame of round 1 with the Done button showing, and round 2 is then played
- **THEN** within about 2 seconds the Done button is replaced by Next round with its countdown

### Requirement: Finished match
When the match is finished, the waiting screen SHALL show the final result (winner by name, or a draw, and the wins per seat) and links to every round's replay, and SHALL stop polling. After the last round's replay of a finished match, the end-of-replay button SHALL be **Done** and lead to this result.

#### Scenario: Match over
- **WHEN** a spectator presses Done after watching the deciding round
- **THEN** the waiting screen shows the winner and the final score, with links to all rounds, and does not poll
