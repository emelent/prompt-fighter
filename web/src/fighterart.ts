// The pixel-art fighter: an original gi-wearing martial artist, generated in
// code. Poses are angles on a small skeleton; rasterise() turns a pose into a
// 72×64 grid of palette indices with shading and a one-pixel outline. Pure (no
// DOM, no Phaser), so it is unit-tested directly; src/stage.ts turns the grids
// into textures.
import type { FighterFrame } from './types';

/** Wide enough for the lying poses; the hip is at the centre column. */
export const SPRITE_W = 72;
export const SPRITE_H = 64;

/** Palette indices used in sprite grids (0 is transparent). */
export const IDX = { out: 1, gi: 2, giS: 3, giH: 4, skin: 5, skinS: 6, hair: 7, belt: 8, wrap: 9, eye: 10, wrapS: 11 } as const;

export type PaletteName = 'p1' | 'p2';

/** Colours by palette index. p1 and p2 differ only in gi, skin and hair. */
export const PALETTES: Record<PaletteName, readonly string[]> = {
  p1: ['', '#120c18', '#2b9a8d', '#1b645c', '#76d9c8', '#c98a5e', '#94603f', '#18121a', '#22202b', '#ece1c8', '#120c18', '#b9ad95'],
  p2: ['', '#120c18', '#8b4fc4', '#5a2f88', '#c7a0f2', '#e6b38c', '#b07f5e', '#3a2418', '#22202b', '#ece1c8', '#120c18', '#b9ad95'],
};

/** The gi highlight colour, used for each player's name and labels. */
export const PLAYER_TEXT: Record<PaletteName, string> = { p1: '#76d9c8', p2: '#c7a0f2' };

type Pair = readonly [number, number];

/**
 * A pose. Angles are degrees. Limb angles are measured from straight down and
 * rotate towards the facing direction (90 is horizontal forward, 180 is up).
 * torso is the lean from vertical (positive is forward). fa/ba are the front
 * and back arms [upper, fore]; fl/bl are the front and back legs [thigh, shin].
 * air poses are drawn centred instead of standing on the bottom row.
 */
export interface Pose {
  torso: number;
  head: number;
  fa: Pair;
  ba: Pair;
  fl: Pair;
  bl: Pair;
  air?: boolean;
}

const G: Pose = { torso: 8, head: 0, fa: [115, 205], ba: [60, 150], fl: [22, -8], bl: [-28, -12] };
const P = (o: Partial<Pose>): Pose => ({ ...G, ...o });

const D1: Pose = { torso: 22, head: -10, fa: [110, 200], ba: [70, 150], fl: [86, -28], bl: [62, -62] };
const HB: Pose = { torso: -4, head: -6, fa: [126, 186], ba: [110, 176], fl: [22, -8], bl: [-28, -12] };

const POSE_TABLE = {
  G,
  G2: P({ torso: 10, fl: [26, -18], bl: [-24, -24] }),
  W1: P({ fl: [32, 6], bl: [-30, -22] }),
  W2: P({ fl: [10, -12], bl: [-6, -32] }),
  W3: P({ fl: [-18, -30], bl: [26, 2] }),
  W4: P({ fl: [0, -22], bl: [2, -16] }),
  P0: P({ torso: 4, fa: [70, 140] }),
  P1: P({ torso: 16, fa: [92, 92], ba: [15, 100], fl: [32, 2], bl: [-36, -16] }),
  PR: P({ torso: 10, fa: [96, 150] }),
  K0: P({ torso: 0, fl: [102, 12], ba: [40, 120] }),
  K1: { torso: -18, head: 8, fa: [60, 140], ba: [-30, 60], fl: [92, 92], bl: [-8, -8] },
  KR: P({ torso: 2, fl: [80, 0] }),
  S0: { torso: 32, head: -22, fa: [100, 170], ba: [70, 140], fl: [86, -62], bl: [-86, -92] },
  S1: { torso: 36, head: -22, fa: [112, 182], ba: [-60, 40], fl: [86, -64], bl: [90, 91] },
  SR: { torso: 26, head: -12, fa: [110, 190], ba: [60, 140], fl: [84, -60], bl: [30, -70] },
  T0: P({ torso: 18, fa: [86, 86], ba: [80, 96] }),
  T1: { torso: 26, head: 0, fa: [96, 112], ba: [96, 116], fl: [36, 6], bl: [-36, -16] },
  T2: { torso: -16, head: 12, fa: [160, 200], ba: [150, 190], fl: [12, -10], bl: [-40, -26] },
  J0: { torso: 5, head: 0, fa: [130, 200], ba: [80, 160], fl: [86, -30], bl: [62, -52], air: true },
  J1: { torso: 5, head: 0, fa: [110, 190], ba: [60, 150], fl: [30, 0], bl: [-10, -22], air: true },
  JK0: { torso: 0, head: 0, fa: [120, 200], ba: [70, 150], fl: [80, -40], bl: [60, -52], air: true },
  JK1: { torso: -12, head: 6, fa: [100, 170], ba: [30, 110], fl: [56, 56], bl: [72, -40], air: true },
  DP0: { torso: 15, head: 0, fa: [80, 120], ba: [60, 150], fl: [70, -40], bl: [50, -52], air: true },
  DP1: { torso: 30, head: -6, fa: [56, 56], ba: [0, 80], fl: [62, -40], bl: [42, -52], air: true },
  D0: P({ torso: 14, fl: [60, -26], bl: [30, -52] }),
  D1,
  LJ0: { ...D1, fa: [90, 130] },
  LJ1: { ...D1, torso: 25, fa: [95, 95], ba: [40, 110] },
  HB0: P({ fa: [120, 195], ba: [90, 170] }),
  HB,
  BS: { ...HB, torso: -12, head: -10, bl: [-36, -16] },
  LB: { torso: 14, head: 0, fa: [56, 96], ba: [46, 92], fl: [60, -26], bl: [30, -52] },
  LBS: { torso: 6, head: -8, fa: [50, 90], ba: [40, 86], fl: [56, -30], bl: [26, -56] },
  H0: { torso: -28, head: -18, fa: [-30, -70], ba: [-50, -80], fl: [30, 10], bl: [-26, -10] },
  H1: { torso: -20, head: -10, fa: [-10, -40], ba: [-40, -62], fl: [26, 0], bl: [-30, -16] },
  KD0: { torso: -56, head: -20, fa: [-60, -100], ba: [-80, -112], fl: [62, 42], bl: [42, 22], air: true },
  KD1: { torso: -92, head: -6, fa: [-100, -96], ba: [-88, -80], fl: [92, 90], bl: [88, 86] },
  KD2: { torso: 20, head: 0, fa: [62, 30], ba: [40, 20], fl: [86, -22], bl: [2, -90] },
  L: P({ torso: 14, fl: [52, -22], bl: [12, -42] }),
  // Guard broken: dazed, arms dropped, leaning back with the head lolling.
  GB: { torso: -14, head: 18, fa: [20, 10], ba: [-10, -20], fl: [18, -4], bl: [-26, -14] },
  WIN: { torso: 0, head: -8, fa: [172, 176], ba: [32, 62], fl: [12, -6], bl: [-16, -10] },
} satisfies Record<string, Pose>;

export type PoseName = keyof typeof POSE_TABLE;
export const POSES: Record<PoseName, Pose> = POSE_TABLE;
export const POSE_NAMES = Object.keys(POSES) as PoseName[];

/** Default knockdown length (config knockdown_frames); only shapes the get-up. */
const KNOCKDOWN_FRAMES = 20;

export interface PoseChoice {
  pose: PoseName;
  /** True when the anim was not recognised and the state's fallback was used. */
  fallback: boolean;
}

function byPhase(phase: FighterFrame['phase'], startup: PoseName, active: PoseName, recovery: PoseName): PoseName {
  return phase === 'startup' ? startup : phase === 'recovery' ? recovery : active;
}

function stateFallback(state: string): PoseName {
  switch (state) {
    case 'airborne':
      return 'J1';
    case 'duck':
      return 'D1';
    case 'knockdown':
    case 'ko':
      return 'KD1';
    case 'hitstun':
      return 'H1';
    case 'block':
      return 'HB';
    case 'blockstun':
      return 'BS';
    case 'landing':
      return 'L';
    case 'guard_break':
      return 'GB';
  }
  return 'G';
}

/**
 * Picks the sprite pose for one fighter on one frame, from the frame's data
 * alone. logFrame drives the idle breathing; win is true for the winner on
 * the last frame.
 */
export function choosePose(ff: Pick<FighterFrame, 'state' | 'anim' | 'anim_frame' | 'phase'>, logFrame: number, win = false): PoseChoice {
  if (win && ff.state !== 'ko') return { pose: 'WIN', fallback: false };
  const k = Math.max(1, ff.anim_frame);
  const ok = (pose: PoseName): PoseChoice => ({ pose, fallback: false });
  const walk = (cycle: readonly PoseName[]): PoseChoice => ok(cycle[Math.floor((k - 1) / 8) % cycle.length]!);
  switch (ff.anim) {
    case 'idle':
    case 'check':
    case 'wait':
      return ok(Math.floor(logFrame / 16) % 2 === 0 ? 'G' : 'G2');
    case 'walk_forward':
      return walk(['W1', 'W2', 'W3', 'W4']);
    case 'walk_back':
      return walk(['W4', 'W3', 'W2', 'W1']);
    case 'punch':
      return ok(byPhase(ff.phase, 'P0', 'P1', 'PR'));
    case 'kick':
      return ok(byPhase(ff.phase, 'K0', 'K1', 'KR'));
    case 'sweep':
      return ok(byPhase(ff.phase, 'S0', 'S1', 'SR'));
    case 'throw':
      return ok(byPhase(ff.phase, 'T0', 'T1', 'T2'));
    case 'low_jab':
      return ok(byPhase(ff.phase, 'LJ0', 'LJ1', 'D1'));
    case 'jump':
      return ok(k <= 10 ? 'J0' : 'J1');
    case 'jump_kick':
      return ok(ff.phase === 'startup' ? 'JK0' : ff.phase === 'active' ? 'JK1' : 'J1');
    case 'dive_punch':
      return ok(ff.phase === 'startup' ? 'DP0' : ff.phase === 'active' ? 'DP1' : 'J1');
    case 'landing':
      return ok('L');
    case 'duck':
      return ok(byPhase(ff.phase, 'D0', 'D1', 'D0'));
    case 'high_block':
      // A defender in blockstun keeps its block anim.
      if (ff.state === 'blockstun') return ok('BS');
      return ok(byPhase(ff.phase, 'HB0', 'HB', 'HB0'));
    case 'low_block':
      if (ff.state === 'blockstun') return ok('LBS');
      return ok(byPhase(ff.phase, 'D0', 'LB', 'D0'));
    case 'blockstun':
      // The attacker whose attack was blocked.
      return ok('BS');
    case 'hitstun':
      return ok(k <= 4 ? 'H0' : 'H1');
    case 'guard_break':
      return ok('GB');
    case 'knockdown':
      return ok(k <= 5 ? 'KD0' : k <= KNOCKDOWN_FRAMES - 4 ? 'KD1' : 'KD2');
    case 'ko':
      return ok('KD1');
  }
  return { pose: stateFallback(ff.state), fallback: true };
}

// --- rasteriser ---------------------------------------------------------------

type Pt = readonly [number, number];

const rad = (d: number): number => (d * Math.PI) / 180;
const step = (p: Pt, a: number, l: number): Pt => [p[0] + Math.sin(rad(a)) * l, p[1] + Math.cos(rad(a)) * l];
const rot = (dx: number, dy: number, deg: number): Pt => {
  const c = Math.cos(rad(deg));
  const s = Math.sin(rad(deg));
  return [dx * c - dy * s, dx * s + dy * c];
};

function joints(p: Pose) {
  const hip: Pt = [0, 0];
  const torsoA = 180 - p.torso;
  const sh = step(hip, torsoA, 15);
  const head = step(hip, torsoA - p.head, 21);
  const fs: Pt = [sh[0] + 1, sh[1] + 1];
  const bs: Pt = [sh[0] - 1, sh[1] + 1];
  const fe = step(fs, p.fa[0], 8);
  const ff = step(fe, p.fa[1], 8);
  const be = step(bs, p.ba[0], 8);
  const bf = step(be, p.ba[1], 8);
  const fh: Pt = [2, 0];
  const bh: Pt = [-2, 0];
  const fk = step(fh, p.fl[0], 12);
  const ft = step(fk, p.fl[1], 12);
  const bk = step(bh, p.bl[0], 12);
  const bt = step(bk, p.bl[1], 12);
  return { hip, sh, head, fs, fe, ff, bs, be, bf, fh, fk, ft, bh, bk, bt };
}

/** Rasterises a pose into a SPRITE_W×SPRITE_H grid of palette indices, facing right. */
export function rasterise(p: Pose): Uint8Array {
  const j = joints(p);
  // Grounded poses rest their lowest point on the bottom rows; air poses are centred.
  const extents: [Pt, number][] = [
    [j.ft, 3], [j.bt, 3], [j.ff, 2], [j.bf, 2], [j.head, 5], [j.hip, 5], [j.sh, 5], [j.fk, 3], [j.bk, 3],
  ];
  const lowest = Math.max(...extents.map(([q, r]) => q[1] + r));
  const ox = SPRITE_W / 2;
  const oy = p.air ? Math.min(40, 61.5 - lowest) : 61.5 - lowest;
  const g = new Uint8Array(SPRITE_W * SPRITE_H);
  const T = (q: Pt): Pt => [q[0] + ox, q[1] + oy];
  const plot = (x: number, y: number, c: number): void => {
    if (x >= 0 && y >= 0 && x < SPRITE_W && y < SPRITE_H) g[y * SPRITE_W + x] = c;
  };
  const disc = (q: Pt, r: number, c: number): void => {
    const [cx, cy] = T(q);
    for (let y = Math.floor(cy - r - 1); y <= Math.ceil(cy + r + 1); y++)
      for (let x = Math.floor(cx - r - 1); x <= Math.ceil(cx + r + 1); x++)
        if ((x + 0.5 - cx) ** 2 + (y + 0.5 - cy) ** 2 <= r * r) plot(x, y, c);
  };
  const line = (a: Pt, b: Pt, r: number, c: number): void => {
    const n = Math.ceil(Math.hypot(b[0] - a[0], b[1] - a[1]) * 2) + 1;
    for (let i = 0; i <= n; i++) disc([a[0] + ((b[0] - a[0]) * i) / n, a[1] + ((b[1] - a[1]) * i) / n], r, c);
  };
  const foot = (q: Pt): Pt => [q[0] + 3.5, q[1]];

  // Back limbs, in shadow tones.
  line(j.bs, j.be, 1.9, IDX.giS);
  line(j.be, j.bf, 1.4, IDX.skinS);
  disc(j.bf, 1.9, IDX.wrapS);
  line(j.bh, j.bk, 3.1, IDX.giS);
  line(j.bk, j.bt, 2.5, IDX.giS);
  line(j.bt, foot(j.bt), 1.3, IDX.skinS);
  // Torso, a highlight seam and the belt with its tail.
  line(j.hip, j.sh, 4.2, IDX.gi);
  disc(j.sh, 4.6, IDX.gi);
  const hl = rot(3, 0, p.torso);
  line([j.hip[0] + hl[0], j.hip[1] + hl[1] - 2], [j.sh[0] + hl[0], j.sh[1] + hl[1] + 1], 0.55, IDX.giH);
  const bw = rot(4.5, 0, p.torso);
  line([j.hip[0] - bw[0], j.hip[1] - bw[1] - 1], [j.hip[0] + bw[0], j.hip[1] + bw[1] - 1], 1.1, IDX.belt);
  const tail = rot(-3, 4, p.torso);
  line([j.hip[0] - 3, j.hip[1] - 1], [j.hip[0] - 3 + tail[0], j.hip[1] - 1 + tail[1]], 0.6, IDX.belt);
  // Head: face, hair cap, topknot, and an eye on the facing side.
  const hdeg = p.torso + p.head;
  {
    const [cx, cy] = T(j.head);
    for (let y = Math.floor(cy - 6); y <= Math.ceil(cy + 6); y++)
      for (let x = Math.floor(cx - 6); x <= Math.ceil(cx + 6); x++) {
        const dx = x + 0.5 - cx;
        const dy = y + 0.5 - cy;
        if (dx * dx + dy * dy > 26) continue;
        const [lx, ly] = rot(dx, dy, -hdeg);
        plot(x, y, ly < -1.2 || (lx < -2.2 && ly < 3) ? IDX.hair : IDX.skin);
      }
    const knot = rot(-2.5, -6.2, hdeg);
    disc([j.head[0] + knot[0], j.head[1] + knot[1]], 2.1, IDX.hair);
    const e = rot(2.4, -0.4, hdeg);
    plot(Math.round(cx + e[0]), Math.round(cy + e[1]), IDX.eye);
  }
  // Front limbs.
  line(j.fh, j.fk, 3.1, IDX.gi);
  line(j.fk, j.ft, 2.5, IDX.gi);
  line(j.ft, foot(j.ft), 1.3, IDX.skin);
  line(j.fs, j.fe, 1.9, IDX.gi);
  line(j.fe, j.ff, 1.4, IDX.skin);
  disc(j.ff, 1.9, IDX.wrap);
  // Outline: every empty pixel next to a filled one.
  const out = g.slice();
  const filled = (x: number, y: number): boolean => x >= 0 && y >= 0 && x < SPRITE_W && y < SPRITE_H && g[y * SPRITE_W + x] !== 0;
  for (let y = 0; y < SPRITE_H; y++)
    for (let x = 0; x < SPRITE_W; x++) {
      if (g[y * SPRITE_W + x] !== 0) continue;
      if (filled(x - 1, y) || filled(x + 1, y) || filled(x, y - 1) || filled(x, y + 1)) out[y * SPRITE_W + x] = IDX.out;
    }
  return out;
}

/** Parses "#rrggbb" into [r, g, b]. */
export function hexRGB(hex: string): [number, number, number] {
  return [parseInt(hex.slice(1, 3), 16), parseInt(hex.slice(3, 5), 16), parseInt(hex.slice(5, 7), 16)];
}

/** Colours a palette-index grid as RGBA bytes (index 0 is transparent). */
export function colourise(grid: Uint8Array, palette: readonly string[]): Uint8ClampedArray {
  const rgb = palette.map((h) => (h ? hexRGB(h) : null));
  const px = new Uint8ClampedArray(grid.length * 4);
  for (let i = 0; i < grid.length; i++) {
    const c = rgb[grid[i]!];
    if (!c) continue;
    px[i * 4] = c[0];
    px[i * 4 + 1] = c[1];
    px[i * 4 + 2] = c[2];
    px[i * 4 + 3] = 255;
  }
  return px;
}

const gridCache = new Map<PoseName, Uint8Array>();

/** The (cached) grid for a named pose. */
export function poseGrid(name: PoseName): Uint8Array {
  let g = gridCache.get(name);
  if (!g) {
    g = rasterise(POSES[name]);
    gridCache.set(name, g);
  }
  return g;
}
