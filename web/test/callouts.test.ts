import { describe, expect, it } from 'vitest';
import { CALLOUT_FRAMES, calloutsAt, countdownAt, roundLabel } from '../src/callouts';
import type { Log } from '../src/types';
import { goldenLogs } from './fixtures';

const log = (name: string): Log => goldenLogs[name]!;
const texts = (l: Log, f: number) => calloutsAt(l, f).map((c) => `${c.text}@${c.anchor}`);

describe('calloutsAt', () => {
  it('shows BLOCKED above the defender, then lets it expire', () => {
    const l = log('rushdown_vs_turtle'); // first block: frame 50, target p2; next at 124
    expect(texts(l, 49)).toEqual([]);
    expect(texts(l, 55)).toEqual(['BLOCKED@p2']);
    expect(calloutsAt(l, 55)[0]!.age).toBe(5);
    expect(texts(l, 50 + CALLOUT_FRAMES - 1)).toEqual(['BLOCKED@p2']);
    expect(texts(l, 90)).toEqual([]);
  });
  it('shows TRADE between the fighters and THROW above the thrown fighter', () => {
    expect(texts(log('turtle_vs_jumper'), 59)).toEqual(['TRADE@mid']);
    expect(texts(log('lazy_vs_thrower'), 57)).toEqual(['THROW@p1']);
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
  it('counts the frames left in a 600-frame round down to 000', () => {
    const l = log('rushdown_vs_turtle');
    expect(l.frame_budget).toBe(600);
    expect(countdownAt(l, 0)).toBe('599');
    expect(countdownAt(l, 250)).toBe('349');
    expect(countdownAt(l, 599)).toBe('000');
    expect(countdownAt(l, 595)).toBe('004');
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
