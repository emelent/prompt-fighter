# Spec Delta

## MODIFIED Requirements

### Requirement: Menu music
The lobby and the spectator waiting screen SHALL play the menu music track on a loop, starting as soon as the browser allows audio (at the first click or key press on the page if it has to wait) and stopping when the visitor leaves the page. On the waiting screen it keeps playing through the countdown and stops when a round opens, where the round's fight music takes over. A labelled **sound** toggle on both pages SHALL turn the music off and on. It SHALL share the sound preference with the simple round player, so turning sound off in one place turns it off in both. The music SHALL be served from the same origin like the other viewer assets.

#### Scenario: Music in the lobby
- **WHEN** a visitor opens `/play` with sound on and clicks anywhere on the page
- **THEN** the menu music starts and loops until they leave the lobby

#### Scenario: Sound off
- **WHEN** the visitor turns the lobby's sound toggle off
- **THEN** the music stops, and it stays off on the next visit and in simple replays

#### Scenario: Music while waiting to watch
- **WHEN** a host creates a match and waits on the spectator waiting screen with sound on
- **THEN** the menu music plays through the countdown, and when the round opens the menu music stops and the fight music starts
