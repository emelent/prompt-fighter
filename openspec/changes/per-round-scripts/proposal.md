# Proposal

## Why

With `per-round-llm-play`, `/pf:join` created a starter `fighter.pf`, and players tweaked that one file between rounds. The user wants no script at all on join, and an explicit, prompted script step for every round, round 1 included: the player says how they want to fight, the LLM writes that round's script, and only `/pf:submit` sends it.

Three smaller issues are recorded here too:
- **Install commands:** the host's setup view put both plugin install commands in one copy box, and pasting it into Claude Code merged them into one invalid command.
- **Round-1 wording:** the setup view still said round 1 starts when both players join.
- **Waiting-screen music:** the spectator's waiting screen was silent between the lobby and the round.

This change records work that is already implemented. It was built directly at the user's request, and the plugin and waiting-screen main specs were updated by hand at the time. Its tasks are verification only.

## What Changes

- **`/pf:join <game link> [name]`** only joins. It saves `.prompt-fighter/match.json`, writes no `.pf` file, submits nothing, and is not allowed to call `submit_script`.
- **New `/pf:script <how you want to fight>`:**
  - It takes N from the match's `next_round` and writes `roundN.pf` (`round1.pf`, `round2.pf`, …) from the player's instructions.
  - From round 2 on, it studies round N-1's replay first.
  - It validates the script and fixes it until it is valid. Running it again revises the file.
  - It never submits and is not allowed to call `submit_script`.
- **`/pf:submit`** submits `roundN.pf` for the current round, exactly as it is. If the file is missing, it points the player to `/pf:script`. If the script is invalid, it reports the errors and submits nothing.
- **Plugin version and manifests:** the plugin is now `0.2.0`, and its manifest and marketplace descriptions list the three commands.
- **Host's setup view:** every command is its own numbered copy field:
  1. Add marketplace
  2. Install plugin
  3. Join
  4. Write the round
  5. Submit the round

  The round-1 text now says round 1 starts once both players have joined *and submitted*.
- **Menu music on the waiting screen:**
  - The menu music, with its sound toggle, now plays on the spectator waiting screen too, through the countdown.
  - Opening a round loads a new page, where the fight music takes over.
  - The menu-music code moved out of the lobby code into `web/src/menumusic.ts`.
- **Docs:** README and player guide describe join → script → submit.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `claude-plugin`:
  - "/pf:join" changes to join only, with no starter script.
  - "/pf:script" is new.
  - "/pf:submit" now sends `roundN.pf`.
  - "Plugin and marketplace" now lists three commands.

  These are recorded as MODIFIED because the main spec was already updated by hand.
- `spectator-mode`: "Waiting screen" now shows the per-round commands, each in its own copy field.
- `play-ui`: "Menu music" now also plays on the spectator waiting screen until a round opens.

## Impact

- **Plugin:** `plugins/pf/commands/{join,script,submit}.md`, `plugin.json` (0.2.0), `.claude-plugin/marketplace.json`. `plugins/plugin_test.go` covers the three commands, and checks that only `submit` may submit.
- **Web:**
  - `web/src/spectate.ts`: `marketplaceCommand`, `installCommand`, `joinCommand`, `scriptCommand` and `submitCommand`.
  - `web/src/watch.ts`: copy fields, wording and menu music.
  - `web/src/menumusic.ts` (new), and `web/src/play.ts`.
  - Tests in `web/test/spectate.test.ts`.
- **Docs:** `README.md`, `docs/PLAYER_GUIDE.md`, `docs/API.md` (the waiting-screen row).
- **Not affected:** the server and MCP server.
