// The stage: a Phaser 3 scene that draws buildScene() output for the current
// frame and drives the playback clock. Only Phaser 3 APIs are used.
import * as Phaser from 'phaser';
import type { Playback } from './playback';
import { buildScene, MAX_HP, type FighterDraw, type Scene } from './scene';
import type { Log } from './types';

export const GAME_WIDTH = 960;
export const GAME_HEIGHT = 380;

const STAGE_LEFT = 40;
const STAGE_PX = GAME_WIDTH - 2 * STAGE_LEFT; // pixels for 100 stage units
const UNIT = STAGE_PX / 100;
const GROUND_Y = 330;
const Y_UNIT = 4; // pixels per unit of jump height
const BODY_W = 4 * UNIT; // fighters are 4 units wide
const STAND_H = 120;
const CROUCH_H = 78;
const DOWN_H = 26;

const LIMB_STYLE = {
  startup: { color: 0xf6e7a1, width: 4, extent: 0.45 },
  active: { color: 0xffd84a, width: 10, extent: 1 },
  recovery: { color: 0x8a7f5a, width: 6, extent: 0.7 },
} as const;

const FLASH_COLORS = { hit: 0xffffff, block: 0x6fe3d2, throw: 0xf2b134 } as const;

const TEXT_STYLE: Phaser.Types.GameObjects.Text.TextStyle = { fontFamily: 'system-ui, sans-serif', fontSize: '16px', color: '#e6e9ec' };

function sx(x: number): number {
  return STAGE_LEFT + x * UNIT;
}

function bodyHeight(fd: FighterDraw): number {
  switch (fd.pose) {
    case 'crouch':
    case 'land':
      return CROUCH_H;
    case 'down':
      return DOWN_H;
  }
  return STAND_H;
}

export interface StageSource {
  log: Log;
  playback: Playback;
  showRanges: () => boolean;
}

export class StageScene extends Phaser.Scene {
  private g!: Phaser.GameObjects.Graphics;
  private names: Phaser.GameObjects.Text[] = [];
  private combos: Phaser.GameObjects.Text[] = [];
  private frameText!: Phaser.GameObjects.Text;
  private bannerTitle!: Phaser.GameObjects.Text;
  private bannerDetail!: Phaser.GameObjects.Text;
  private drawn = -1;
  private dirty = true;

  constructor(private readonly src: StageSource) {
    super({ key: 'stage' });
  }

  /** Forces a redraw on the next update (e.g. when an overlay is toggled). */
  invalidate(): void {
    this.dirty = true;
  }

  create(): void {
    this.g = this.add.graphics();
    for (let i = 0; i < 2; i++) {
      this.names.push(this.add.text(0, 0, '', TEXT_STYLE));
      this.combos.push(this.add.text(0, 0, '', { ...TEXT_STYLE, fontSize: '18px', color: '#f2b134', fontStyle: 'bold' }).setOrigin(0.5, 1));
    }
    this.frameText = this.add.text(GAME_WIDTH / 2, 40, '', { ...TEXT_STYLE, color: '#8b96a1' }).setOrigin(0.5, 0);
    this.bannerTitle = this.add.text(GAME_WIDTH / 2, 92, '', { ...TEXT_STYLE, fontSize: '30px', fontStyle: 'bold' }).setOrigin(0.5).setDepth(2);
    this.bannerDetail = this.add.text(GAME_WIDTH / 2, 126, '', { ...TEXT_STYLE, color: '#c9d1d9' }).setOrigin(0.5).setDepth(2);
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
    if (f !== this.drawn || this.dirty) {
      this.draw(buildScene(this.src.log, f, { showRanges: this.src.showRanges() }));
      this.drawn = f;
      this.dirty = false;
    }
  }

  private draw(sc: Scene): void {
    const g = this.g;
    g.clear();
    // Ground and stage edges.
    g.fillStyle(0x1e262e, 1).fillRect(0, GROUND_Y, GAME_WIDTH, GAME_HEIGHT - GROUND_Y);
    g.lineStyle(2, 0x3a4652, 1).lineBetween(STAGE_LEFT, GROUND_Y, STAGE_LEFT + STAGE_PX, GROUND_Y);
    for (let x = 0; x <= 100; x += 10) {
      g.lineStyle(1, 0x2a333c, 1).lineBetween(sx(x), GROUND_Y, sx(x), GROUND_Y + (x % 50 === 0 ? 14 : 7));
    }
    sc.fighters.forEach((fd, i) => this.drawFighter(fd, i, sc.showRanges));
    for (const fl of sc.flashes) {
      const target = fl.target === 'p1' ? sc.fighters[0] : sc.fighters[1];
      const cy = GROUND_Y - fl.y * Y_UNIT - bodyHeight(target) * 0.6;
      g.fillStyle(FLASH_COLORS[fl.kind], 0.85 * fl.strength).fillCircle(sx(fl.x), cy, 14 + fl.age * 9);
      g.lineStyle(3, FLASH_COLORS[fl.kind], fl.strength).strokeCircle(sx(fl.x), cy, 22 + fl.age * 12);
    }
    this.drawHud(sc);
  }

  private drawFighter(fd: FighterDraw, i: number, showRanges: boolean): void {
    const g = this.g;
    const cx = sx(fd.x);
    const foot = GROUND_Y - fd.y * Y_UNIT;
    const h = bodyHeight(fd);
    const w = fd.pose === 'down' ? STAND_H * 0.9 : BODY_W;
    const left = cx - w / 2;
    const top = foot - h;
    g.fillStyle(fd.color, fd.alpha).fillRect(left, top, w, h);
    g.lineStyle(3, fd.outline, fd.alpha).strokeRect(left, top, w, h);
    if (fd.pose !== 'down') {
      // Head with a facing "eye".
      g.fillStyle(fd.color, fd.alpha).fillCircle(cx, top - 12, 12);
      g.lineStyle(3, fd.outline, fd.alpha).strokeCircle(cx, top - 12, 12);
      g.fillStyle(0x101418, fd.alpha).fillCircle(cx + fd.facing * 6, top - 14, 3);
    }
    if (fd.state === 'block' || fd.state === 'blockstun') {
      g.lineStyle(5, 0x6fe3d2, fd.alpha).lineBetween(cx + fd.facing * (w / 2 + 6), top + 6, cx + fd.facing * (w / 2 + 6), foot - 6);
    }
    if (fd.limb) {
      const reach = fd.limb.reach * UNIT;
      const y = fd.limb.height === 'high' ? top + h * 0.25 : fd.limb.height === 'low' ? foot - 10 : top + h * 0.55;
      const startX = cx + fd.facing * (w / 2);
      if (showRanges) {
        g.lineStyle(1, 0xffffff, 0.35).lineBetween(cx, y, cx + fd.facing * reach, y);
      }
      const st = LIMB_STYLE[fd.limb.phase];
      const len = Math.max(reach * st.extent - w / 2, 6);
      g.lineStyle(st.width, st.color, fd.alpha).lineBetween(startX, y, startX + fd.facing * len, y);
    }
    const name = this.names[i]!;
    name.setText(fd.name).setPosition(cx, foot + 8).setOrigin(0.5, 0).setColor(i === 0 ? '#4fa3ff' : '#ff6b6b');
    this.combos[i]!.setText(fd.comboLabel ?? '').setPosition(cx, top - 30);
  }

  private drawHud(sc: Scene): void {
    const g = this.g;
    const barW = 380;
    sc.fighters.forEach((fd, i) => {
      const x = i === 0 ? 20 : GAME_WIDTH - 20 - barW;
      const frac = Math.max(0, Math.min(1, fd.hp / MAX_HP));
      g.fillStyle(0x2a333c, 1).fillRect(x, 16, barW, 18);
      const fillX = i === 0 ? x + barW * (1 - frac) : x; // bars drain towards the centre
      g.fillStyle(frac > 0.3 ? 0x5ad17a : 0xff5c5c, 1).fillRect(fillX, 16, barW * frac, 18);
      g.lineStyle(2, fd.outline, 1).strokeRect(x, 16, barW, 18);
    });
    this.frameText.setText(sc.frameLabel);
    if (sc.banner) {
      g.fillStyle(0x000000, 0.7).fillRect(GAME_WIDTH / 2 - 260, 64, 520, 84);
      this.bannerTitle.setText(sc.banner.title);
      this.bannerDetail.setText(sc.banner.detail);
    } else {
      this.bannerTitle.setText('');
      this.bannerDetail.setText('');
    }
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
    backgroundColor: '#14191f',
    banner: false,
    audio: { noAudio: true },
    input: { keyboard: false, mouse: false, touch: false, gamepad: false },
    scale: { mode: Phaser.Scale.FIT, autoCenter: Phaser.Scale.CENTER_HORIZONTALLY },
    scene,
  });
  document.documentElement.dataset.phaser = Phaser.VERSION;
  return { game, scene };
}
