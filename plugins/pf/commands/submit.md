---
description: Submit fighter.pf for the next round of your Prompt Fighter match (never edits it)
allowed-tools: mcp__plugin_pf_prompt-fighter__validate_script, mcp__plugin_pf_prompt-fighter__submit_script, mcp__plugin_pf_prompt-fighter__get_match, mcp__plugin_pf_prompt-fighter__get_round_replay, Read
---

Submit the player's current script for the next round of their Prompt Fighter match, exactly as it is.

1. Read `.prompt-fighter/match.json`. If it does not exist, tell the player to join first with `/pf:join <game link> <name>` and stop.
2. Read `fighter.pf` from the current directory. If it does not exist, tell the player and stop.
3. Call `validate_script` with the file's exact contents. If there are errors, list them (line:column and message, with the suggested fix) and stop. **Do not edit `fighter.pf` and do not submit**; the player decides what to change.
4. Call `submit_script` with `match_id` set to the saved `link`, the saved `token`, and the file's exact contents. Never modify the script before submitting.
5. Report the result briefly:
   - if the round ran (the response has a new round): the result, and offer to study it with `get_round_replay` when the player asks;
   - otherwise: that the script is in and which player the round is waiting for (`get_match` with the link).

Do not change `fighter.pf` in this command, even if you think it could be better. Suggest changes only if the player asks.
