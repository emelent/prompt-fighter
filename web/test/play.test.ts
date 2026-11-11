import { describe, expect, it } from 'vitest';
import { playState, type StoredSeat } from '../src/playstate';
import { SAMPLES, sample } from '../src/samples';
import { SeatStore, type StorageLike } from '../src/seats';
import type { MatchView } from '../src/types';

const A: StoredSeat = { seat: 'a', token: 'ta', name: 'alice' };
const B: StoredSeat = { seat: 'b', token: 'tb', name: 'bob' };

function view(over: Partial<MatchView>): MatchView {
  return {
    match_id: '0123456789abcdef',
    status: 'awaiting_scripts',
    config_version: '2026.11-2',
    next_round: 2,
    seats: { a: { name: 'alice', wins: 1, submitted: false }, b: { name: 'bob', wins: 0, submitted: false } },
    rounds: [],
    winner: null,
    ...over,
  };
}

const waiting = view({ status: 'waiting_for_opponent', next_round: 1, seats: { a: { name: 'alice', wins: 0, submitted: true }, b: null } });

describe('playState', () => {
  it('not found', () => expect(playState(null, A).kind).toBe('notFound'));
  it('join: no seat, seat b free', () => expect(playState(waiting, null).kind).toBe('join'));
  it('spectator: no seat, match full', () => expect(playState(view({}), null).kind).toBe('spectator'));
  it('waiting for opponent: seat a, nobody joined', () => expect(playState(waiting, A).kind).toBe('waitingOpponent'));
  it('your move: not yet submitted', () => {
    const s = playState(view({}), B);
    expect(s.kind).toBe('yourMove');
    expect(s.kind === 'yourMove' && s.round).toBe(2);
  });
  it('waiting for their script: submitted, opponent not', () => {
    const s = playState(view({ seats: { a: { name: 'alice', wins: 1, submitted: true }, b: { name: 'bob', wins: 0, submitted: false } } }), A);
    expect(s.kind).toBe('waitingTheirs');
    expect(s.kind === 'waitingTheirs' && s.round).toBe(2);
  });
  it('finished, with and without a stored seat', () => {
    const fin = view({ status: 'finished', winner: 'b' });
    expect(playState(fin, A)).toMatchObject({ kind: 'finished', seat: A });
    expect(playState(fin, null)).toMatchObject({ kind: 'finished', seat: null });
  });
  it('a stored seat the match does not know is treated as a visitor', () => {
    expect(playState(waiting, B).kind).toBe('join');
  });
  describe('hosted matches', () => {
    const hosted = view({ status: 'waiting_for_players', next_round: 1, seats: { a: null, b: null } });
    const onlyB = view({ status: 'waiting_for_opponent', next_round: 1, seats: { a: null, b: { name: 'bob', wins: 0, submitted: true } } });
    it('a visitor can join a hosted match with no players', () => expect(playState(hosted, null).kind).toBe('join'));
    it('the first player waits for an opponent', () => expect(playState(waiting, A).kind).toBe('waitingOpponent'));
    it('whichever seat is held waits while the other is empty', () => {
      expect(playState(onlyB, B).kind).toBe('waitingOpponent');
      expect(playState(onlyB, null).kind).toBe('join');
    });
    it('a full hosted match is for spectators only', () => expect(playState(view({}), null).kind).toBe('spectator'));
  });
});

class MapStorage implements StorageLike {
  m = new Map<string, string>();
  getItem(k: string) {
    return this.m.get(k) ?? null;
  }
  setItem(k: string, v: string) {
    this.m.set(k, v);
  }
  removeItem(k: string) {
    this.m.delete(k);
  }
}

class ThrowingStorage implements StorageLike {
  getItem(): string | null {
    throw new Error('blocked');
  }
  setItem(): void {
    throw new Error('blocked');
  }
  removeItem(): void {
    throw new Error('blocked');
  }
}

describe('SeatStore', () => {
  it('round-trips seats and drafts, and forgets seats', () => {
    const backing = new MapStorage();
    const st = new SeatStore(backing);
    expect(st.persistent).toBe(true);
    st.setSeat('m1', A);
    st.setDraft('m1', 'kick');
    const again = new SeatStore(backing); // a reload
    expect(again.seat('m1')).toEqual(A);
    expect(again.draft('m1')).toBe('kick');
    again.forgetSeat('m1');
    expect(again.seat('m1')).toBeNull();
    expect(again.draft('m1')).toBe('kick');
  });
  it('ignores corrupt entries', () => {
    const backing = new MapStorage();
    backing.setItem('pf.seat.m1', '{nope');
    backing.setItem('pf.seat.m2', JSON.stringify({ seat: 'c', token: 'x', name: 'y' }));
    const st = new SeatStore(backing);
    expect(st.seat('m1')).toBeNull();
    expect(st.seat('m2')).toBeNull();
  });
  it('falls back to memory when storage throws or is missing', () => {
    for (const s of [new ThrowingStorage(), null, undefined]) {
      const st = new SeatStore(s);
      expect(st.persistent).toBe(false);
      st.setSeat('m1', B);
      expect(st.seat('m1')).toEqual(B);
    }
  });
});

describe('samples', () => {
  const files = import.meta.glob<string>('../../scripts/*.pf', { eager: true, query: '?raw', import: 'default' });
  it('bundles the six sample bots with their exact sources', () => {
    expect(SAMPLES.map((s) => s.name)).toEqual(['ducker', 'jumper', 'lazy', 'rushdown', 'thrower', 'turtle']);
    for (const s of SAMPLES) expect(s.source).toBe(files[`../../scripts/${s.name}.pf`]);
    expect(sample('turtle')?.source).toContain('turtle.pf');
    expect(sample('ninja')).toBeUndefined();
  });
});
