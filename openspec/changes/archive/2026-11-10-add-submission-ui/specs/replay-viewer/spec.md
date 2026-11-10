# Spec Delta

## ADDED Requirements

### Requirement: Auto-play and return links
A round player URL with the query parameter `autoplay=1` SHALL start playing from frame 0 as soon as the log has loaded, unless a `#f=N` frame link is present, which takes precedence and opens paused. With `from=play`, the round player's back link SHALL go to `/play/matches/{id}` and read "back to the match". Without it, the back link goes to the viewer's match page, as before.

#### Scenario: Auto-play from the play page
- **WHEN** the browser opens `/view/matches/{id}/rounds/2?from=play&autoplay=1`
- **THEN** playback starts immediately, and the back link points to `/play/matches/{id}`

#### Scenario: Frame link wins over autoplay
- **WHEN** the browser opens `/view/matches/{id}/rounds/2?autoplay=1#f=300`
- **THEN** the player opens paused at frame 300
