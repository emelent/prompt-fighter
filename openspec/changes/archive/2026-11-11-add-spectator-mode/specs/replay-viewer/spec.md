# Spec Delta

## ADDED Requirements

### Requirement: Simple round player
Spectator replays and replays opened from the play page (`?from=play`) SHALL use the simple round player. It shows the stage with only these controls visible:
- play/pause
- rewind, which returns to frame 0 (and keeps playing if it was playing)
- at the last frame, the end-of-replay button (Next round or Done)
- a **settings** button

Playback SHALL default to 0.5× with sound on. The player scripts, event feed, step buttons, scrub bar, frame counter control, speed control and the checks and ranges toggles SHALL be hidden. The settings button SHALL open a menu with a toggle for each hidden part: scripts, event feed, timeline (scrub bar, step buttons and frame counter), speed, checks and ranges, and sound. Turning a toggle on shows that part at once. The toggles and the chosen speed SHALL be remembered in the browser and used on later simple replays. If browser storage is unavailable, the defaults are used. Keyboard shortcuts keep working. The `/view` analysis pages SHALL keep their full controls and are not affected by these settings.

#### Scenario: Default simple view
- **WHEN** a spectator opens a round replay for the first time in this browser
- **THEN** only play/pause, rewind and settings are visible, the round plays at 0.5× with sound on, and no script is shown

#### Scenario: Show the scripts
- **WHEN** the viewer opens settings and turns on "scripts"
- **THEN** both script panels appear with the executing line highlighted, and they are shown again on the next simple replay

#### Scenario: Speed is remembered
- **WHEN** the viewer turns on "speed", picks 1×, and later opens another simple replay
- **THEN** that replay plays at 1×

#### Scenario: Analysis viewer unchanged
- **WHEN** a round is opened at `/view/matches/{id}/rounds/{n}` without `from=play`
- **THEN** the full controls, scripts and event feed are shown and playback starts at 1×

## MODIFIED Requirements

### Requirement: Auto-play and return links
A round player URL with the query parameter `autoplay=1` SHALL start playing from frame 0 as soon as the log has loaded, unless a `#f=N` frame link is present, which takes precedence and opens paused. With `from=play`, the round player SHALL use the simple round player, its back link SHALL go to `/play/matches/{id}` and read "back to the match", and at the last frame it SHALL show a **Done** button that returns to `/play/matches/{id}`. Without it, the back link goes to the viewer's match page and the full controls are shown, as before.

#### Scenario: Auto-play from the play page
- **WHEN** the browser opens `/view/matches/{id}/rounds/2?from=play&autoplay=1`
- **THEN** playback starts immediately, and the back link points to `/play/matches/{id}`

#### Scenario: Frame link wins over autoplay
- **WHEN** the browser opens `/view/matches/{id}/rounds/2?autoplay=1#f=300`
- **THEN** the player opens paused at frame 300

#### Scenario: Done after watching your round
- **WHEN** a player's round replay opened with `?from=play&autoplay=1` reaches its last frame
- **THEN** a Done button appears, and pressing it returns to `/play/matches/{id}`
