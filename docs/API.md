# Prompt Fighter — HTTP API (Phase 2)

`pf serve` runs a JSON-over-HTTP server for validating fight scripts, simulating single rounds and playing best-of-3 matches. The server is built for **local, trusted use**: it binds to `127.0.0.1:8080` by default and has no user accounts. Seat tokens only stop the two players in a match from submitting for each other.

```sh
pf serve                                  # 127.0.0.1:8080, data in ./data
pf serve --addr 0.0.0.0:9000 --data /var/lib/pf
```

The server logs one line on start, and SIGINT or SIGTERM shuts it down gracefully (exit 0). A bad address or an unusable data directory exits with code 2.

Engine rules, frame data and the event log format are described in [`RULES.md`](RULES.md). Versions in this document: config `2026.11-4`, log format `1`, store `1`.

## Playing with an LLM (MCP)

`pf mcp` exposes the same functionality to an LLM client as Model Context Protocol tools over stdio. The match tools call this HTTP API at `--server`, which defaults to `http://127.0.0.1:8080`; pass `--server ""` to disable them. Local tools (`validate_script`, `simulate_round`, `list_sample_bots`, `get_player_guide`) work without a server.

```sh
claude mcp add prompt-fighter -- pf mcp                            # Claude Code
claude mcp add prompt-fighter -- pf mcp --server http://host:9000  # another pf serve
```

Claude Desktop (`claude_desktop_config.json`):

```json
{ "mcpServers": { "prompt-fighter": { "command": "pf", "args": ["mcp"] } } }
```

Every match tool's `match_id` can be the bare id or a **game link** (`http://host:port/play/matches/<id>`), so you can hand a model the link from a host's waiting screen. With a link, the tool talks to the `pf serve` server in the link, so one `pf mcp` works with any host (a bare id uses `--server`). `join_match` takes the first free seat and says which one you got; its `script` is optional, and without one you submit round 1 later with `submit_script`.

The [player guide](PLAYER_GUIDE.md) is what the model should read first. It's available as the `get_player_guide` tool and the `promptfighter://docs/player-guide` resource.

## Endpoints

| method | path | purpose |
|---|---|---|
| GET | `/healthz` | status and versions |
| POST | `/validate` | validate one script |
| POST | `/sim` | simulate one round, return the event log |
| POST | `/matches` | create a match (take seat `a`) |
| POST | `/matches/{id}/join` | join a match (take seat `b`); plays round 1 |
| POST | `/matches/{id}/scripts` | submit your script for the next round (seat token required) |
| GET | `/matches/{id}` | match state |
| GET | `/matches/{id}/rounds/{n}` | event log of played round `n` |
| GET | `/view` | replay viewer home page (HTML) |
| GET | `/view/matches/{id}` | live match page (HTML) |
| GET | `/view/matches/{id}/rounds/{n}` | round replay player (HTML) |
| GET | `/view/static/{file}` | viewer assets (JS/CSS) |
| GET | `/play` | lobby: create or join a match (HTML) |
| GET | `/play/matches/{id}` | play page: write, validate and submit scripts; waits and auto-plays each round (HTML) |
| GET | `/watch/matches/{id}` | spectator waiting screen: round, score, who has submitted; plays the menu music and counts down into each new round, where the fight music takes over (HTML) |
| GET | `/watch/matches/{id}/rounds/{n}` | spectator round replay in the simple player (HTML) |
| GET | `/{$}` | the exact root `/`: redirects to `/play` |

All API requests and responses are JSON, and unknown JSON fields are rejected. The `/view` pages and assets are the only non-JSON responses (see Replay viewer).

### `GET /healthz`

```json
{"config_version":"2026.11-4","format_version":"1","status":"ok","store_version":1}
```

### `POST /validate`

Request: `{"script": "<source>"}`. An invalid script is still a successful request (200):

```sh
curl -s localhost:8080/validate -d '{"script": "kik"}'
```
```json
{"errors":[{"line":1,"col":1,"message":"unknown action \"kik\" — did you mean \"kick\"?","text":"1:1: unknown action \"kik\" — did you mean \"kick\"?"}],"valid":false}
```

`text` is exactly the line `pf validate` prints, minus the file name. A valid script returns `{"errors":[],"valid":true}`.

### `POST /sim`

Request:
```json
{"p1": {"name": "alice", "script": "loop { kick }"},
 "p2": {"script": "loop { high_block hold 8 }"},
 "frames": 300}
```
- `name` is optional. It defaults to `p1` or `p2` and is at most 40 bytes.
- `frames` is optional. It defaults to the server's round length (1200 frames unless `pf serve --frames N` changed it) and must be from 1 to that length.

The response (200) is the round's event log, **byte for byte what `sim.Simulate` and `pf sim --json` produce**. Its format is described in [`RULES.md`](RULES.md) §7. If either script is invalid, the response is 422 `invalid_script` and each error says which `player` it belongs to.

## Matches

A match is a best-of-3 between two **seats**, `a` and `b`. Before each round both seats submit a script, so you can watch the replay, ask your LLM for a fix, and come back with a revised script.

**Rules:**
- **When a round runs:** as soon as both seats have submitted. It runs inside the request that completed the pair, and runs exactly once.
- **Sides alternate:** seat `a` is p1 in odd rounds and seat `b` is p1 in even rounds.
- **Winning:** the first seat to win **2 rounds** wins the match. Drawn rounds count for nobody.
- **Round cap:** at most **5 rounds**. After round 5, the seat with more wins takes the match, or the match is drawn (`winner: null`) if they're level.
- **Keeping or replacing a script:** resubmitting the same script keeps it. You can resubmit as often as you like before the round runs; the latest submission counts.
- **Statuses:** `waiting_for_players` (hosted match, nobody joined yet) → `waiting_for_opponent` → `awaiting_scripts` → `finished`.

**Visibility:**
- Match state and played-round logs need no token. Anyone with the match id can read them, so replays can be shared.
- A seat's **pending (unplayed) script is never shown** to anyone. The state only says `"submitted": true`. Scripts become visible in a round's log once that round has been played.

### Create: `POST /matches`

```sh
curl -s localhost:8080/matches -d '{"name": "alice", "script": "loop { kick }"}'
```
```json
{"match_id":"b8fc4050df5bfab2","seat":"a","token":"db0bbc0b901149a158f2fa29e390cd42",
 "match":{"match_id":"b8fc4050df5bfab2","status":"waiting_for_opponent","config_version":"2026.11-4","next_round":1,
          "seats":{"a":{"name":"alice","wins":0,"submitted":true},"b":null},"rounds":[],"winner":null}}
```
**Keep the `token`: it is returned only once.** Names are 1–40 bytes, and the script must be valid, or the response is 422. Your create script is your round-1 submission.

**Hosting a match without playing:** post an empty object. Both seats stay free, and players take them by joining:

```sh
curl -s localhost:8080/matches -d '{}'
```
```json
{"match_id":"5c1f0e2a9b7d4e63",
 "match":{"match_id":"5c1f0e2a9b7d4e63","status":"waiting_for_players","config_version":"2026.11-4","next_round":1,
          "seats":{"a":null,"b":null},"rounds":[],"winner":null}}
```
There is no seat or token in the response: the host only watches (for example on `/watch/matches/{id}`). Sending only one of `name` and `script` is a 400.

### Join: `POST /matches/{id}/join`

`{"name"}` with an optional `"script"`. You take the **first free seat**: seat `a` in a hosted match nobody has joined yet, otherwise seat `b`. The response is 201 with that `seat` and your token. Without a script you join with nothing submitted, and submit round 1 like any other round. A script given here counts as your round-1 submission. Every round, round 1 included, runs as soon as both seats have submitted, so a join that brings the second round-1 script runs round 1 right away:

```json
{"match_id":"b8fc4050df5bfab2","seat":"b","token":"94098b550a8d72679c12e28299bda923",
 "match":{"match_id":"b8fc4050df5bfab2","status":"awaiting_scripts","config_version":"2026.11-4","next_round":2,
          "seats":{"a":{"name":"alice","wins":0,"submitted":false},"b":{"name":"bob","wins":0,"submitted":false}},
          "rounds":[{"n":1,"p1":"a","winner":null,"reason":"timeout","end_frame":1199,"hp":{"a":200,"b":200},
                     "config_version":"2026.11-4","log":"/matches/b8fc4050df5bfab2/rounds/1"}],
          "winner":null}}
```
Joining a match whose seats are both taken returns 409.

### Submit: `POST /matches/{id}/scripts`

```sh
curl -s localhost:8080/matches/$ID/scripts \
  -H "Authorization: Bearer $TOKEN" -d '{"script": "loop { sweep }"}'
```
The response is 200 with `{"match": <state>}`. If your submission completed the pair, the new round is already in `rounds`.

| case | response |
|---|---|
| missing or wrong token | 401 |
| match already finished | 409 |
| invalid script | 422; your earlier submission stays in place |

### State: `GET /matches/{id}`

The response is the match state object shown above. Fields:

| field | meaning |
|---|---|
| `status` | the match status |
| `next_round` | the round that runs next |
| `seats.{a,b}` | `name`, `wins`, and `submitted` (has a pending script for the next round) |
| `rounds[]` | `n`, `p1` (the seat that played p1), `winner` (seat or null), `reason` (`ko`/`double_ko`/`timeout`), `end_frame`, `hp` per seat, `config_version` (the engine config the round used), `log` (the URL of its event log) |
| `winner` | the winning seat, or null while playing or for a drawn match |

Clients poll this endpoint to see when the opponent has submitted or a round has run.

### Round log: `GET /matches/{id}/rounds/{n}`

Returns the stored event log of played round `n`. It's byte-identical to `sim.Simulate` for that round's scripts, names and sides. Its `players[]` holds the p1 and p2 names and full scripts as played. An unplayed round returns 404.

## Playing in the browser

`pf serve` hosts the submission UI. Opening the server root (`GET /`) redirects to the lobby.

| URL | page |
|---|---|
| `/play` | **lobby**: create a match (name + round-1 script), open one by id, or watch one (id or any match link). Plays the menu music on a loop after the first click; the "sound" toggle turns it off |
| `/play/matches/{id}` | **play page**: the page each player keeps open during a match. It's also the invite link to send your opponent. |

**What the play page shows,** depending on the match and on whether this browser holds a seat:

| state | when |
|---|---|
| join | you hold no seat and seat b is free |
| spectator | both seats are taken; links to the replay viewer |
| waiting for an opponent | you created the match; shows the invite link with a copy button |
| your move | submit your script for round N |
| waiting for their script | you've submitted; you can still replace it until the round runs |
| finished | the winner and links to every round's replay |

**Editor:**
- CodeMirror 6 with `.pf` highlighting.
- Live validation through `POST /validate`, after a 300 ms pause in typing. Errors are marked at their line and column with the same messages as `pf validate`.
- A byte counter against the 4096-byte limit.
- Create, Join and Submit stay disabled until the current text has validated.
- Any sample bot can be loaded as a starting point.

**Auto-play:** the play page polls the match every 2 s. When a round resolves, whether through your submission or your opponent's, it opens `/view/matches/{id}/rounds/{n}?from=play&autoplay=1`. That replay starts playing immediately, and its back link returns to the play page.

**Seats and drafts:**
- Your seat token and your editor draft are kept in this browser's `localStorage`, keyed by match id, so a reload keeps your seat.
- "forget this seat" removes the stored seat.
- If storage is blocked, the page says that your seat won't survive a reload.
- The token is what lets you submit for your seat, so don't share it.

The pages use only the API above; no extra endpoints exist for them. They're served with the same strict Content-Security-Policy as the viewer. The editor renders inside a shadow root, so its styles need no CSP exception.

## Watching a match

Anyone can watch a match without a seat or token. Open the lobby's **Watch a match** box, or the watch link each player sees on their play page (`/watch/matches/{id}`, with a copy button). Spectators only read `GET /matches/{id}` and the round logs; they can't change anything.

| URL | page |
|---|---|
| `/watch/matches/{id}` | **waiting screen**: the round being waited for, the score, who has submitted and who we're waiting for, and links to played rounds. It polls every 2 s. When a round is played it shows "Round N starting in 5… 1" (with a **Watch now** button) and opens it. When the match is over it shows the result. Opened directly, it counts the rounds already played as watched and waits for the next one |
| `/watch/matches/{id}/rounds/{n}` | **spectator replay** in the simple round player. At the last frame: **Next round** with a 5-second countdown if the next round has been played (the page keeps polling, so this appears as soon as it is), otherwise **Done**, back to the waiting screen |

**The simple round player** is used by spectator replays and by players watching their own rounds from the play page (`?from=play`, where **Done** returns to the play page):
- Only play/pause, rewind (back to the start) and a ⚙ settings button are shown. Playback starts at 0.5× with sound on, and the scripts are hidden.
- The settings menu turns on the scripts, event feed, timeline and frame steps, speed control, checks and ranges, and sound.
- Choices and the chosen speed are remembered in the browser (`localStorage`, key `pf.simple`). Keyboard shortcuts still work.

The `/view` pages below keep the full controls.

## Replay viewer

`pf serve` also hosts a browser replay viewer, built with Phaser 3 (pinned at 3.90.0), strict TypeScript and Vite, and embedded in the `pf` binary:

| URL | page |
|---|---|
| `/view` | home: open a match by id, or open a local log file (file picker or drag-and-drop) from `pf sim --json`, `POST /sim` or a saved round. Local files are read in the browser and never uploaded. |
| `/view/matches/{id}` | live match page: seats, wins and rounds, polled every 2 s. With "open new rounds automatically" on (the default), it jumps to each new round as it's played. |
| `/view/matches/{id}/rounds/{n}` | round player: a pixel-art stage (one gi-wearing fighter design, teal for p1 and violet for p2, animated from each frame's `anim`, `phase` and `anim_frame`), HP bars, hit and block sparks, callouts (`BLOCKED`, `THROW`, `TECH`, `TRADE`, `K.O.`), combo counters, a frame countdown, an event feed (the same lines as `pf sim`; click one to jump to its frame) and both scripts with each fighter's executing line highlighted |

**Frame links:** add `#f=N` to a round URL to open it paused at frame N (for example `/view/matches/{id}/rounds/2#f=412`). While paused, the address bar always holds the current frame, so you can copy it to share that moment.

**Keyboard:**

| key | action |
|---|---|
| Space | play/pause |
| ← / → | 1 frame back / forward |
| Shift+← / Shift+→ | 10 frames back / forward |
| Home / End | first / last frame |
| `[` / `]` | slower / faster (0.25×–4×; 1× = 60 frames per second) |
| `F` | full screen stage (also the ⛶ button on the stage; Esc leaves it) |

**Round intro:** playing from frame 0 (pressing play there, replaying from the end, or `autoplay=1`) first shows `ROUND n` (or `FINAL ROUND` for round 3) and then `FIGHT!`, for about 1.5 s at any speed. Local files show only `FIGHT!`. The intro sits before frame 0, so the scrub bar, panels and feed stay on frame 0. Stepping, seeking or pausing skips it, and `#f=N` links never show it.

**Frame countdown:** the number at the top centre is the frames left in the round, `frame_budget − 1 − f` from the log (1199 at frame 0 of a default round, 0000 when a round times out).

**Sound:** hits, misses, blocks (guard breaks included) and jumps play sound effects as playback advances (playing or stepping forward; scrubbing and jumping are silent), with background music: each page picks one `fight_music_*` track at random, starts it when the round first plays, and keeps it looping through pauses and after the round ends. Punch-type attacks (`punch`, `dive_punch`, `low_jab`) and kick-type attacks (`kick`, `sweep`, `jump_kick`) have their own hit and miss sounds, and a successful throw uses the kick hit. Browsers start audio only after the first click or key press on the page. The "sound" toggle turns it off and is remembered in the browser. The sources are the files in `audio/`. The viewer uses MP3 copies in `web/src/audio/`, made by `npm run audio` in `web/` (needs ffmpeg); a new `fight_music_N` file joins the rotation after `npm run audio && npm run build`.

The "checks" toggle adds sensor reads to the event feed, and "ranges" draws each active attack's full reach. The drawn limbs are illustrative and shorter than the real reach (a punch reaches 8 units, about 70 px), so use "ranges" to see what can connect.

**Security:**
- Viewer responses carry a strict Content-Security-Policy: same-origin scripts, styles and connections only, no inline scripts, no framing.
- They also send `X-Content-Type-Options: nosniff` and `Referrer-Policy: no-referrer`.
- Script text and names are always shown as text, never as HTML.

**Developing the viewer:** the sources are in `web/src`. The build (`web/dist`) is not committed: `make` (or `npm run build` in `web/`) builds it before `pf` is compiled, because `pf` embeds it.

```sh
cd web
npm ci            # exact versions from package-lock.json
npm run dev       # Vite dev server; proxies /matches and /healthz to pf serve on :8080
npm run check     # tsc --noEmit (strict) + Vitest + vite build (refreshes dist/)
```

`go test ./web` checks that the embedded build is self-consistent. It also runs the frontend tests when `web/node_modules` is installed, and skips them otherwise.

## Errors

Every error uses the same envelope:

```json
{"error": {"code": "invalid_script", "message": "script is invalid"},
 "errors": [{"player": "p2", "line": 1, "col": 1, "message": "...", "text": "1:1: ..."}]}
```
`errors` appears only with `invalid_script`. `player` is `p1`/`p2` on `/sim` and `a`/`b` on match endpoints.

| code | status | when |
|---|---|---|
| `bad_request` | 400 | malformed JSON, unknown or missing field, value out of range (name length, frames) |
| `unauthorized` | 401 | missing, malformed or unrecognised seat token |
| `not_found` | 404 | unknown route, match or unplayed round |
| `method_not_allowed` | 405 | wrong HTTP method (the response includes an `Allow` header) |
| `conflict` | 409 | seat `b` already taken, or submitting to a finished match |
| `payload_too_large` | 413 | request body over 64 KiB |
| `invalid_script` | 422 | a script failed validation |
| `internal` | 500 | unexpected server failure, such as a storage write error |
| `busy` | 503 | no simulation slot came free within 5 s (`/sim`) |

## Limits

- The request body is limited to 64 KiB. Scripts themselves are limited to 4096 bytes by the language (see `RULES.md` §8).
- Simulations run concurrently up to the number of CPUs. A request waits up to 5 s for a free slot.
  - On `/sim`, if no slot comes free it gets 503 `busy`.
  - On a match submission it still succeeds, and the submissions stay pending (both seats show `submitted: true`). The round then runs on the next submission to that match; either seat may simply resubmit.
- The server timeouts are: read header 5 s, read 10 s, write 30 s, idle 60 s.

## Storage

```
<data>/matches/<match_id>/match.json        match state, store_version 1
<data>/matches/<match_id>/rounds/<n>.json   exact event log of round n
```
- **Writes:** every file is written atomically (temp file, fsync, rename), so a crash leaves either the old file or the new one, never a partial one. Within a round, the log is written before `match.json`.
- **Restarts:** the server rereads from disk on every request, so a restart loses nothing, pending submissions included.
- **Tokens:** only stored as SHA-256 hashes.

**Limitations (Phase 2):**
- **One process per data directory.** Match locking is in-process only, so don't run two servers on the same `--data`.
- **No cleanup.** Matches are never deleted or expired.
- **No idle-seat timeout.** A seat that never submits stalls its match.
