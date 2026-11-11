---
description: Join a Prompt Fighter match from its game link (takes a seat, does not submit)
argument-hint: <game link> [fighter name]
allowed-tools: mcp__plugin_pf_prompt-fighter__join_match, mcp__plugin_pf_prompt-fighter__get_player_guide, mcp__plugin_pf_prompt-fighter__validate_script, mcp__plugin_pf_prompt-fighter__list_sample_bots, mcp__plugin_pf_prompt-fighter__get_match, Read, Write
---

Join the Prompt Fighter match the player gave you, and only join. Arguments: `$ARGUMENTS`

1. The first argument is the game link (it looks like `http://HOST:PORT/play/matches/<16-character id>`). If it is missing, ask the player for it and stop. The rest, if any, is the fighter name; if there is no name, ask the player for one and stop.
2. If you have not read the player guide in this session, call `get_player_guide` and read it, so you can help with scripts later.
3. Call `join_match` with `match_id` set to the game link and `name` set to the fighter name. Do **not** pass a script: joining only takes a seat.
4. Save the result to `.prompt-fighter/match.json` in the current directory, exactly in this shape (keep the token secret; it is the player's seat key):
   ```json
   {"link": "<game link>", "match_id": "<match_id>", "seat": "<a or b>", "token": "<token>", "name": "<fighter name>"}
   ```
5. If `fighter.pf` does not exist in the current directory, create it as a starting point: call `list_sample_bots` and write the `rushdown` bot's `source` (or another bot the player names) to the file. Call `validate_script` on `fighter.pf` and fix it only if it is invalid.
6. Tell the player, briefly:
   - which seat they have and the match state (`get_match` with the link);
   - that **nothing has been submitted**;
   - how each round works: they ask you to study the last replay and change `fighter.pf`, then run `/pf:submit` to send it. Each round runs once both players have submitted.

Do not submit a script and do not start changing `fighter.pf` beyond creating the starter. Wait for the player.
