// buildScene turns (log, frame) into a plain description of what to draw.
// It is pure and imports nothing from Phaser, so it is unit-tested directly;
// src/stage.ts renders its output.
import type { FighterFrame, Log, PlayerID } from './types';
import { lastFrame } from './replay';

export const STAGE_WIDTH = 100;
/** Starting HP for logs from before `max_hp` was recorded (always 100 then). */
export const LEGACY_MAX_HP = 100;

/** Each fighter's starting HP: the full length of an HP bar. */
export function maxHP(log: Log): number {
  return log.max_hp && log.max_hp > 0 ? log.max_hp : LEGACY_MAX_HP;
}
/** Frames a hit/block/throw flash stays visible after its event frame. */
export const FLASH_FRAMES = 3;

export type Pose = 'stand' | 'walk' | 'crouch' | 'air' | 'land' | 'down';
export type Phase = 'startup' | 'active' | 'recovery';
export type LimbHeight = 'high' | 'low' | 'air';

/** Attack reach (stage units) and limb height, mirroring the default config. */
export const ATTACKS: Record<string, { reach: number; height: LimbHeight }> = {
  punch: { reach: 8, height: 'high' },
  kick: { reach: 12, height: 'high' },
  sweep: { reach: 12, height: 'low' },
  throw: { reach: 5, height: 'high' },
  jump_kick: { reach: 10, height: 'air' },
  dive_punch: { reach: 8, height: 'air' },
  low_jab: { reach: 7, height: 'low' },
};

/** Body colour per state. */
export const STATE_COLORS: Record<string, number> = {
  idle: 0x9aa7b4,
  walk: 0xb5c4d1,
  attack: 0xf2a33a,
  block: 0x3fb8a6,
  blockstun: 0x6fe3d2,
  duck: 0x8f9bd6,
  airborne: 0xc6b3ff,
  landing: 0xa99ad8,
  hitstun: 0xff5c5c,
  knockdown: 0xa33a3a,
  ko: 0x4a2a2a,
};

export const PLAYER_COLORS: Record<PlayerID, number> = { p1: 0x4fa3ff, p2: 0xff6b6b };

export interface Limb {
  attack: string;
  phase: Phase;
  reach: number;
  height: LimbHeight;
}

export interface FighterDraw {
  id: PlayerID;
  name: string;
  x: number;
  y: number;
  facing: 1 | -1;
  pose: Pose;
  state: string;
  anim: string;
  color: number;
  outline: number;
  alpha: number;
  limb: Limb | null;
  hp: number;
  combo: number;
  comboLabel: string | null;
  scriptLine: number;
}

export interface Flash {
  kind: 'hit' | 'block' | 'throw';
  /** The fighter that was hit, blocked or thrown. */
  target: PlayerID;
  x: number;
  y: number;
  /** Frames since the event (0 on the event frame). */
  age: number;
  /** 1 on the event frame, fading towards 0. */
  strength: number;
}

export interface Banner {
  title: string;
  detail: string;
}

export interface Scene {
  frame: number;
  last: number;
  frameLabel: string;
  fighters: [FighterDraw, FighterDraw];
  flashes: Flash[];
  banner: Banner | null;
  showRanges: boolean;
  /** HP a full bar stands for. */
  maxHp: number;
}

export interface SceneOptions {
  showRanges?: boolean;
}

export function poseFor(state: string): Pose {
  switch (state) {
    case 'airborne':
      return 'air';
    case 'landing':
      return 'land';
    case 'knockdown':
    case 'ko':
      return 'down';
    case 'duck':
      return 'crouch';
    case 'walk':
      return 'walk';
  }
  return 'stand';
}

function limbFor(ff: FighterFrame): Limb | null {
  const atk = ATTACKS[ff.anim];
  if (!atk || ff.phase === '') {
    return null;
  }
  return { attack: ff.anim, phase: ff.phase, reach: atk.reach, height: atk.height };
}

function fighter(log: Log, id: PlayerID, ff: FighterFrame): FighterDraw {
  const player = id === 'p1' ? log.players[0] : log.players[1];
  return {
    id,
    name: player.name,
    x: ff.x,
    y: ff.y,
    facing: ff.facing,
    pose: poseFor(ff.state),
    state: ff.state,
    anim: ff.anim,
    color: STATE_COLORS[ff.state] ?? STATE_COLORS.idle!,
    outline: PLAYER_COLORS[id],
    alpha: ff.invulnerable ? 0.45 : 1,
    limb: limbFor(ff),
    hp: ff.hp,
    combo: ff.combo,
    comboLabel: ff.combo >= 2 ? `${ff.combo} hits` : null,
    scriptLine: ff.script_line,
  };
}

function flashesAt(log: Log, f: number): Flash[] {
  const out: Flash[] = [];
  for (const e of log.events) {
    if (e.frame > f) {
      break; // events are ordered by frame
    }
    const age = f - e.frame;
    if (age > FLASH_FRAMES) {
      continue;
    }
    let kind: Flash['kind'] | null = null;
    if (e.type === 'hit') kind = 'hit';
    else if (e.type === 'block') kind = 'block';
    else if (e.type === 'throw' && e.detail.outcome === 'success') kind = 'throw';
    if (!kind || !e.target) {
      continue;
    }
    const frame = log.frames[e.frame];
    if (!frame) {
      continue;
    }
    const at = e.target === 'p1' ? frame.p1 : frame.p2;
    out.push({ kind, target: e.target, x: at.x, y: at.y, age, strength: 1 - age / (FLASH_FRAMES + 1) });
  }
  return out;
}

function banner(log: Log): Banner {
  const r = log.result;
  const how = r.reason === 'ko' ? 'KO' : r.reason === 'double_ko' ? 'double KO' : r.reason;
  const detail = `by ${how} at f${r.end_frame} (hp ${r.hp[0]}-${r.hp[1]})`;
  if (r.winner === null) {
    return { title: 'Draw', detail };
  }
  const p = r.winner === 'p1' ? log.players[0] : log.players[1];
  return { title: `${p.name} (${r.winner}) wins`, detail };
}

/** Describes frame f (clamped by the caller) of log. */
export function buildScene(log: Log, f: number, opts: SceneOptions = {}): Scene {
  const last = lastFrame(log);
  const frame = log.frames[f];
  if (!frame) {
    throw new RangeError(`frame ${f} is outside 0..${last}`);
  }
  return {
    frame: f,
    last,
    frameLabel: `f ${f} / ${last}`,
    fighters: [fighter(log, 'p1', frame.p1), fighter(log, 'p2', frame.p2)],
    flashes: flashesAt(log, f),
    banner: f === last ? banner(log) : null,
    showRanges: opts.showRanges ?? false,
    maxHp: maxHP(log),
  };
}
