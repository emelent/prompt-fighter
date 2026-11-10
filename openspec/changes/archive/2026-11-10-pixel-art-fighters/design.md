# Design

## Context

- `web/src/stage.ts` is a Phaser 3.90.0 scene that redraws one `Graphics` object from `buildScene(log, f)` whenever the frame changes. It draws rectangles, a head circle and a limb line.
- `buildScene` (`web/src/scene.ts`) is pure and unit-tested. Its `FighterDraw` carries `pose`, `state`, `anim`, `limb.phase`, `facing`, `x`, `y` and `alpha`, but not `anim_frame`.
- The viewer runs under `img-src 'self' data: blob:` and loads nothing from third parties. `web/dist` is committed and embedded in `pf`.
- The approved look is the preview at `scratchpad/pixel/pixel-fighters.html` (published as an artifact). It is a single-file canvas-2D prototype: skeleton poses, a rasteriser into a 72×64 palette-index grid, palettes, a 5×7 pixel font, a backdrop generator and sparks.
- Engine facts the art must follow, checked against `sim/fighter.go`, the config tests and the golden logs:
  - `anim_frame` is 1-based within the current anim.
  - Recovery length varies: a blocked attack's recovery equals the attack's, and a landing-low air attack carries through.
  - `ko` is emitted on the last frame of the log.
  - `throw` with outcome `tech` and `trade` events exist, though no golden log has a tech.

## Goals / Non-Goals

**Goals:**
- Port the preview into typed, tested modules and render it with Phaser 3 APIs only.
- Never desync from the engine: the sprite shown for a frame is derived from that frame's data, not from wall-clock animation time.

**Non-Goals:**
- Any change to `buildScene`, the `Scene` type, the event log, the engine or the playback controls.
- The preview's choreographed bout and gallery. The fighters only ever do what the log says.
- Sound, per-player character choice, or new characters.
- `/replay?…` URLs and the Playwright smoke test. These remain ISSUES items.

## Decisions

### 1. Generate the art at boot into Phaser canvas textures, not image files
`fighterart.ts` builds palette-index grids (pure, no DOM). At `create()`, the stage turns each needed (pose, palette) into a `CanvasTexture` via `this.textures.createCanvas(key, 72, 64)` and `putImageData`. Fighters are `Phaser.GameObjects.Image`s whose texture key is swapped per frame. The backdrop is generated the same way, once, at 320×180 and drawn at 3×.

- **Why:** there are no binary assets to commit, review or keep in sync, and the CSP needs no change. Palette swaps are free, and the art stays editable as data.
- **Alternatives:**
  - A build script that writes PNG sprite sheets into `web/public`: adds a generation step and binary diffs, for no runtime gain.
  - Drawing each frame with `Graphics`: no pixel-exact outlines, and slower.

### 2. Choose sprite frames from `anim` + `phase` + `anim_frame`, not fixed segment lengths
The preview animated with fixed frame counts per pose segment. The viewer instead maps `(state, anim, phase, k=anim_frame)` to a pose with one table per anim. For example:
- punch: startup→`P0`, active→`P1`, recovery: first half `PR`, then `G`.
- jump: by `k` (rising `J0` for k ≤ 10, then `J1`).
- walk: a 4-pose cycle on `floor(k/8) mod 4`.
- idle, check and wait: a 2-pose breathe cycle on the *log* frame number.
- hitstun: `H0` for k ≤ 4, then `H1`.
- knockdown: `KD0`, `KD1`, then `KD2` near its end.

The stage gets `anim_frame` by reading `log.frames[f]` directly, the same log it already holds. `FighterDraw` is not changed.

- **Why:** phases come from the engine, so a blocked kick whose recovery is extended still shows recovery frames for exactly as long as the engine says. Scrubbing or frame links give the same sprite as playing.
- **Alternative:** add `animFrame` to `FighterDraw`. This is rejected because the user asked for `buildScene` to stay untouched, and the extra lookup is trivial.

### 3. Pixel scale and stage geometry
- The game becomes 960×540 (16:9), matching the preview's 384×216 proportions. `Scale.FIT` keeps it fitting the stage box, which simply gets taller. Phaser config gets `pixelArt: true` (nearest-neighbour scaling, `roundPixels`).
- Sprites draw at a fixed integer scale of 3 (216×192 on screen, most of it transparent), with origin at the feet (0.5, 1), and are mirrored with `setFlipX` for facing.
- The ground line moves to y 486, which is backdrop row 162, the rooftop's floor edge.
- Stage x mapping stays `40 + x × 8.8`.
- Jump height stays at 4 px per `y` unit. With 540 px of height, arcs have the headroom the 380 px stage lacked.

- **Why:** an integer scale keeps every art pixel square. 960×540 at 3× is exactly the backdrop's 320×180. The fighter is about 36% of the stage height, close to the preview's 30%.
- **Trade-off:** drawn limbs are shorter than true attack reach, because a punch reaches 8 units (70 px). "Show ranges" still draws the exact reach, so the hitbox truth is one toggle away. See Risks.

### 4. Palettes and the character
- One `PALETTES` table with 11 indices: outline, gi with shade and highlight, skin with shade, hair, belt, wraps with shade, and eye.
- P1 is teal (`#2b9a8d`) and P2 violet (`#8b4fc4`), as in the preview.
- The design is the preview's original character: topknot, wrapped hands, black belt, no headband.
- The old `PLAYER_COLORS` outline colours keep identifying players in the names and the HP bar frame, but the HP-bar fill becomes the arcade yellow-on-red. Name text uses each palette's gi highlight colour.

### 5. Callouts as a pure module plus cached word textures
`callouts.ts` exports `calloutsAt(log, f): Callout[]` (`{text, kind, anchor: 'p1'|'p2'|'mid'|'center', age}`), using the spec's table and a 30-frame lifetime. Its shape is like `flashesAt`: it scans events up to `f`, so the result depends only on `f`. The stage renders each distinct word once to a canvas texture using the 5×7 pixel font from the preview (ported into `fighterart.ts`, or a small `pixelfont.ts`). Callouts are drawn as Images that rise 1 px every 3 frames of age.

- **Layout (960×540):**
  - The HP bars sit at y 24–51.
  - The frame countdown is centred between them in the pixel font at scale 4.
  - Names sit under the bars, and the frame counter is small under the countdown at y 60.
  - Event callouts sit above the target's head.
  - `K.O.`, the intro words (`ROUND n`, `FINAL ROUND`, `FIGHT!`) and the result banner use the band between y 120 and y 300: `K.O.` at scale 6 around y 130, and the banner panel at y 220–300 below it.
  - Nothing overlaps the HP bars, countdown or frame counter.
- **Text:** every on-stage string, including names (player content, drawn as glyphs, never parsed), the combo counter, the banner and the countdown, goes through `pixelfont.ts`. The font covers A–Z, a–z (drawn as capitals), 0–9 and common punctuation, and a missing glyph draws as a blank box. Strings are rendered to cached canvas textures keyed by text and colour. Names are truncated to fit the bar width.
- **Sparks:** these replace the circle flashes and use `buildScene().flashes` unchanged (`kind`, `target`, `x`, `y`, `age`, `strength`). They are an 8-ray pixel burst, white/yellow/orange for hit and throw and white/cyan for block. They are drawn with `Graphics.fillRect` at 3× pixel size.

### 6. Frame countdown
The countdown shows `max(0, log.frame_budget − 1 − f)`, zero-padded to three digits and wider when the budget exceeds 1000. It is computed by a pure `countdownAt(log, f)`. It uses the log's own `frame_budget`, so `pf sim --frames N` logs count down from N − 1, and a timeout round reaches 000 on its last frame. It counts frames, not seconds, because frames are the unit players script against.

### 7. Round intro lives in `Playback`
`Playback` gains an `intro: { elapsedMs: number } | null` state and an `INTRO_MS` of 1500. The intro has two beats: the round word for 0–800 ms, then `FIGHT!` for 800–1500 ms. A round with no label shows `FIGHT!` for the whole intro.
- `play()` starts the intro when it begins at frame 0. That covers pressing play at frame 0, replaying from the last frame (which already resets to 0), and auto-play.
- `tick()` spends real milliseconds on the intro first, ignoring speed so 0.25× doesn't make it 6 s, and only then advances frames.
- `seek()`, `step()` and `pause()` clear it, and listeners are told so the controls stay in sync.
- The frame stays 0 throughout, so the scrub bar, panels and feed need no changes.
- The stage reads `playback.intro` and draws the intro words. Fighters are drawn from frame 0's positions with the idle pose forced.

The round label is an optional `StageSource.roundLabel: string | null`. `app.ts` sets it to `ROUND n`, or `FINAL ROUND` when n is 3 (rounds are best of 3), and to null for local files.

- **Why in `Playback`:** it already owns the clock and is unit-tested without Phaser. Putting the intro in the stage would need the stage to intercept play and fight the controls.
- **Alternative:** an overlay on frames 0–40. This was rejected because scripts act from frame 0, so it would hide real moves.

### 8. Rendering structure
In `stage.ts`, drawing is split into:
- a static backdrop Image (depth 0)
- a shadow Graphics (1)
- two fighter Images (2)
- sparks and HUD Graphics (3)
- callout Images (4)
- Texts and banner (5)

`update()` keeps today's dirty and redraw-on-frame-change logic (it also redraws while `playback.intro` is active) and the playback clock (`rawDelta`, 250 ms cap). `mountStage` and `StageScene.invalidate` keep their signatures, so `app.ts` and `play.ts` don't change.

## Risks / Trade-offs

- **[Drawn limbs shorter than the real reach could mislead players]** → The show-ranges overlay draws exact reach, and the event feed states hits and whiffs. Documented in `docs/API.md`.
- **[Sprite overlap at minimum distance of 4 units, 35 px]** → Expected in the genre. Draw order puts the fighter that is acting (attack, throw) in front, otherwise p2 behind p1.
- **[Texture memory]** → About 45 poses × 2 palettes × 72×64 is about 1 MB of RGBA in total, made lazily on first use. That is acceptable.
- **[Pose table drifts from engine anims when new anims are added]** → Unknown anims fall back to the standing pose for their `state`. A test walks every golden log frame and asserts every (state, anim, phase) maps to a defined pose with no fallback.
- **[A taller stage pushes the panels down on small laptop screens]** → `Scale.FIT` caps the stage at the existing max width of 1100 px. If it's too tall in practice, a CSS max-height on `.stage` scales it down without changing the game size.
- **[Player names may contain characters outside the pixel font]** → They draw as blank boxes on the stage. The full name stays readable as normal text in the script panel headings.
- **[Visual regressions are hard to unit-test]** → Pure modules get Vitest coverage. Rendering is checked in headless Chrome, with screenshots at chosen frames and an assertion of no console errors, as in earlier phases.

## Migration Plan

This is frontend-only. Rebuild `web/dist`, then commit and embed it. Rollback is to restore the previous `stage.ts` and dist. Logs, API and engine are unaffected.
