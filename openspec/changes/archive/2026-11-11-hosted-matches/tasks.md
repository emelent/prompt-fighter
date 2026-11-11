# Tasks

## 1. Server: hosted matches and first-free-seat join

- [x] 1.1 Add `waiting_for_players`, `newHostedMatch`, and `join` returning the assigned seat (a then b) to `server/match.go`, per design §2. Extend `server/match_test.go`, which verifies `go test ./server` passes:
  - hosted match starts with no seats
  - first join → a and `waiting_for_opponent`
  - second join → b and round 1 played with seat a as p1
  - third join → `errSeatTaken`
  - a player-created match still joins as b
- [x] 1.2 Update `handleCreateMatch` (empty body → hosted, `{match_id, match}`; one of name/script → 400) and `handleJoin` (assigned seat in the response; 409 when full) in `server/matches.go`, per design §1. Add HTTP tests for each spec scenario: host a match, players join, half a create request, seat already taken. Verify `go test ./server` passes.

## 2. MCP

- [x] 2.1 Add `parseMatchRef` to `mcpserver` and use it in `join_match`, `get_match`, `submit_script` and `get_round_replay`, per design §3:
  - accepts a bare id or a game link
  - a link to another origin → an error naming both origins and the `--server` fix
  - `join_match` description and results name the assigned seat, and summaries cover `waiting_for_players`

  Add tests: the link parse table, a join by link against a test server, the mismatch error, and a join into a hosted match returning seat a. Verify `go test ./mcpserver` passes.

## 3. Cover image

- [x] 3.1 Rename `web/convert-audio.sh` to `web/convert-assets.sh`, and add the cover resize (800 px wide JPEG into `web/src/images/cover.jpg`). Add the `assets` npm script and keep `audio` as an alias. Run it and verify the output is under 300 KB.

## 4. Web

- [x] 4.1 Rewrite `lobbyPage` in `web/src/play.ts` as the cover screen, per design §4:
  - the cover image, Create match (hosted → `/watch/matches/{id}`) and Join match (id or link → Join as a player or Watch), with errors
  - keep the menu music toggle

  Add responsive styles in `style.css` with no inline styles. Verify `npm run check` passes.
- [x] 4.2 Update `web/src/playstate.ts` (join when either seat is free; `waitingOpponent` for whichever seat is held; `waiting_for_players`) and `web/src/match.ts` (status text). Extend `playstate.test.ts` for a hosted match with no seats, one seat a, and a full match. Verify `npm test` passes.
- [x] 4.3 Add the `setup` state to `waitingState` in `web/src/spectate.ts`, and render the setup view in `web/src/watch.ts` per design §5: joined n/2 and names, the game link and Copy, the MCP command and Copy, and the LLM prompt and Copy. Extend `spectate.test.ts`: setup with 0, 1 and 2 joined, and the game link and origin. Verify `npm test` passes.

## 5. Docs and end-to-end check

- [x] 5.1 Update `docs/API.md` (hosted create, first-free-seat join, the new status, MCP links), `docs/PLAYER_GUIDE.md` §11 (joining from a game link with an LLM), and the README (host a match, share the game link). Verify `go test ./docs ./server` passes.
- [x] 5.2 Rebuild `web/dist` and `pf`, start `pf serve` on a fresh data dir, and walk through it in headless Chrome:
  1. The cover screen renders with the image.
  2. Create match lands on the setup view, showing 0/2, the game link and the instructions.
  3. Player 1 joins in the web UI from the game link and gets seat a; the host shows 1/2.
  4. Player 2 joins with `join_match` through `pf mcp` given the game link (scripted over stdio) and gets seat b.
  5. Round 1 is played: the host counts down, plays the replay, and Done returns to the waiting screen.
  6. The players submit round 2 (one web, one MCP), and the host advances again.

  Also check:
  - Join match → Watch works for a full match
  - no console errors

  Review screenshots. Then run `go vet ./...` and `go test ./...`, and add follow-ups to `docs/ISSUES.md`.
