import { describe, expect, it } from 'vitest';
import { MAX_SOUND_STEP, pickTrack, soundsBetween } from '../src/sounds';
import type { Log } from '../src/types';
import { goldenLogs } from './fixtures';

const base = goldenLogs.rushdown_vs_turtle!;
const withEvents = (events: Log['events']): Log => ({ ...structuredClone(base), events });
const keys = (l: Log, from: number, to: number) => soundsBetween(l, from, to).map((c) => `${c.frame}:${c.key}`);

describe('soundsBetween', () => {
  it('maps hits, whiffs, blocks and throws to the right sounds', () => {
    const l = withEvents([
      { frame: 3, type: 'hit', actor: 'p1', target: 'p2', detail: { attack: 'punch', damage: 5, combo: 1, hp: 95 } },
      { frame: 4, type: 'hit', actor: 'p1', target: 'p2', detail: { attack: 'sweep', damage: 8, combo: 1, hp: 87 } },
      { frame: 5, type: 'whiff', actor: 'p2', detail: { attack: 'low_jab', reason: 'out_of_range' } },
      { frame: 6, type: 'whiff', actor: 'p2', detail: { attack: 'jump_kick', reason: 'stale' } },
      { frame: 7, type: 'block', actor: 'p1', target: 'p2', detail: { attack: 'kick' } },
      { frame: 8, type: 'guard_break', actor: 'p1', target: 'p2', detail: { attack: 'punch' } },
      { frame: 9, type: 'throw', actor: 'p1', target: 'p2', detail: { attack: 'throw', outcome: 'success', damage: 12, hp: 75 } },
      { frame: 10, type: 'throw', actor: 'p1', detail: { attack: 'throw', outcome: 'tech' } },
      { frame: 11, type: 'check', actor: 'p1', detail: { sensor: 'distance', value: 6, line: 1, cost: 2 } },
    ]);
    expect(keys(l, 2, 11)).toEqual(['3:punch_hit', '4:kick_hit', '5:punch_miss', '6:kick_miss', '7:block', '8:block', '9:kick_hit']);
  });
  it('plays a trade once, and only frames after from', () => {
    const hit = (actor: 'p1' | 'p2', attack: string) => ({ frame: 20, type: 'hit', actor, target: actor === 'p1' ? ('p2' as const) : ('p1' as const), detail: { attack, damage: 5, combo: 1, hp: 95 } });
    const l = withEvents([hit('p1', 'punch'), hit('p2', 'punch')]);
    expect(keys(l, 19, 20)).toEqual(['20:punch_hit']);
    expect(keys(l, 20, 21)).toEqual([]);
  });
  it('plays nothing when seeking backwards or jumping far ahead', () => {
    const l = withEvents([{ frame: 30, type: 'block', actor: 'p1', target: 'p2', detail: { attack: 'kick' } }]);
    expect(keys(l, 40, 30)).toEqual([]);
    expect(keys(l, 30 - MAX_SOUND_STEP, 30)).toEqual(['30:block']);
    expect(keys(l, 29 - MAX_SOUND_STEP, 30)).toEqual([]);
  });
  it('plays the jump sound when a fighter leaves the ground', () => {
    const l = goldenLogs.jumper_vs_lazy!;
    const takeoff = l.frames.findIndex((fr, i) => i > 0 && fr.p1.state === 'airborne' && l.frames[i - 1]!.p1.state !== 'airborne');
    expect(takeoff).toBeGreaterThan(0);
    expect(keys(l, takeoff - 1, takeoff)).toContain(`${takeoff}:jump`);
    expect(keys(l, takeoff, takeoff + 1)).not.toContain(`${takeoff + 1}:jump`);
  });
});

describe('pickTrack', () => {
  const tracks = ['/a/fight_music_3.mp3', '/a/fight_music_1.mp3', '/a/fight_music_2.mp3'];
  it('picks every track for some random value, in sorted order', () => {
    expect([0, 0.34, 0.67, 0.999].map((r) => pickTrack(tracks, () => r))).toEqual([
      '/a/fight_music_1.mp3', '/a/fight_music_2.mp3', '/a/fight_music_3.mp3', '/a/fight_music_3.mp3',
    ]);
  });
  it('returns null without tracks', () => {
    expect(pickTrack([])).toBeNull();
  });
});
