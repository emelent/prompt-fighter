---
description: Submit this round's script (roundN.pf, written by /pf:script) for your Prompt Fighter match, unchanged
allowed-tools: mcp__plugin_pf_prompt-fighter__get_match, mcp__plugin_pf_prompt-fighter__validate_script, mcp__plugin_pf_prompt-fighter__submit_script, mcp__plugin_pf_prompt-fighter__get_round_replay, Read
---

Submit the player's script for the next round of their Prompt Fighter match, exactly as it is on disk.

1. Read `.prompt-fighter/match.json`. If it does not exist, tell the player to join first with `/pf:join <game link> <name>` and stop.
2. Call `get_match` with the saved `link`. If the match is finished, say so and stop. Otherwise the round is `next_round`; call it N.
3. Read `roundN.pf` from the current directory (for example `round1.pf`). If it does not exist, tell the player to write it first with `/pf:script <how you want to fight>` and stop.
4. Call `validate_script` with the file's exact contents. If there are errors, list them (line:column and message, with the suggested fix) and stop. **Do not edit the file and do not submit**; the player can fix it with `/pf:script`.
5. Call `submit_script` with `match_id` set to the saved `link`, the saved `token`, and the file's exact contents. Never modify the script before submitting.
6. Report briefly:
   - if the round ran (the response has round N): the result, and that `/pf:script` writes the next round's script (it will study the replay; `get_round_replay` shows it in detail if the player asks);
   - otherwise: that round N's script is in and which player the round is waiting for.

Do not change any `.pf` file in this command.
