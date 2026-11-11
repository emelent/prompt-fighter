import { describe, expect, it } from 'vitest';
import { matchRows } from '../src/match';
import { allSeen, markSeen, seenRounds, UNSEEN_MATCH_TEXT, UNSEEN_TEXT, visibleWins } from '../src/seen';
import { waitingState } from '../src/spectate';
import { buildScene } from '../src/scene';
import type { MatchView, RoundView } from '../src/types';
import { goldenLogs } from './fixtures';

function round(n: number, winner: 'a' | 'b' | null): RoundView {
  return { n, p1: 'a', winner, reason: winner ? 'ko' : 'timeout', end_frame: 700, hp: { a: 50, b: 0 }, config_version: '2026.11-4', log: `/matches/m/rounds/${n}` };
}

const finished: MatchView = {
  match_id: 'fedcba9876543210',
  status: 'finished',
  config_version: '2026.11-4',
  next_round: 3,
  seats: { a: { name: 'alice', wins: 2, submitted: false }, b: { name: 'bob', wins: 0, submitted: false } },
  rounds: [round(1, 'a'), round(2, 'a')],
  winner: 'a',
};

describe('seen rounds', () => {
  it('remembers watched rounds per match (in memory when storage is unavailable)', () => {
    expect(seenRounds('m1').size).toBe(0);
    markSeen('m1', 2);
    markSeen('m1', 1);
    expect([...seenRounds('m1')].sort()).toEqual([1, 2]);
    expect(seenRounds('m2').size).toBe(0);
  });

  it('counts wins and the match result only from watched rounds', () => {
    expect(visibleWins(finished, new Set())).toEqual({ a: 0, b: 0 });
    expect(visibleWins(finished, new Set([1]))).toEqual({ a: 1, b: 0 });
    expect(allSeen(finished, new Set([1]))).toBe(false);
    expect(allSeen(finished, new Set([1, 2]))).toBe(true);
  });

  it('keeps unwatched results off the match page', () => {
    const rows = matchRows(finished, new Set([1]));
    expect(rows.rounds[0]!.text).toContain('won by ko');
    expect(rows.rounds[1]!.text).toContain(UNSEEN_TEXT);
    expect(rows.rounds[1]!.text).not.toMatch(/won|ko|f700|hp/);
    expect(rows.seats[0]!.text).toContain('1 win');
    expect(rows.outcome).toBe(UNSEEN_MATCH_TEXT);
    expect(matchRows(finished, new Set([1, 2])).outcome).toContain('won the match');
  });

  it('keeps unwatched results off the waiting screen', () => {
    const none = waitingState(finished, '', new Set());
    expect(none).toMatchObject({ kind: 'finished', revealed: false, score: { a: { wins: 0 }, b: { wins: 0 } } });
    expect(waitingState(finished, '', new Set([1, 2]))).toMatchObject({ revealed: true, score: { a: { wins: 2 } } });
  });

  it('leaves the round length out of the stage frame label', () => {
    const l = goldenLogs['rushdown_vs_turtle']!;
    expect(buildScene(l, 10, { hideEnd: true }).frameLabel).toBe('f 10');
    expect(buildScene(l, 10).frameLabel).toMatch(/^f 10 \/ \d+$/);
  });
});
