---
description: Join a Prompt Fighter match from its game link (only takes a seat; writes and submits no script)
argument-hint: <game link> [fighter name]
allowed-tools: mcp__plugin_pf_prompt-fighter__join_match, mcp__plugin_pf_prompt-fighter__get_match, Read, Write
---

Join the Prompt Fighter match the player gave you, and **only** join. Arguments: `$ARGUMENTS`

1. The first argument is the game link (it looks like `http://HOST:PORT/play/matches/<16-character id>`). If it is missing, ask the player for it and stop. The rest, if any, is the fighter name; if there is no name, ask the player for one and stop.
2. Call `join_match` with `match_id` set to the game link and `name` set to the fighter name. Do **not** pass a script.
3. Save the result to `.prompt-fighter/match.json` in the current directory, exactly in this shape (keep the token secret; it is the player's seat key):
   ```json
   {"link": "<game link>", "match_id": "<match_id>", "seat": "<a or b>", "token": "<token>", "name": "<fighter name>"}
   ```
4. Tell the player, briefly, which seat they have and the match state (`get_match` with the link), and how each round works:
   - `/pf:script <how you want to fight>` writes the script for the next round as `roundN.pf` (run it again to revise);
   - `/pf:submit` sends that round's script. The round runs once both players have submitted.

Do not write any `.pf` script and do not submit anything in this command. Wait for the player.
