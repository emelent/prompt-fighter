# Prompt Fighter

A fighting game where you don't press buttons: you write a fight script in a small language (`.pf`), and two scripts fight a best-of-3 match. You can write scripts yourself or have an LLM write them for you. The rules punish turtling, running away and spamming one move (guard breaks, passivity boosts and stale attacks), and reward long combos.

## Get pf

Download `pf` for your platform from the [GitHub Releases](https://github.com/emelent/prompt-fighter/releases) page (one binary: server, CLI and MCP server), or build it with Go 1.27+ and Node 24+:

```sh
make            # builds the web app, then bin/pf
bin/pf version
```

`make test` runs all tests, `make release` builds archives for every platform in `dist/`, and `make smoke` runs the MCP smoke test. The web app (`web/dist`) is a build output, not committed: `make` builds it before compiling `pf`, which embeds it.

## Host the game

```sh
pf serve                       # http://127.0.0.1:8080, matches stored in ./data
pf serve --addr 0.0.0.0:8080   # let others on your network join
pf serve --frames 1800 --hp 300  # longer rounds (defaults: 1200 frames, 200 HP)
```

Open http://127.0.0.1:8080/ for the cover screen. **Create match** hosts a match without playing: you get a waiting screen with the **game link**. Send it to two players. Each round runs once both have submitted a script; the waiting screen counts down and plays it, and **Done** returns to it for the next round.

Anyone else can watch: pick **Join match**, paste the game link and choose **Watch**.

The server is built for trusted, local use. It has no accounts, and seat tokens only stop the two players from submitting for each other. Don't expose it to the internet.

## Play in the browser

Open the game link, pick a name to take a seat, then write your round-1 script in the editor (or start from a sample bot) and submit it. After each round its replay plays; then revise your script and submit the next one.

## Play with an LLM (Claude Code)

The `pf` plugin lets your LLM write and tweak your script while you decide when to submit. It needs `pf` on your PATH.

1. Install the plugin once. Run these two commands in Claude Code **one at a time** (pasting both lines together makes Claude Code read them as a single command):
   ```
   /plugin marketplace add emelent/prompt-fighter
   ```
   ```
   /plugin install pf@prompt-fighter
   ```
2. Join with the game link the host sent you. This only takes a seat; no script is written or submitted:
   ```
   /pf:join http://192.168.1.20:8080/play/matches/<id> my-fighter
   ```
3. Every round, round 1 included, describe how you want to fight. The LLM writes that round's script to `roundN.pf` (`round1.pf`, `round2.pf`, …), studying the previous replay from round 2 on, and validates it. Run it again to revise:
   ```
   /pf:script rush in with kicks, and block low when they start sweeping
   ```
4. When you're happy with it, send it. `/pf:submit` submits `roundN.pf` for the current round unchanged; if it has errors it reports them and submits nothing:
   ```
   /pf:submit
   ```
   The round runs once both players have submitted.

**Other MCP clients** (or without the plugin): add the MCP server yourself, then ask the model to use the `prompt-fighter` tools. Match tools accept the game link and talk to the server in it.

```sh
claude mcp add prompt-fighter -- /full/path/to/pf mcp      # Claude Code
```
```json
{ "mcpServers": { "prompt-fighter": { "command": "/full/path/to/pf", "args": ["mcp"] } } }
```
(Claude Desktop's `claude_desktop_config.json`. GUI apps may not see your shell's PATH, so use the full path.)

Without a match, the model can still use `validate_script`, `simulate_round`, `list_sample_bots` and `get_player_guide` to practise.

## Releases

Every push to `main` that passes the tests is released automatically. The version comes from [conventional commit](https://www.conventionalcommits.org/) messages since the last release: `feat:` bumps the minor version, `fix:` (and anything else) the patch version, and `feat!:` or a `BREAKING CHANGE:` footer the major version (the minor version while below 1.0). `go run ./tools/nextversion` prints the version the next push would get.

## More

- [Player guide](docs/PLAYER_GUIDE.md): the language, the moves and strategy. This is what the LLM reads.
- [Rules](docs/RULES.md): frame data and engine details.
- [HTTP API and MCP tools](docs/API.md).
- Command line: `pf validate my.pf`, `pf sim my.pf scripts/turtle.pf`, `pf selfplay scripts/`, `pf version`.
