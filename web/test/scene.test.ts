import { describe, expect, it } from 'vitest';
import { buildScene, FLASH_FRAMES, LEGACY_MAX_HP, maxHP, poseFor } from '../src/scene';
import type { FighterFrame, Log } from '../src/types';
import { goldenLogs } from './fixtures';

const logs = Object.values(goldenLogs);

function findFrame(pred: (ff: FighterFrame) => boolean): { log: Log; f: number; who: 0 | 1 } {
  for (const log of logs) {
    for (const fr of log.frames) {
      if (pred(fr.p1)) return { log, f: fr.f, who: 0 };
      if (pred(fr.p2)) return { log, f: fr.f, who: 1 };
    }
  }
  throw new Error('no matching frame in the golden logs');
}

describe('buildScene', () => {
  it('draws an active jump kick raised with an active limb', () => {
    // The sample bots only land late jump kicks, so set the spec's fields on a cloned frame.
    const log = structuredClone(goldenLogs.jumper_vs_lazy!);
    Object.assign(log.frames[40]!.p1, { state: 'airborne', anim: 'jump_kick', phase: 'active', y: 10 });
    const fd = buildScene(log, 40).fighters[0];
    expect(fd.pose).toBe('air');
    expect(fd.y).toBeGreaterThan(0);
    expect(fd.limb).toEqual({ attack: 'jump_kick', phase: 'active', reach: 10, height: 'air' });
  });

  it('draws knockdown lying and translucent', () => {
    const { log, f, who } = findFrame((ff) => ff.state === 'knockdown' && ff.invulnerable);
    const fd = buildScene(log, f).fighters[who];
    expect(fd.pose).toBe('down');
    expect(fd.alpha).toBeLessThan(1);
  });

  it('shows a combo counter from 2 hits', () => {
    const log = structuredClone(goldenLogs.rushdown_vs_turtle!);
    log.frames[50]!.p2.combo = 3;
    expect(buildScene(log, 50).fighters[1].comboLabel).toBe('3 hits');
    log.frames[50]!.p2.combo = 1;
    expect(buildScene(log, 50).fighters[1].comboLabel).toBeNull();
  });

  it('shows the result banner only on the last frame', () => {
    for (const log of logs) {
      const last = log.frames.length - 1;
      expect(buildScene(log, last - 1).banner).toBeNull();
      const b = buildScene(log, last).banner!;
      expect(b.detail).toContain(`at f${log.result.end_frame}`);
      if (log.result.winner) {
        const name = log.players[log.result.winner === 'p1' ? 0 : 1].name;
        expect(b.title).toBe(`${name} (${log.result.winner}) wins`);
      } else {
        expect(b.title).toBe('Draw');
      }
    }
  });

  it('labels frames and carries script lines', () => {
    const log = goldenLogs.jumper_vs_lazy!;
    const s = buildScene(log, 100);
    expect(s.frameLabel).toBe(`f 100 / ${log.frames.length - 1}`);
    expect(s.fighters[0].scriptLine).toBe(log.frames[100]!.p1.script_line);
    expect(s.fighters[0].name).toBe(log.players[0].name);
  });

  it('flashes hits, blocks and successful throws for 3 frames after the event', () => {
    for (const log of logs) {
      for (const e of log.events.filter((e) => e.type === 'hit' || e.type === 'block' || (e.type === 'throw' && e.detail.outcome === 'success'))) {
        const kind = e.type as 'hit' | 'block' | 'throw';
        for (let age = 0; age <= FLASH_FRAMES; age++) {
          const f = e.frame + age;
          if (f >= log.frames.length) break;
          const fl = buildScene(log, f).flashes.filter((x) => x.kind === kind && x.age === age);
          expect(fl.length, `${e.type} at f${e.frame} age ${age}`).toBeGreaterThan(0);
          expect(fl[0]!.strength).toBeCloseTo(1 - age / (FLASH_FRAMES + 1));
        }
        const after = e.frame + FLASH_FRAMES + 1;
        if (after < log.frames.length) {
          expect(buildScene(log, after).flashes.some((x) => x.age === FLASH_FRAMES + 1)).toBe(false);
        }
      }
    }
  });

  it('maps states to poses', () => {
    expect(['airborne', 'landing', 'knockdown', 'ko', 'duck', 'walk', 'idle', 'hitstun'].map(poseFor)).toEqual(['air', 'land', 'down', 'down', 'crouch', 'walk', 'stand', 'stand']);
  });

  it('rejects frames outside the log', () => {
    expect(() => buildScene(goldenLogs.jumper_vs_lazy!, 100000)).toThrow(RangeError);
  });
});

describe('maxHP', () => {
  it('takes the HP bar length from the log, and 100 for logs without max_hp', () => {
    const l = goldenLogs['rushdown_vs_turtle']!;
    expect(l.max_hp).toBe(200);
    expect(maxHP(l)).toBe(200);
    expect(buildScene(l, 0).maxHp).toBe(200);
    expect(buildScene(l, 0).fighters[0].hp).toBe(200);
    const { max_hp: _, ...old } = l;
    expect(maxHP(old as Log)).toBe(LEGACY_MAX_HP);
    expect(maxHP({ ...l, max_hp: 50 })).toBe(50);
  });
});
