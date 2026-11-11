---
description: Write (or revise) your fight script for the next round as roundN.pf, from how you want to fight (does not submit)
argument-hint: <how you want to fight>
allowed-tools: mcp__plugin_pf_prompt-fighter__get_match, mcp__plugin_pf_prompt-fighter__get_round_replay, mcp__plugin_pf_prompt-fighter__get_player_guide, mcp__plugin_pf_prompt-fighter__validate_script, mcp__plugin_pf_prompt-fighter__simulate_round, mcp__plugin_pf_prompt-fighter__list_sample_bots, Read, Write
---

Write the player's fight script for the next round of their Prompt Fighter match. How they want to fight: `$ARGUMENTS`

1. Read `.prompt-fighter/match.json`. If it does not exist, tell the player to join first with `/pf:join <game link> <name>` and stop.
2. Call `get_match` with the saved `link`. If the match is finished, say so and stop. Otherwise the round to write is `next_round`; call it N. The file is `roundN.pf` in the current directory (for example `round1.pf`).
3. If you have not read the player guide in this session, call `get_player_guide` and read it.
4. Work out the script:
   - Follow the player's instructions above. If they gave none, ask how they want to fight and stop.
   - If `roundN.pf` already exists, treat this as a revision of it.
   - For round 2 and later, study the previous round with `get_round_replay` (round N-1) and use what happened to inform the script.
   - You may test ideas with `simulate_round` against the sample bots (`list_sample_bots`).
5. Write the script to `roundN.pf`, then call `validate_script` on its exact contents and fix it until it is valid.
6. Tell the player, briefly: the file name, the plan behind the script, and that `/pf:submit` sends it (or `/pf:script` again to change it).

Do **not** submit anything in this command; only `/pf:submit` submits.
