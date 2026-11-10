# Tasks

## 1. Pixel-art fighter model (pure)

- [x] 1.1 Create `web/src/fighterart.ts`. Port from the preview:
  - the skeleton pose table, the joint solver and the rasteriser into a 72×64 palette-index (widened from the preview's 48, whose punch, kick and lying poses were clipped) `Uint8Array` grid (back limbs, torso, belt, head with topknot and eye, front limbs, outline pass)
  - the `PALETTES` for p1 (teal) and p2 (violet)

  Strict TS with no DOM access. Verify `npm run check` passes.
- [x] 1.2 Add `choosePose(ff, logFrame, win): { pose, fallback }` (named to avoid clashing with `scene.ts`'s `poseFor`) to `fighterart.ts`. It maps every format-"1" anim through `phase` and `anim_frame` as in design §2, with the `WIN` pose for the winner on the last frame and a state-based standing fallback for unknown anims.

  Verify with `web/test/fighterart.test.ts`:
  - startup, active and recovery of punch, kick, sweep, throw and low_jab give three different poses
  - jump changes pose by `anim_frame`
  - walk cycles
  - unknown anim falls back without throwing
- [x] 1.3 Add rasteriser tests to `fighterart.test.ts`:
  - every pose's grid is non-empty and within bounds
  - every non-empty pixel that borders an empty one is drawn in the outline colour
  - grounded poses touch the bottom rows, and air poses do not
  - p1 and p2 colourings of the same pose differ only in gi, skin and hair colours

  Verify the tests pass.
- [x] 1.4 Add a coverage test that walks every frame of every golden log (`web/test/fixtures.ts`) for both fighters, and asserts every observed (state, anim, phase) maps to a defined pose without the fallback. Verify it passes.

## 2. Callouts (pure)

- [x] 2.1 Create `web/src/callouts.ts` with `calloutsAt(log, f)` implementing the spec's table (`BLOCKED`, `THROW`, `TECH` once, `TRADE` once, `K.O.`), a 30-frame lifetime (`K.O.` lasts to the end), and `age`. Verify `npm run check` passes.
- [x] 2.2 Add `web/test/callouts.test.ts`:
  - `BLOCKED` above the defender 5 frames after a block in `rushdown_vs_turtle`, and gone 40 frames after
  - `TRADE` in `turtle_vs_jumper`
  - `THROW` in `lazy_vs_thrower`
  - a single `TECH` from a synthetic two-throw tech log
  - `K.O.` on the last frame of KO rounds and never in the timeout round
  - the same result whether frames are visited in order or by jumping straight to one

  Verify the tests pass.

- [x] 2.3 Add `countdownAt(log, f)` (design §6) with tests:
  - a 600-budget log reads `599` at frame 0, `349` at frame 250 and `000` at frame 599 of `rushdown_vs_turtle`
  - a synthetic 300-budget log reads `299` at frame 0

  Verify the tests pass.

## 3. Round intro in playback

- [x] 3.1 Add the intro state to `web/src/playback.ts` per design §7:
  - `play()` at frame 0 starts it
  - `tick()` spends real time on it at any speed before advancing frames
  - `seek()`, `step()` and `pause()` end it and emit

  Extend `web/test/playback.test.ts`:
  - play from 0 holds frame 0 for 1500 ms at 1× and at 0.25×, then advances
  - replaying from the end starts the intro again
  - → during the intro lands paused on frame 1
  - pausing ends it
  - play from frame 100 has no intro

  Verify the tests pass.
- [x] 3.2 Add the optional `roundLabel` to `StageSource`, and set it in `web/src/app.ts`: `ROUND n`, `FINAL ROUND` for round 3, and null for local files. Make sure a `#f=N` link never starts the intro. Verify `npm run check` passes and the existing app and play tests are unchanged.

## 4. Stage rendering in Phaser

- [x] 4.1 Create `web/src/pixelfont.ts` and the backdrop generator as pure grid or RGBA producers:
  - The font has 5×7 glyphs for A–Z (lower case drawn as capitals), 0–9 and common punctuation. A missing glyph draws as a blank box.
  - The backdrop is 320×180: sky bands with dithered seams, sun, two skyline layers with seeded windows, water tower, parapet, railing, and floor seams below row 162. It is fully opaque.

  Add tests:
  - every backdrop pixel is opaque, which guards against the preview's smearing bug
  - strings render to the expected width
  - unknown characters give a box and never throw
  - markup in a name such as `<img>` renders as glyphs

  Verify the tests pass.
- [x] 4.2 Rewrite the drawing in `web/src/stage.ts` per design §3–8:
  - a 960×540 game with `pixelArt: true`
  - the backdrop Image at 3×
  - lazily created `CanvasTexture`s per (pose, palette)
  - fighter Images at scale 3, with feet origin, `setFlipX` by facing, jump at 3 px per unit, alpha from `FighterDraw.alpha`, and acting fighter in front
  - ground shadows
  - show-ranges reach line
  - pixel sparks from `flashes`
  - arcade HP bars draining to the centre
  - all text in the pixel font: names, combo counter, frame counter, countdown, callouts, banner
  - `K.O.` above the result banner on the last frame
  - the intro words while `playback.intro` is active, with fighters idle at frame-0 positions

  Keep the `update()` clock, `invalidate()` and a compatible `mountStage` signature. Verify `npm run check` passes and `play.ts` is unchanged.
- [x] 4.3 Build (`npm run build`), run `pf serve` against the demo data, and check in headless Chrome (puppeteer, as in earlier phases):
  - no console errors and no image requests
  - screenshots of the auto-play intro (`ROUND n` beat and `FIGHT!` beat), frame 0 with countdown `599`, an attack's active frame, a block with `BLOCKED`, a trade with `TRADE`, a knockdown (translucent, lying), and the last frame with `K.O.` and a readable banner
  - stepping startup→active→recovery changes the sprite
  - a `#f=120` link shows no intro
  - the page layout still fits at 1280×800

  Review the screenshots and fix any issues before ticking this task.

## 5. Build, docs and wrap-up

- [x] 5.1 Rebuild `web/dist` (embedded in `pf`), and bump `web/package.json` to 0.5.0. Verify `go vet ./...` and `go test ./...` pass, including `web/web_test.go`'s dist consistency and the `npm test` run.
- [x] 5.2 Update the round-player row in `docs/API.md`: pixel-art fighters, sparks, callouts, the frame countdown, the round intro (and what skips it), and the note that limbs are illustrative while "show ranges" shows true reach. Verify `go test ./docs` still passes.
- [x] 5.3 Update `docs/ISSUES.md`:
  - mark the Phase 4 "Sprites and animation" follow-up and the "BLOCKED/THROW/TRADE/KO callouts" item as done by `pixel-art-fighters`
  - keep `/replay?…` URLs and the Playwright smoke test open
  - add any new follow-ups found while implementing, such as character select or sound

  Verify by reading the updated sections.
