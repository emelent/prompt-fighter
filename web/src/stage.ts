// The stage: a Phaser 3 scene that draws the current frame in pixel art and
// drives the playback clock. Only Phaser 3 APIs are used. All art is
// generated in code (src/fighterart.ts, src/backdrop.ts, src/pixelfont.ts)
// and turned into canvas textures here, so nothing is fetched.
import * as Phaser from 'phaser';
import { BACKDROP_GROUND, BACKDROP_H, BACKDROP_W, backdropRGBA } from './backdrop';
import { calloutsAt, countdownAt, type Callout } from './callouts';
import { choosePose, colourise, PALETTES, PLAYER_TEXT, poseGrid, SPRITE_H, SPRITE_W, type PaletteName, type PoseName } from './fighterart';
import { INTRO_FIGHT_MS, type Playback } from './playback';
import { ADVANCE, fitText, textMask, textWidth } from './pixelfont';
import { buildScene, type FighterDraw, type Flash, type Scene } from './scene';
import { pickTrack, SOUND_KEYS, soundsBetween, type SoundKey } from './sounds';
import blockUrl from './audio/block.mp3';
import jumpUrl from './audio/jump.mp3';
import kickHitUrl from './audio/kick_hit.mp3';
import kickMissUrl from './audio/kick_miss.mp3';
import punchHitUrl from './audio/punch_hit.mp3';
import punchMissUrl from './audio/punch_miss.mp3';
import type { Log } from './types';

export const GAME_WIDTH = 960;
export const GAME_HEIGHT = 540;

/** Screen pixels per art pixel. */
const PX = 3;
const STAGE_LEFT = 40;
const UNIT = (GAME_WIDTH - 2 * STAGE_LEFT) / 100; // pixels per stage unit
const GROUND_Y = BACKDROP_GROUND * PX;
const Y_UNIT = 4; // pixels per unit of jump height

const INK = '#120c18';
const BAR_W = 380;
const BAR_Y = 24;
const BAR_H = 27;

const CALLOUT_COLOURS: Record<Callout['kind'], string> = {
  block: '#8fe9ff',
  throw: '#ffcc33',
  tech: '#ffffff',
  trade: '#ff9a3d',
  ko: '#ff4d3d',
  guard_break: '#ff5ca8',
  boost: '#ff6a3d',
  stale: '#b9ad95',
};
const SPARKS: Record<Flash['kind'], readonly string[]> = {
  hit: ['#ffffff', '#ffcc33', '#ff6a3d'],
  throw: ['#ffffff', '#ffcc33', '#ff6a3d'],
  block: ['#e8fbff', '#62d6ff', '#62d6ff'],
};

const SOUND_URLS: Record<SoundKey, string> = {
  punch_hit: punchHitUrl,
  punch_miss: punchMissUrl,
  kick_hit: kickHitUrl,
  kick_miss: kickMissUrl,
  block: blockUrl,
  jump: jumpUrl,
};
/** Every fight_music_* track in src/audio (see convert-audio.sh); one is picked per page. */
const FIGHT_MUSIC = Object.values(import.meta.glob<string>('./audio/fight_music_*.mp3', { eager: true, query: '?url', import: 'default' }));
const EFFECT_VOLUME = 0.8;
const MUSIC_VOLUME = 0.3;

function sx(x: number): number {
  return STAGE_LEFT + x * UNIT;
}

function hexNum(hex: string): number {
  return parseInt(hex.slice(1), 16);
}

export interface StageSource {
  log: Log;
  playback: Playback;
  showRanges: () => boolean;
  /** The intro's round word ("ROUND 2", "FINAL ROUND"), or null for none. */
  roundLabel?: string | null;
  /** Whether sound effects and music are on (default on). */
  soundOn?: () => boolean;
  /** Whether to keep the round's length hidden (an unwatched round). */
  hideEnd?: () => boolean;
}

type Align = 'left' | 'center' | 'right';

export class StageScene extends Phaser.Scene {
  private shadows!: Phaser.GameObjects.Graphics;
  private fx!: Phaser.GameObjects.Graphics;
  private hud!: Phaser.GameObjects.Graphics;
  private fighters: Phaser.GameObjects.Image[] = [];
  private glyphs: Phaser.GameObjects.Image[] = [];
  private glyphsUsed = 0;
  private drawn = -1;
  private dirty = true;
  private soundFrame = 0; // the frame sound effects were last played up to
  private music: Phaser.Sound.BaseSound | null = null;
  private musicStarted = false;

  constructor(private readonly src: StageSource) {
    super({ key: 'stage' });
  }

  /** Forces a redraw on the next update (e.g. when an overlay is toggled). */
  invalidate(): void {
    this.dirty = true;
  }

  preload(): void {
    for (const key of SOUND_KEYS) this.load.audio(key, SOUND_URLS[key]);
  }

  create(): void {
    // One fight track, picked at random; it is large, so it loads after the
    // stage is up, and only the chosen track is downloaded.
    const track = pickTrack(FIGHT_MUSIC);
    if (track) {
      this.load.audio('music', track);
      this.load.once(Phaser.Loader.Events.COMPLETE, () => {
        this.music = this.sound.add('music', { loop: true, volume: MUSIC_VOLUME });
      });
      this.load.start();
    }
    this.soundFrame = this.src.playback.frame;
    const bd = backdropRGBA();
    this.canvasTexture('backdrop', bd.width, bd.height, bd.data);
    this.add.image(0, 0, 'backdrop').setOrigin(0, 0).setScale(GAME_WIDTH / BACKDROP_W, GAME_HEIGHT / BACKDROP_H).setDepth(0);
    this.shadows = this.add.graphics().setDepth(1);
    for (let i = 0; i < 2; i++) this.fighters.push(this.add.image(0, 0, this.poseTexture('p1', 'G')).setOrigin(0.5, 1).setScale(PX));
    this.fx = this.add.graphics().setDepth(4);
    this.hud = this.add.graphics().setDepth(5);
    this.src.playback.onChange(() => {
      this.dirty = true;
    });
  }

  override update(): void {
    // Use the loop's raw (unsmoothed) elapsed time so playback follows the
    // wall clock: 1x is 60 frames per second. Cap it so a background tab
    // resuming does not jump far ahead.
    this.src.playback.tick(Math.min(this.game.loop.rawDelta, 250));
    const f = this.src.playback.frame;
    this.updateSound(f);
    if (f !== this.drawn || this.dirty) {
      this.draw(buildScene(this.src.log, f, { showRanges: this.src.showRanges(), hideEnd: this.src.hideEnd?.() ?? false }));
      this.drawn = f;
      this.dirty = false;
    }
  }

  // --- sound -------------------------------------------------------------------

  private updateSound(f: number): void {
    const on = this.src.soundOn?.() ?? true;
    this.sound.mute = !on;
    if (f !== this.soundFrame) {
      if (on) for (const cue of soundsBetween(this.src.log, this.soundFrame, f)) this.sound.play(cue.key, { volume: EFFECT_VOLUME });
      this.soundFrame = f;
    }
    // The music starts the first time the round plays and then loops in the
    // background, through pauses and after the round ends; the sound toggle
    // mutes it.
    const m = this.music;
    if (m && !m.isPlaying && (this.src.playback.playing || this.musicStarted)) {
      m.play();
      this.musicStarted = true;
    }
  }

  // --- textures ----------------------------------------------------------------

  private canvasTexture(key: string, w: number, h: number, rgba: Uint8ClampedArray): string {
    if (this.textures.exists(key)) return key;
    const tex = this.textures.createCanvas(key, w, h);
    if (!tex) throw new Error(`cannot create texture ${key}`);
    tex.context.putImageData(new ImageData(new Uint8ClampedArray(rgba), w, h), 0, 0);
    tex.refresh();
    return key;
  }

  private poseTexture(pal: PaletteName, pose: PoseName): string {
    const key = `pose:${pal}:${pose}`;
    if (this.textures.exists(key)) return key;
    return this.canvasTexture(key, SPRITE_W, SPRITE_H, colourise(poseGrid(pose), PALETTES[pal]));
  }

  private glyphTexture(ch: string, colour: string): string {
    const key = `glyph:${colour}:${ch.codePointAt(0)}`;
    if (this.textures.exists(key)) return key;
    const m = textMask(ch);
    const ink = hexNum(colour);
    const out = hexNum(INK);
    const px = new Uint8ClampedArray(m.width * m.height * 4);
    m.bits.forEach((b, i) => {
      if (!b) return;
      const c = b === 1 ? ink : out;
      px[i * 4] = c >> 16;
      px[i * 4 + 1] = (c >> 8) & 0xff;
      px[i * 4 + 2] = c & 0xff;
      px[i * 4 + 3] = 255;
    });
    return this.canvasTexture(key, m.width, m.height, px);
  }

  /** Draws text in the pixel font; y is the top of the text. Returns its width. */
  private text(str: string, x: number, y: number, scale: number, colour: string, align: Align = 'left', alpha = 1): number {
    const chars = [...str];
    const w = (textWidth(str) + 2) * scale;
    const left = align === 'left' ? x : align === 'right' ? x - w : x - w / 2;
    chars.forEach((ch, i) => {
      if (ch === ' ') return;
      let img = this.glyphs[this.glyphsUsed];
      if (!img) {
        img = this.add.image(0, 0, this.glyphTexture(ch, colour)).setOrigin(0, 0).setDepth(6);
        this.glyphs.push(img);
      }
      this.glyphsUsed++;
      img.setTexture(this.glyphTexture(ch, colour)).setPosition(Math.round(left + i * ADVANCE * scale), Math.round(y)).setScale(scale).setAlpha(alpha).setVisible(true);
    });
    return w;
  }

  // --- drawing -------------------------------------------------------------------

  private draw(sc: Scene): void {
    const pb = this.src.playback;
    const log = this.src.log;
    const frame = log.frames[sc.frame]!;
    this.glyphsUsed = 0;
    this.shadows.clear();
    this.fx.clear();
    this.hud.clear();

    const intro = pb.intro;
    const winner = sc.banner ? log.result.winner : null;
    sc.fighters.forEach((fd, i) => {
      const ff = fd.id === 'p1' ? frame.p1 : frame.p2;
      const pose = intro ? choosePose({ state: 'idle', anim: 'idle', anim_frame: 1, phase: '' }, sc.frame).pose : choosePose(ff, sc.frame, winner === fd.id).pose;
      this.drawFighter(fd, i, pose, sc.showRanges && !intro);
    });
    // The fighter that is acting draws in front; otherwise p1 does.
    const acting = (fd: FighterDraw) => fd.state === 'attack' || fd.anim === 'throw';
    const p2Front = acting(sc.fighters[1]) && !acting(sc.fighters[0]);
    this.fighters[0]!.setDepth(p2Front ? 2 : 3);
    this.fighters[1]!.setDepth(p2Front ? 3 : 2);

    for (const fl of sc.flashes) this.drawSpark(fl, sc);
    this.drawHud(sc);
    if (intro) {
      this.drawIntro(intro.elapsedMs);
    } else {
      this.drawCallouts(sc);
    }
    for (let i = this.glyphsUsed; i < this.glyphs.length; i++) this.glyphs[i]!.setVisible(false);
  }

  private feet(fd: FighterDraw): number {
    return GROUND_Y - fd.y * Y_UNIT;
  }

  private drawFighter(fd: FighterDraw, i: number, pose: PoseName, showRanges: boolean): void {
    const cx = sx(fd.x);
    const foot = this.feet(fd);
    // Ground shadow, smaller the higher the fighter is.
    const sw = Math.max(18, 54 - fd.y * 2);
    this.shadows.fillStyle(0x000000, 0.35).fillRect(Math.round(cx - sw), GROUND_Y - 3, sw * 2, 6);
    this.fighters[i]!.setTexture(this.poseTexture(fd.id, pose))
      .setPosition(Math.round(cx), Math.round(foot) + PX) // the bottom art row is the outline under the feet
      .setFlipX(fd.facing < 0)
      .setAlpha(fd.alpha);
    if (showRanges && fd.limb) {
      const reach = fd.limb.reach * UNIT;
      const y = fd.limb.height === 'high' ? foot - 135 : fd.limb.height === 'low' ? foot - 12 : foot - 90;
      const x0 = Math.min(cx, cx + fd.facing * reach);
      this.fx.fillStyle(0xffcc33, fd.limb.phase === 'active' ? 0.8 : 0.35).fillRect(x0, y - 2, reach, 4);
    }
    if (fd.comboLabel) this.text(fd.comboLabel.toUpperCase(), cx, foot - 206, 2, '#ffcc33', 'center');
  }

  private drawSpark(fl: Flash, sc: Scene): void {
    const target = fl.target === 'p1' ? sc.fighters[0] : sc.fighters[1];
    const cx = Math.round(sx(fl.x) + target.facing * 18);
    const cy = Math.round(GROUND_Y - fl.y * Y_UNIT - 120);
    const cols = SPARKS[fl.kind];
    const r = 3 + fl.age * 3; // art pixels
    this.fx.fillStyle(hexNum(cols[0]!), 1).fillRect(cx - PX, cy - PX, PX * 3, PX * 3);
    for (let i = 0; i < 8; i++) {
      const a = (i * Math.PI) / 4 + (fl.kind === 'block' ? Math.PI / 8 : 0);
      for (let d = r - 3; d <= r; d++) {
        const c = cols[Math.min(cols.length - 1, Math.floor((d - r + 3) / 2) + (fl.age > 1 ? 1 : 0))]!;
        this.fx.fillStyle(hexNum(c), Math.max(0.4, fl.strength)).fillRect(Math.round(cx + Math.cos(a) * d * PX), Math.round(cy + Math.sin(a) * d * PX), PX * 2, PX * 2);
      }
    }
  }

  private drawHud(sc: Scene): void {
    const g = this.hud;
    sc.fighters.forEach((fd, i) => {
      const x = i === 0 ? 20 : GAME_WIDTH - 20 - BAR_W;
      const frac = Math.max(0, Math.min(1, fd.hp / sc.maxHp));
      const low = frac <= 0.3;
      g.fillStyle(hexNum(INK), 1).fillRect(x - 3, BAR_Y - 3, BAR_W + 6, BAR_H + 6);
      g.fillStyle(low ? 0x3a2030 : 0xb03a2e, 1).fillRect(x, BAR_Y, BAR_W, BAR_H);
      const w = Math.round(BAR_W * frac);
      const fillX = i === 0 ? x + BAR_W - w : x; // bars drain towards the centre
      g.fillStyle(low ? 0xff5c5c : 0xffcc33, 1).fillRect(fillX, BAR_Y, w, BAR_H);
      g.fillStyle(low ? 0xffa0a0 : 0xfff2b0, 1).fillRect(fillX, BAR_Y, w, 6);
      const name = fitText(fd.name, 28);
      this.text(name, i === 0 ? 17 : GAME_WIDTH - 17, BAR_Y + BAR_H + 6, 2, PLAYER_TEXT[fd.id], i === 0 ? 'left' : 'right');
    });
    this.text(countdownAt(this.src.log, sc.frame), GAME_WIDTH / 2, BAR_Y - 6, 4, '#ffffff', 'center');
    this.text(sc.frameLabel.toUpperCase(), GAME_WIDTH / 2, BAR_Y + 40, 2, '#b9ad95', 'center');
    if (sc.banner) {
      const titleScale = textWidth(sc.banner.title) * 3 <= 600 ? 3 : 2;
      const w = Math.max(textWidth(sc.banner.title) * titleScale, textWidth(sc.banner.detail) * 2) + 48;
      g.fillStyle(0x000000, 0.75).fillRect(GAME_WIDTH / 2 - w / 2, 220, w, 80);
      g.lineStyle(3, hexNum(INK), 1).strokeRect(GAME_WIDTH / 2 - w / 2, 220, w, 80);
      this.text(sc.banner.title.toUpperCase(), GAME_WIDTH / 2, 234, titleScale, '#ffffff', 'center');
      this.text(sc.banner.detail.toUpperCase(), GAME_WIDTH / 2, 272, 2, '#b9ad95', 'center');
    }
  }

  private drawCallouts(sc: Scene): void {
    const stacked = new Map<string, number>(); // callouts already drawn per anchor
    // Newest first, so the newest callout sits closest to the fighter.
    for (const c of calloutsAt(this.src.log, sc.frame).reverse()) {
      const colour = CALLOUT_COLOURS[c.kind];
      if (c.anchor === 'center') {
        this.text(c.text, GAME_WIDTH / 2, 130, 6, colour, 'center', c.alpha);
        continue;
      }
      const [a, b] = sc.fighters;
      const x = c.anchor === 'mid' ? (sx(a.x) + sx(b.x)) / 2 : sx((c.anchor === 'p1' ? a : b).x);
      const foot = c.anchor === 'mid' ? Math.min(this.feet(a), this.feet(b)) : this.feet(c.anchor === 'p1' ? a : b);
      const n = stacked.get(c.anchor) ?? 0;
      stacked.set(c.anchor, n + 1);
      const y = Math.max(96, foot - 240 - n * 33 - Math.floor(c.age / 3) * PX);
      this.text(c.text, x, y, 3, colour, 'center', c.alpha);
    }
  }

  private drawIntro(elapsedMs: number): void {
    const label = this.src.roundLabel ?? null;
    const word = label && elapsedMs < INTRO_FIGHT_MS ? label : 'FIGHT!';
    const scale = word === 'FIGHT!' ? 7 : 6;
    this.text(word, GAME_WIDTH / 2, 160, scale, '#ffcc33', 'center');
  }
}

/** Creates the Phaser game inside parent and returns the stage scene. */
export function mountStage(parent: HTMLElement, src: StageSource): { game: Phaser.Game; scene: StageScene } {
  const scene = new StageScene(src);
  const game = new Phaser.Game({
    type: Phaser.AUTO,
    parent,
    width: GAME_WIDTH,
    height: GAME_HEIGHT,
    backgroundColor: INK,
    pixelArt: true,
    banner: false,
    input: { keyboard: false, mouse: false, touch: false, gamepad: false },
    scale: { mode: Phaser.Scale.FIT, autoCenter: Phaser.Scale.CENTER_BOTH },
    scene,
  });
  document.documentElement.dataset.phaser = Phaser.VERSION;
  return { game, scene };
}
