import { describe, expect, it } from 'vitest';
import {
  Countdown, DEFAULT_SETTINGS, endButton, loadSimpleSettings, matchIdFrom, nextToWatch, parseAfter, saveSimpleSettings, SETTINGS_KEY, waitingState,
  type Store, type Timers,
} from '../src/spectate';
import type { MatchView, RoundView } from '../src/types';

const round = (n: number): RoundView => ({ n, p1: n % 2 ? 'a' : 'b', winner: 'a', reason: 'ko', end_frame: 300, hp: { a: 50, b: 0 }, config_version: '2026.11-3', log: `/matches/m/rounds/${n}` });
const match = (o: Partial<MatchView> = {}): MatchView => ({
  match_id: 'm',
  status: 'awaiting_scripts',
  config_version: '2026.11-3',
  next_round: 2,
  seats: { a: { name: 'alice', wins: 1, submitted: true }, b: { name: 'bob', wins: 0, submitted: false } },
  rounds: [round(1)],
  winner: null,
  ...o,
});

describe('waitingState', () => {
  it('shows the round, score and who we are waiting for', () => {
    const s = waitingState(match());
    expect(s).toMatchObject({ kind: 'waiting', round: 2, score: { a: { name: 'alice', wins: 1 }, b: { name: 'bob', wins: 0 } }, waitingFor: ['bob'] });
    if (s.kind === 'waiting') expect(s.seats.map((x) => `${x.name}:${x.submitted}`)).toEqual(['alice:true', 'bob:false']);
  });
  it('shows the setup view while seat b is empty', () => {
    const s = waitingState(match({ status: 'waiting_for_opponent', next_round: 1, rounds: [], seats: { a: { name: 'alice', wins: 0, submitted: true }, b: null } }));
    expect(s).toMatchObject({ kind: 'setup', setup: { joined: ['alice'] }, score: { b: null } });
  });
  it('reports the result of a finished match, and a missing match', () => {
    const fin = match({ status: 'finished', winner: 'a', rounds: [round(1), round(2)], seats: { a: { name: 'alice', wins: 2, submitted: false }, b: { name: 'bob', wins: 0, submitted: false } } });
    expect(waitingState(fin)).toMatchObject({ kind: 'finished', winner: 'alice', score: { a: { wins: 2 } } });
    expect(waitingState(match({ status: 'finished', winner: null }))).toMatchObject({ kind: 'finished', winner: null });
    expect(waitingState(null)).toEqual({ kind: 'not_found' });
  });
});

describe('nextToWatch', () => {
  it('treats rounds played before a direct open as watched', () => {
    expect(nextToWatch(match(), null, 1)).toBeNull(); // joined mid-match: no countdown
    expect(nextToWatch(match({ rounds: [round(1), round(2)] }), null, 1)).toBe(2); // round 2 arrived since
  });
  it('starts after the round just watched, lowest first', () => {
    const v = match({ rounds: [round(1), round(2), round(3)] });
    expect(nextToWatch(v, 1, 3)).toBe(2);
    expect(nextToWatch(v, 3, 3)).toBeNull();
  });
  it('parses after=N', () => {
    expect(parseAfter('?after=2')).toBe(2);
    expect(parseAfter('?after=x')).toBeNull();
    expect(parseAfter('')).toBeNull();
  });
});

describe('endButton', () => {
  it('offers the next round when it has been played, otherwise Done', () => {
    expect(endButton(match({ rounds: [round(1), round(2)] }), 1)).toEqual({ kind: 'next', round: 2, href: '/watch/matches/m/rounds/2?autoplay=1' });
    expect(endButton(match(), 1)).toEqual({ kind: 'done', href: '/watch/matches/m?after=1' });
  });
  it('is Done after the deciding round', () => {
    expect(endButton(match({ status: 'finished', rounds: [round(1), round(2)] }), 2)).toEqual({ kind: 'done', href: '/watch/matches/m?after=2' });
  });
});

describe('Countdown', () => {
  const fakeTimers = () => {
    let fn: (() => void) | null = null;
    const t: Timers & { tick(): void; active(): boolean } = {
      setInterval: (f) => ((fn = f), 1),
      clearInterval: () => (fn = null),
      tick: () => fn?.(),
      active: () => fn !== null,
    };
    return t;
  };
  it('ticks 5 to 1 and then fires once', () => {
    const t = fakeTimers();
    const ticks: number[] = [];
    let done = 0;
    new Countdown(t).start(5, (n) => ticks.push(n), () => done++);
    for (let i = 0; i < 7; i++) t.tick();
    expect(ticks).toEqual([5, 4, 3, 2, 1]);
    expect(done).toBe(1);
    expect(t.active()).toBe(false);
  });
  it('can be cancelled', () => {
    const t = fakeTimers();
    let done = 0;
    const c = new Countdown(t);
    c.start(5, () => {}, () => done++);
    t.tick();
    c.cancel();
    for (let i = 0; i < 6; i++) t.tick();
    expect(done).toBe(0);
    expect(c.running).toBe(false);
  });
});

describe('simple player settings', () => {
  const memory = (init: Record<string, string> = {}): Store & { data: Record<string, string> } => ({
    data: { ...init },
    getItem(k) { return this.data[k] ?? null; },
    setItem(k, v) { this.data[k] = v; },
  });
  it('defaults to everything hidden, sound on, 0.5x', () => {
    expect(loadSimpleSettings(memory())).toEqual({ scripts: false, feed: false, timeline: false, speed: false, toggles: false, sound: true, playbackSpeed: 0.5 });
  });
  it('round-trips', () => {
    const m = memory();
    saveSimpleSettings({ ...DEFAULT_SETTINGS, scripts: true, playbackSpeed: 1 }, m);
    expect(loadSimpleSettings(m)).toMatchObject({ scripts: true, playbackSpeed: 1, feed: false });
  });
  it('ignores invalid stored values field by field', () => {
    expect(loadSimpleSettings(memory({ [SETTINGS_KEY]: '{not json' }))).toEqual(DEFAULT_SETTINGS);
    expect(loadSimpleSettings(memory({ [SETTINGS_KEY]: '{"scripts":"yes","feed":true,"playbackSpeed":3}' }))).toEqual({ ...DEFAULT_SETTINGS, feed: true });
  });
  it('survives storage that throws', () => {
    const broken: Store = { getItem: () => { throw new Error('blocked'); }, setItem: () => { throw new Error('blocked'); } };
    expect(loadSimpleSettings(broken)).toEqual(DEFAULT_SETTINGS);
    expect(() => saveSimpleSettings(DEFAULT_SETTINGS, broken)).not.toThrow();
  });
});

describe('matchIdFrom', () => {
  it('accepts an id or any match link', () => {
    for (const t of ['0123456789abcdef', '  0123456789ABCDEF ', 'http://host:8080/play/matches/0123456789abcdef', '/watch/matches/0123456789abcdef/rounds/2?autoplay=1', 'http://h/view/matches/0123456789abcdef#f=3']) {
      expect(matchIdFrom(t), t).toBe('0123456789abcdef');
    }
  });
  it('rejects anything else', () => {
    for (const t of ['', 'hello', '0123', 'http://host/play', '/play/matches/0123456789abcdefgh']) expect(matchIdFrom(t), t).toBeNull();
  });
});

describe('setup (players joining)', () => {
  const origin = 'http://192.168.1.20:8080';
  const hosted = match({ status: 'waiting_for_players', next_round: 1, rounds: [], seats: { a: null, b: null } });
  it('shows nobody joined, the game link and the plugin instructions', () => {
    const s = waitingState(hosted, origin);
    expect(s.kind).toBe('setup');
    if (s.kind !== 'setup') return;
    expect(s.setup.joined).toEqual([]);
    expect(s.setup.gameLink).toBe('http://192.168.1.20:8080/play/matches/m');
    expect(s.setup.marketplaceCommand).toBe('/plugin marketplace add emelent/prompt-fighter');
    expect(s.setup.installCommand).toBe('/plugin install pf@prompt-fighter');
    expect(s.setup.scriptCommand).toBe('/pf:script <how you want to fight>');
    expect(s.setup.submitCommand).toBe('/pf:submit');
    // One command per copy field: pasting two at once makes Claude Code read them as one.
    expect(s.setup.marketplaceCommand).not.toContain('\n');
    expect(s.setup.joinCommand).toBe('/pf:join http://192.168.1.20:8080/play/matches/m');
  });
  it('lists who has joined', () => {
    const one = waitingState(match({ status: 'waiting_for_opponent', next_round: 1, rounds: [], seats: { a: { name: 'alice', wins: 0, submitted: true }, b: null } }), origin);
    expect(one.kind === 'setup' && one.setup.joined).toEqual(['alice']);
  });
  it('becomes the normal waiting state once both have joined', () => {
    expect(waitingState(match(), origin).kind).toBe('waiting');
  });
  it('treats round 1 as new for a page that watched the setup', () => {
    // The watch page records 0 rounds when it first sees the setup state.
    expect(nextToWatch(match({ rounds: [round(1)] }), null, 0)).toBe(1);
  });
});
