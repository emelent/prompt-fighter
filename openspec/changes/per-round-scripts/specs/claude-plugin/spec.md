# Spec Delta

## MODIFIED Requirements

### Requirement: Plugin and marketplace
The repository SHALL contain a Claude Code plugin named `pf` at `plugins/pf`, with a plugin manifest, an MCP configuration that runs `pf mcp` as the `prompt-fighter` server, and the commands `join`, `script` and `submit`. The repository root SHALL contain a marketplace manifest named `prompt-fighter` that lists the plugin, so it can be installed with `/plugin marketplace add <owner>/<repo>` and `/plugin install pf@prompt-fighter`. The plugin requires the `pf` binary on the player's PATH. An automated test SHALL check that the manifests parse, that the marketplace entry points at the plugin, and that every MCP tool a command names exists in the MCP server's tool catalogue.

#### Scenario: Manifests are consistent
- **WHEN** the plugin test runs
- **THEN** it confirms that both manifests parse, that the marketplace lists `pf` with source `./plugins/pf`, and that the commands reference only existing tools

### Requirement: /pf:join
`/pf:join <game link> [name]` SHALL only join the match at the game link: it calls `join_match` with the link and the given name (asking for one if missing) and no script, saves `{link, match_id, seat, token, name}` to `.prompt-fighter/match.json` in the working directory, and tells the player their seat and that each round is written with `/pf:script` and sent with `/pf:submit`. It SHALL NOT write any `.pf` script or submit anything, and its allowed tools SHALL NOT include `submit_script`.

#### Scenario: Join a hosted match
- **WHEN** a player runs `/pf:join http://192.168.1.20:8080/play/matches/0123456789abcdef claude-bot`
- **THEN** the LLM joins as the first free seat, writes `.prompt-fighter/match.json` with the seat token, writes no script, and reports that round 1 starts with `/pf:script`

### Requirement: /pf:script
`/pf:script <how you want to fight>` SHALL write the player's script for the next round: it reads `.prompt-fighter/match.json`, takes the round N from `get_match`'s `next_round`, and writes `roundN.pf` in the working directory following the player's instructions (revising the file when it already exists, and studying round N-1 with `get_round_replay` from round 2 on). It SHALL validate the file and fix it until it is valid, then report the file and the plan. It SHALL NOT submit, and its allowed tools SHALL NOT include `submit_script`. This is the process for every round, round 1 included.

#### Scenario: First round
- **WHEN** a player who has just joined runs `/pf:script rush in with kicks`
- **THEN** `round1.pf` is written and validated, and nothing is submitted

#### Scenario: Later round
- **WHEN** round 1 has been played and the player runs `/pf:script stop getting swept`
- **THEN** the LLM studies round 1's replay and writes `round2.pf`

### Requirement: /pf:submit
`/pf:submit` SHALL submit the next round's script, `roundN.pf` (N from `get_match`'s `next_round`), exactly as it is on disk: it validates it with `validate_script` and calls `submit_script` with the saved link and token. If no match has been joined it says to run `/pf:join`; if `roundN.pf` does not exist it says to run `/pf:script`; if the script is invalid it reports the errors and stops. It SHALL NOT change any `.pf` file. After submitting it reports the match state: whether the round ran, and if so its result.

#### Scenario: Submit the round's script
- **WHEN** `round1.pf` has been written with `/pf:script` and the player runs `/pf:submit`
- **THEN** the exact contents of `round1.pf` are validated and submitted, and the reply says whether the opponent has submitted yet

#### Scenario: No script for the round
- **WHEN** the player runs `/pf:submit` before `/pf:script` for round 2
- **THEN** nothing is submitted and the player is told to run `/pf:script` first

#### Scenario: Invalid script
- **WHEN** `round1.pf` contains `kik`
- **THEN** `/pf:submit` reports the validation error with its line and column, and nothing is submitted
