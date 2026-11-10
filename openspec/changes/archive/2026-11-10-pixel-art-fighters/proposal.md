# Proposal

## Why

The replay viewer's fighters are still geometric placeholders: coloured rectangles with a limb line. That makes replays hard to read at a glance and nothing like the arcade fighter the game is meant to feel like. The user approved a code-generated pixel-art fighter in a preview (https://claude.ai/artifact/2WhyXuNsoasD56fWo7YWeW) and asked for it in the real viewer. The same work picks up two open follow-ups in `docs/ISSUES.md`: sprites and animation (Phase 4), and the "BLOCKED/THROW/TRADE/KO" callouts from the original Phase 4 plan.

## What Changes

- Replace the placeholder fighter drawing in the Phaser stage with animated pixel-art sprites:
  - One original gi-wearing martial artist (topknot, wrapped hands, black belt) as a single design for both players. It is not a likeness of any existing game's character.
  - P1 and P2 are palette swaps: teal and violet gis.
  - Every sprite frame is generated in code from skeleton poses and rasterised into a 72×64 pixel grid with outline and shading. There are no image files, so nothing is fetched and the existing CSP is unchanged.
  - One animation per engine `anim` (idle, check, wait, walk forward and back, punch, kick, sweep, throw, jump, jump kick, dive punch, landing, duck, low jab, high and low block, hitstun, blockstun, knockdown, ko), plus a win pose for the winner on the last frame.
  - The sprite frame is picked from the log's `phase` and `anim_frame`, so startup, active and recovery match the engine's frame data exactly.
- Replace the flat stage with a pixel-art rooftop backdrop at dusk, drawn in code: sky bands, skyline, railing and floor.
- Replace the circle flashes with pixel hit sparks (hit, throw) and block sparks.
- Add event callouts in a pixel font: `BLOCKED`, `THROW`, `TECH`, `TRADE` and `K.O.`, shown briefly at the fighters when those events happen.
- Restyle the HP bars in an arcade style: they drain towards the centre and keep the low-HP colour.
- Make the stage 16:9 (960×540 instead of 960×380), so the backdrop and the fighter-to-stage proportions match the approved preview.
- Draw all on-stage text in the pixel font: names, combo counter, frame counter, result banner and callouts.
- Add a frame countdown at the top centre: the frames left in the round, `frame_budget − 1 − f`, taken from the log. It reads 599 on frame 0 of a default round and 000 when a round times out.
- Add a round intro: a short "ROUND n" → "FIGHT!" lead-in (about 1.5 s) before frame 0.
  - It plays whenever playback starts from frame 0, including autoplay.
  - It is skipped by seeking, stepping, frame links or pausing.
  - It is not part of the timeline, so the frame counter, scrub bar and event feed are unaffected.
  - Round 3 shows "FINAL ROUND". A local log file with no round number shows "FIGHT!" only.
- Turn on Phaser pixel-art rendering (nearest-neighbour scaling, integer sprite scale).
- These stay unchanged:
  - what the names, combo counter, frame counter and result banner say, and the show-ranges overlay
  - the translucent look while invulnerable
  - controls, panels, the event feed and the play UI
- `buildScene` and its `Scene`/`FighterDraw` output are not changed. Sprite selection and callouts are new pure modules that read the log alongside it.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `replay-viewer`:
  - "Stage rendering" changes from colour-per-state shapes with a limb line to animated pixel-art fighters with per-phase sprite frames, a stage backdrop and sparks.
  - It also adds the 16:9 stage, pixel-font text and the frame countdown.
  - A new "Event callouts" requirement adds `BLOCKED`/`THROW`/`TECH`/`TRADE`/`K.O.`.
  - A new "Round intro" requirement adds the lead-in before frame 0.

## Impact

- **Code:**
  - `web/src/stage.ts` is rewritten (drawing only; the playback clock and `mountStage` API are kept).
  - New pure modules: `web/src/fighterart.ts` (poses, rasteriser, palettes, anim→pose selection), `web/src/pixelfont.ts` and `web/src/callouts.ts` (events→callouts).
  - Phaser config adds `pixelArt: true`, and the game size becomes 960×540, so the replay page's stage is taller.
  - `web/src/playback.ts` gains an intro phase.
  - `web/src/app.ts` passes the round label to the stage through an optional `StageSource` field. `mountStage` stays compatible.
- **Tests:** new Vitest suites for sprite selection, rasterising, the pixel font, callouts, the countdown and the playback intro. Existing `scene.test.ts` is untouched.
- **Build:** `web/dist` is rebuilt and committed, and `go test ./web` checks it is consistent. The bundle grows by a few KB of code and no assets. No new dependencies.
- **Docs:**
  - `docs/API.md`: the round player description.
  - `docs/ISSUES.md`: resolve the sprites follow-up and the callouts item. `/replay?…` URLs and the Playwright smoke test stay open.
