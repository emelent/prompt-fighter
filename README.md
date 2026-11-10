# Prompt Fighter

A fighting game where you don't press buttons: you write a fight script in a small language (`.pf`), and two scripts fight a best-of-3 match. You can write scripts yourself or have an LLM write them for you.

## Host the game

You need Go 1.27 or later. The web app is already built and embedded, so Node isn't needed.

```sh
go build -o pf ./cmd/pf
./pf serve                       # http://127.0.0.1:8080, matches stored in ./data
```

Open http://127.0.0.1:8080/ to reach the lobby. Create a match, write or pick a script, and send the invite link to your opponent. Each round's replay plays in the browser when the round is played.

To let others on your network join, bind to all interfaces:

```sh
./pf serve --addr 0.0.0.0:8080 --data /path/to/data
```

The server is built for trusted, local use. It has no accounts, and seat tokens only stop the two players from submitting for each other. Don't expose it to the internet.

## Play with an LLM

`pf mcp` gives an MCP client (Claude Code, Claude Desktop, and others) tools to read the player guide, validate scripts, spar against sample bots, and play matches on your `pf serve` server.

1. Start the server: `./pf serve`.
2. Add the MCP server to your client, using the full path to the `pf` binary:

   ```sh
   claude mcp add prompt-fighter -- /full/path/to/pf mcp                                   # Claude Code
   claude mcp add prompt-fighter -- /full/path/to/pf mcp --server http://192.168.1.20:8080 # another machine's server
   ```

   For Claude Desktop, add this to `claude_desktop_config.json`:

   ```json
   { "mcpServers": { "prompt-fighter": { "command": "/full/path/to/pf", "args": ["mcp"] } } }
   ```

3. Ask the model to play, for example:

   > Read the Prompt Fighter player guide, spar against the sample bots until your script beats most of them, then create a match called "claude" and tell me the invite link.

   Your opponent joins from the invite link in the browser, or with their own LLM ("join Prompt Fighter match `<id>` and play it"). Between rounds, ask the model to watch the replay with `get_round_replay` and revise its script. Open the match in the browser to watch the rounds as they're played.

Without a server, the model can still use `validate_script`, `simulate_round`, `list_sample_bots` and `get_player_guide`.

## More

- [Player guide](docs/PLAYER_GUIDE.md): the language, the moves and strategy. This is what the LLM reads.
- [Rules](docs/RULES.md): frame data and engine details.
- [HTTP API and MCP tools](docs/API.md).
- Command line: `pf validate my.pf`, `pf sim my.pf scripts/turtle.pf`, `pf selfplay scripts/`.
