# Design

## Context

- The plugin commands are Markdown prompts with `allowed-tools` frontmatter. The MCP tools they call (`join_match`, `get_match`, `validate_script`, `submit_script`, `get_round_replay`, `simulate_round`, `list_sample_bots`, `get_player_guide`) are unchanged.
- `GET /matches/{id}`, and so `get_match`, returns `next_round`: the round the next submissions are for. After round N is played it becomes N+1.
- The waiting screen and the lobby are separate page loads. Opening a round navigates to `/watch/matches/{id}/rounds/{n}`, whose stage plays a fight track.

## Goals / Non-Goals

**Goals:**
- Joining never writes or submits a script.
- One script file per round, written only by `/pf:script` and sent only by `/pf:submit`.
- Copy fields that can't be pasted as two commands at once.

**Non-Goals:**
- Server or MCP changes. Submission still goes through `submit_script` with the saved token.
- Mechanically enforcing "submit unchanged": the commands are prompts. That stays a follow-up (a file-submitting MCP tool).

## Decisions

### 1. The round number comes from the server
`/pf:script` and `/pf:submit` both call `get_match` and use `next_round`, so they always agree on which `roundN.pf` is current, even across sessions. Before round N is played, both refer to `roundN.pf`. Afterwards, `/pf:script` writes `round(N+1).pf` and studies round N's replay.

- **Alternative:** a counter in `match.json`. Rejected because it can drift from the match, for example when the opponent submits from another place or a round runs while the player is away.

### 2. Files in the working directory
The script files are `roundN.pf` in the working directory, and the seat goes in `.prompt-fighter/match.json`. The player can see and edit them, and the history of rounds stays on disk.

### 3. Enforcement through allowed tools
`join.md` and `script.md` don't list `submit_script` in `allowed-tools`, so they can't submit. A plugin test fails if either ever does.

### 4. One command per copy field
The setup view gives each command its own single-line field. Pasting two lines into Claude Code sends them as one message, which caused `/plugin marketplace add` to treat the second command as part of the repo name. A test checks that every command value is a single line.

### 5. Menu music module
`menumusic.ts` exports `menuMusic()`, which starts the looping `<audio>` and returns the sound toggle, and `stopMenuMusic()`. It reuses the `pf.simple.sound` setting. The waiting screen starts it. Navigation into a round ends it, and the round page's own fight track takes over, so no explicit hand-off is needed.

## Risks / Trade-offs

- **[A model may still deviate from the command prompts]** → `submit_script` is blocked outside `/pf:submit`. Within `/pf:submit`, the instructions repeat "unchanged".
- **[Audio on a directly opened watch link]** → Browsers need a click first. After Create match on the same site, it plays without one (verified in headless Chrome).
