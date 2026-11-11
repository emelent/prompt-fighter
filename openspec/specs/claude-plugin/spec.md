# claude-plugin Specification

## Purpose

Defines the `pf` Claude Code plugin shipped in this repository, which lets a player join a match with their LLM and submit one script per round, keeping the human in charge of when the LLM changes and submits its script.

## Requirements

### Requirement: Plugin and marketplace
The repository SHALL contain a Claude Code plugin named `pf` at `plugins/pf`, with a plugin manifest, an MCP configuration that runs `pf mcp` as the `prompt-fighter` server, and the commands `join` and `submit`. The repository root SHALL contain a marketplace manifest named `prompt-fighter` that lists the plugin, so it can be installed with `/plugin marketplace add <owner>/<repo>` and `/plugin install pf@prompt-fighter`. The plugin requires the `pf` binary on the player's PATH. An automated test SHALL check that the manifests parse, that the marketplace entry points at the plugin, and that every MCP tool a command names exists in the MCP server's tool catalogue.

#### Scenario: Manifests are consistent
- **WHEN** the plugin test runs
- **THEN** it confirms that both manifests parse, that the marketplace lists `pf` with source `./plugins/pf`, and that the commands reference only existing tools

### Requirement: /pf:join
`/pf:join <game link> [name]` SHALL join the match at the game link and take a seat without submitting a script:
- It calls `join_match` with the link, the given name (or asks for one), and no script.
- It saves `{link, match_id, seat, token, name}` to `.prompt-fighter/match.json` in the working directory.
- It creates `fighter.pf`, with a starter script from the player guide or a sample bot, unless that file already exists, and validates it.
- It tells the player the seat they got, that nothing has been submitted, and that each round they can ask for changes to `fighter.pf` and then run `/pf:submit`.

It SHALL NOT submit or rewrite any script beyond creating the starter.

#### Scenario: Join a hosted match
- **WHEN** a player runs `/pf:join http://192.168.1.20:8080/play/matches/0123456789abcdef claude-bot`
- **THEN** the LLM joins as the first free seat, writes `.prompt-fighter/match.json` with the seat token, creates `fighter.pf`, and reports that round 1 waits for `/pf:submit`

### Requirement: /pf:submit
`/pf:submit` SHALL submit `fighter.pf` for the next round exactly as it is on disk:
- It reads `.prompt-fighter/match.json` and `fighter.pf`.
- It validates the script with `validate_script`.
- It then calls `submit_script` with the saved link and token.

If the script is invalid, it SHALL report the errors and stop without changing the file or submitting. If no match has been joined, it SHALL say to run `/pf:join` first. After submitting, it reports the match state: whether the round ran, and if so its result and how to study the replay.

#### Scenario: Submit a tweaked script
- **WHEN** the player has asked the LLM to tweak `fighter.pf` and runs `/pf:submit`
- **THEN** the exact file contents are validated and submitted, and the reply says whether the opponent has submitted yet

#### Scenario: Invalid script
- **WHEN** `fighter.pf` contains `kik`
- **THEN** `/pf:submit` reports the validation error with its line and column, and nothing is submitted
