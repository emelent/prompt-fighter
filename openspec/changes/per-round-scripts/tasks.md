# Tasks

The implementation already exists. Each task checks it against the spec and fixes anything that doesn't match.

## 1. Plugin

- [ ] 1.1 Check the three commands:
  - `plugins/pf/commands/join.md` (join only, no `.pf`, no `submit_script` in allowed tools)
  - `script.md` (N from `next_round`, writes and validates `roundN.pf`, studies round N-1 from round 2 on, never submits)
  - `submit.md` (submits `roundN.pf` unchanged, points to `/pf:script` when the file is missing)

  Verify `go test ./plugins` passes (three commands, only submit may submit) and `claude plugin validate plugins/pf` passes.
- [ ] 1.2 Play out the commands' tool sequences for two players against a local `pf serve` with a scripted MCP client:
  - joins leave no `.pf` file
  - submit before script finds no `round1.pf`
  - `round1.pf` is written and validated
  - round 1 runs on the second submission
  - the next script is `round2.pf`

## 2. Web

- [ ] 2.1 Check the setup view: six separate single-line copy fields (game link, then the five commands: marketplace, install, join, script, submit), and the round-1 wording. Verify `npm test` passes, including the single-line check, and that in headless Chrome each Copy puts exactly one command on the clipboard.
- [ ] 2.2 Check the menu music on the waiting screen: in headless Chrome, after Create match it plays without another click, keeps playing through the countdown, and on the round page the fight track plays and the menu track isn't loaded.

## 3. Docs and wrap-up

- [ ] 3.1 Check that README, `docs/PLAYER_GUIDE.md` and `docs/API.md` describe join → script → submit and the waiting-screen music. Verify `go vet ./...` and `go test ./...` pass.
