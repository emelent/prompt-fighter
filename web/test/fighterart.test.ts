import { describe, expect, it } from 'vitest';
import { choosePose, colourise, IDX, PALETTES, POSE_NAMES, POSES, poseGrid, SPRITE_H, SPRITE_W } from '../src/fighterart';
import type { FighterFrame } from '../src/types';
import { goldenLogs } from './fixtures';

type Bit = Pick<FighterFrame, 'state' | 'anim' | 'anim_frame' | 'phase'>;
const fr = (state: string, anim: string, phase: FighterFrame['phase'] = '', anim_frame = 1): Bit => ({ state, anim, phase, anim_frame });

describe('choosePose', () => {
  it('shows different poses for startup, active and recovery of every ground attack', () => {
    for (const [anim, state] of [['punch', 'attack'], ['kick', 'attack'], ['sweep', 'attack'], ['throw', 'attack'], ['low_jab', 'duck']] as const) {
      const poses = (['startup', 'active', 'recovery'] as const).map((ph) => choosePose(fr(state, anim, ph), 0).pose);
      expect(new Set(poses).size, anim).toBe(3);
    }
  });
  it('extends the kicking leg only on active frames', () => {
    const forward = (name: ReturnType<typeof choosePose>['pose']) => POSES[name].fl[0];
    const [s, a, r] = (['startup', 'active', 'recovery'] as const).map((ph) => choosePose(fr('attack', 'kick', ph), 0).pose);
    expect(POSES[a!].fl[1]).toBeGreaterThan(60); // shin straight out
    expect(POSES[s!].fl[1]).toBeLessThan(60);
    expect(POSES[r!].fl[1]).toBeLessThan(60);
    expect(forward(a!)).toBeGreaterThan(80);
  });
  it('changes the jump pose by airborne frame', () => {
    expect(choosePose(fr('airborne', 'jump', '', 3), 0).pose).not.toBe(choosePose(fr('airborne', 'jump', '', 15), 0).pose);
  });
  it('cycles walks and breathes while idle', () => {
    const walk = new Set([1, 9, 17, 25].map((k) => choosePose(fr('walk', 'walk_forward', '', k), 0).pose));
    expect(walk.size).toBe(4);
    expect(choosePose(fr('walk', 'walk_forward', '', 33), 0).pose).toBe(choosePose(fr('walk', 'walk_forward', '', 1), 0).pose);
    expect(choosePose(fr('idle', 'idle'), 0).pose).not.toBe(choosePose(fr('idle', 'idle'), 16).pose);
  });
  it('tells defender and attacker blockstun apart from the block height', () => {
    expect(choosePose(fr('blockstun', 'low_block', 'active'), 0).pose).toBe('LBS');
    expect(choosePose(fr('blockstun', 'high_block', 'active'), 0).pose).toBe('BS');
    expect(choosePose(fr('blockstun', 'blockstun'), 0).pose).toBe('BS');
  });
  it('lies down for knockdown and ko, and the winner poses', () => {
    expect(choosePose(fr('knockdown', 'knockdown', '', 10), 0).pose).toBe('KD1');
    expect(choosePose(fr('ko', 'ko'), 0).pose).toBe('KD1');
    expect(choosePose(fr('idle', 'idle'), 0, true).pose).toBe('WIN');
  });
  it('falls back to the state pose for an unknown anim without throwing', () => {
    expect(choosePose(fr('airborne', 'moonwalk'), 0)).toEqual({ pose: 'J1', fallback: true });
    expect(choosePose(fr('mystery', 'moonwalk'), 0)).toEqual({ pose: 'G', fallback: true });
  });
});

describe('rasterise', () => {
  it('draws every pose fully inside the sprite, outlined', () => {
    for (const name of POSE_NAMES) {
      const g = poseGrid(name);
      expect(g.length).toBe(SPRITE_W * SPRITE_H);
      let filled = 0;
      for (let y = 0; y < SPRITE_H; y++)
        for (let x = 0; x < SPRITE_W; x++) {
          const v = g[y * SPRITE_W + x]!;
          if (v === 0) continue;
          filled++;
          // Nothing touches the left, right or top edge, so nothing was clipped.
          expect(x > 0 && x < SPRITE_W - 1 && y > 0, `${name} clipped at ${x},${y}`).toBe(true);
          if (v === IDX.out) continue;
          for (const [dx, dy] of [[1, 0], [-1, 0], [0, 1], [0, -1]] as const) {
            const n = g[(y + dy) * SPRITE_W + x + dx];
            expect(n === undefined || n !== 0, `${name} unoutlined at ${x},${y}`).toBe(true);
          }
        }
      expect(filled, name).toBeGreaterThan(150);
    }
  });
  it('stands grounded poses on the bottom rows and lifts air poses', () => {
    const lowest = (name: (typeof POSE_NAMES)[number]) => {
      const g = poseGrid(name);
      for (let y = SPRITE_H - 1; y >= 0; y--) for (let x = 0; x < SPRITE_W; x++) if (g[y * SPRITE_W + x]) return y;
      return -1;
    };
    for (const name of POSE_NAMES) {
      if (POSES[name].air) expect(lowest(name), name).toBeLessThan(SPRITE_H - 1);
      else expect(lowest(name), name).toBeGreaterThanOrEqual(SPRITE_H - 3);
    }
  });
  it('colours p1 and p2 differently only in gi, skin and hair', () => {
    const swapped = new Set<number>([IDX.gi, IDX.giS, IDX.giH, IDX.skin, IDX.skinS, IDX.hair]);
    PALETTES.p1.forEach((c, i) => expect(c === PALETTES.p2[i], `index ${i}`).toBe(!swapped.has(i)));
    const g = poseGrid('G');
    const a = colourise(g, PALETTES.p1);
    const b = colourise(g, PALETTES.p2);
    for (let i = 0; i < g.length; i++) {
      expect(a[i * 4 + 3]).toBe(b[i * 4 + 3]); // same silhouette
      const same = a[i * 4] === b[i * 4] && a[i * 4 + 1] === b[i * 4 + 1] && a[i * 4 + 2] === b[i * 4 + 2];
      if (g[i] && !swapped.has(g[i]!)) expect(same).toBe(true);
    }
  });
});

describe('golden-log coverage', () => {
  it('maps every (state, anim, phase) in the golden logs to a defined pose', () => {
    const seen = new Set<string>();
    for (const [name, log] of Object.entries(goldenLogs)) {
      for (const frame of log.frames) {
        for (const ff of [frame.p1, frame.p2]) {
          const c = choosePose(ff, frame.f);
          expect(c.fallback, `${name} f${frame.f} ${ff.state}/${ff.anim}`).toBe(false);
          expect(POSES[c.pose]).toBeDefined();
          seen.add(`${ff.state}/${ff.anim}/${ff.phase}`);
        }
      }
    }
    expect(seen.size).toBeGreaterThan(15);
  });
});
