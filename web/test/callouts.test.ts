import { describe, expect, it } from 'vitest';
import { CALLOUT_FRAMES, calloutsAt, countdownAt, roundLabel } from '../src/callouts';
import type { Log } from '../src/types';
import { goldenLogs } from './fixtures';

const log = (name: string): Log => goldenLogs[name]!;
/** A golden log's frames with the given events instead of its own. */
const withEvents = (events: Log['events']): Log => ({ ...structuredClone(log('rushdown_vs_turtle')), events });
const texts = (l: Log, f: number) => calloutsAt(l, f).map((c) => `${c.text}@${c.anchor}`);

describe('calloutsAt', () => {
  it('shows BLOCKED above the defender, then lets it expire', () => {
    const l = withEvents([{ frame: 50, type: 'block', actor: 'p1', target: 'p2', detail: { attack: 'kick' } }]);
    expect(texts(l, 49)).toEqual([]);
    expect(texts(l, 55)).toEqual(['BLOCKED@p2']);
    expect(calloutsAt(l, 55)[0]!.age).toBe(5);
    expect(texts(l, 50 + CALLOUT_FRAMES - 1)).toEqual(['BLOCKED@p2']);
    expect(texts(l, 90)).toEqual([]);
  });
  it('shows TRADE between the fighters and THROW above the thrown fighter', () => {
    const l = withEvents([
      { frame: 49, type: 'trade', actor: 'p1', target: 'p2', detail: { attacks: ['kick', 'punch'], damages: [9, 5] } },
      { frame: 100, type: 'throw', actor: 'p2', target: 'p1', detail: { attack: 'throw', outcome: 'success', damage: 12, hp: 88 } },
    ]);
    expect(texts(l, 59)).toEqual(['TRADE@mid']);
    expect(texts(l, 100)).toEqual(['THROW@p1']);
    expect(texts(log('lazy_vs_thrower'), 57)).toEqual(['THROW@p1']);
  });
  it('shows GUARD BREAK!, boost multipliers and STALE', () => {
    const l = withEvents([
      { frame: 10, type: 'guard_break', actor: 'p1', target: 'p2', detail: { attack: 'punch' } },
      { frame: 20, type: 'hit', actor: 'p2', target: 'p1', detail: { attack: 'kick', damage: 11, combo: 1, hp: 89, boost: 130 } },
      { frame: 60, type: 'stale', actor: 'p2', detail: { attack: 'kick', repeats: 6, range: 10 } },
      { frame: 100, type: 'hit', actor: 'p1', target: 'p2', detail: { attack: 'kick', damage: 9, combo: 1, hp: 80 } },
    ]);
    expect(texts(l, 12)).toEqual(['GUARD BREAK!@p2']);
    expect(texts(l, 20)).toEqual(['GUARD BREAK!@p2', '×1.3@p1']);
    expect(texts(l, 63)).toEqual(['STALE@p2']);
    expect(texts(l, 100)).toEqual([]); // an unboosted hit has no callout
    // The real boosted kick in ducker_vs_lazy.
    expect(texts(log('ducker_vs_lazy'), 232)).toContain('×1.4@p1');
  });
  it('fades callouts out over their lifetime, except K.O.', () => {
    const l = withEvents([{ frame: 10, type: 'block', actor: 'p1', target: 'p2', detail: { attack: 'kick' } }]);
    expect(calloutsAt(l, 10)[0]!.alpha).toBe(1);
    expect(calloutsAt(l, 30)[0]!.alpha).toBeLessThan(0.5);
    const ko = log('jumper_vs_lazy');
    expect(calloutsAt(ko, ko.frames.length - 1).find((c) => c.kind === 'ko')!.alpha).toBe(1);
  });
  it('shows a single TECH for the two tech events', () => {
    const l = structuredClone(log('lazy_vs_thrower'));
    l.events = [
      { frame: 20, type: 'throw', actor: 'p1', detail: { attack: 'throw', outcome: 'tech' } },
      { frame: 20, type: 'throw', actor: 'p2', detail: { attack: 'throw', outcome: 'tech' } },
    ];
    expect(texts(l, 22)).toEqual(['TECH@mid']);
  });
  it('shows K.O. on the last frame of KO rounds only', () => {
    for (const [name, l] of Object.entries(goldenLogs)) {
      const last = l.frames.length - 1;
      const ko = calloutsAt(l, last).filter((c) => c.kind === 'ko');
      expect(ko.length, name).toBe(l.result.reason === 'timeout' ? 0 : 1);
      if (ko.length) expect(calloutsAt(l, last - 1).some((c) => c.kind === 'ko'), name).toBe(false);
    }
  });
  it('depends only on the frame, not on how it was reached', () => {
    const l = log('rushdown_vs_turtle');
    const direct = [55, 130, 300, 10].map((f) => texts(l, f));
    for (let f = 0; f <= 300; f++) texts(l, f); // play through
    expect([55, 130, 300, 10].map((f) => texts(l, f))).toEqual(direct);
  });
});

describe('countdownAt', () => {
  it('counts the frames left in a 1200-frame round down to 0000', () => {
    const l = log('rushdown_vs_turtle');
    expect(l.frame_budget).toBe(1200);
    expect(countdownAt(l, 0)).toBe('1199');
    expect(countdownAt(l, 250)).toBe('0949');
    expect(countdownAt(l, 1199)).toBe('0000');
    expect(countdownAt(l, 1195)).toBe('0004');
  });
  it('follows the log frame budget', () => {
    const l = { ...log('jumper_vs_lazy'), frame_budget: 300 };
    expect(countdownAt(l, 0)).toBe('299');
    expect(countdownAt({ ...l, frame_budget: 2000 }, 0)).toBe('1999');
    expect(countdownAt({ ...l, frame_budget: 2000 }, 1990)).toBe('0009');
  });
});

describe('roundLabel', () => {
  it('names stored rounds, calls round 3 the final round, and has none for local files', () => {
    expect(roundLabel(1)).toBe('ROUND 1');
    expect(roundLabel(2)).toBe('ROUND 2');
    expect(roundLabel(3)).toBe('FINAL ROUND');
    expect(roundLabel(null)).toBeNull();
    expect(roundLabel(0)).toBeNull();
  });
});
