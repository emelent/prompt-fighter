import { describe, expect, it } from 'vitest';
import { matchRows, newRoundToOpen } from '../src/match';
import type { MatchView, RoundView } from '../src/types';

function round(n: number, p1: 'a' | 'b', winner: 'a' | 'b' | null): RoundView {
  return { n, p1, winner, reason: winner ? 'ko' : 'timeout', end_frame: 300, hp: { a: 50, b: 0 }, config_version: '2026.11-2', log: `/matches/m/rounds/${n}` };
}

const base: MatchView = {
  match_id: '0123456789abcdef',
  status: 'awaiting_scripts',
  config_version: '2026.11-2',
  next_round: 2,
  seats: { a: { name: 'alice', wins: 1, submitted: true }, b: { name: 'bob', wins: 0, submitted: false } },
  rounds: [round(1, 'a', 'a')],
  winner: null,
};

describe('matchRows', () => {
  it('describes a waiting match', () => {
    const r = matchRows({ ...base, status: 'waiting_for_opponent', next_round: 1, seats: { a: { name: 'alice', wins: 0, submitted: true }, b: null }, rounds: [] });
    expect(r.status).toBe('waiting for an opponent to join (next: round 1)');
    expect(r.seats[1]!.text).toBe('seat b: not joined yet');
    expect(r.rounds).toEqual([]);
    expect(r.finished).toBe(false);
  });
  it('describes a match in progress with round links', () => {
    const r = matchRows(base);
    expect(r.title).toBe('Match 0123456789abcdef');
    expect(r.status).toBe('waiting for scripts (next: round 2)');
    expect(r.seats.map((s) => s.text)).toEqual(['seat a — alice: 1 win, next script submitted', 'seat b — bob: 0 wins, next script not submitted']);
    expect(r.rounds[0]).toEqual({ n: 1, text: 'Round 1: seat a (alice) was p1 — seat a (alice) won by ko at f300 (hp a 50, b 0)', href: '/view/matches/0123456789abcdef/rounds/1' });
    expect(r.outcome).toBeNull();
  });
  it('describes finished and drawn matches', () => {
    const won = matchRows({ ...base, status: 'finished', winner: 'b', rounds: [round(1, 'a', 'b'), round(2, 'b', 'b')] });
    expect(won.finished).toBe(true);
    expect(won.outcome).toBe('seat b (bob) won the match');
    expect(won.rounds[1]!.text).toContain('Round 2: seat b (bob) was p1 — seat b (bob) won');
    expect(won.seats[0]!.text).toBe('seat a — alice: 0 wins');
    expect(won.seats[1]!.text).toBe('seat b — bob: 2 wins');
    const draw = matchRows({ ...base, status: 'finished', winner: null });
    expect(draw.outcome).toBe('the match was a draw');
  });
});

describe('newRoundToOpen', () => {
  it('ignores the first load', () => {
    expect(newRoundToOpen(null, base)).toBeNull();
  });
  it('returns null when nothing new was played', () => {
    expect(newRoundToOpen(base, { ...base })).toBeNull();
  });
  it('opens the newest round, even when several were played between polls', () => {
    expect(newRoundToOpen(base, { ...base, rounds: [...base.rounds, round(2, 'b', 'b')] })).toBe(2);
    expect(newRoundToOpen(base, { ...base, rounds: [...base.rounds, round(2, 'b', null), round(3, 'a', 'a')] })).toBe(3);
  });
});
