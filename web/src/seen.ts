// Spoiler protection: a round's result stays hidden until this browser has
// watched its replay to the end. The record is per match, in localStorage;
// when storage is blocked it lasts for the page only.
import type { MatchView, RoundView, Seat } from './types';

const KEY = (id: string) => `pf.seen.${id}`;
const memory = new Map<string, Set<number>>();

/** The rounds of match id this browser has watched to the end. */
export function seenRounds(id: string): Set<number> {
  const mem = memory.get(id);
  try {
    const raw = localStorage.getItem(KEY(id));
    const stored = raw ? (JSON.parse(raw) as unknown) : [];
    const set = new Set<number>(Array.isArray(stored) ? stored.filter((n): n is number => Number.isInteger(n)) : []);
    for (const n of mem ?? []) set.add(n);
    return set;
  } catch {
    return new Set(mem ?? []);
  }
}

/** Records that round n of match id has been watched to the end. */
export function markSeen(id: string, n: number): void {
  const set = seenRounds(id);
  set.add(n);
  memory.set(id, set);
  try {
    localStorage.setItem(KEY(id), JSON.stringify([...set].sort((a, b) => a - b)));
  } catch {
    // Storage blocked: remembered in memory for this page.
  }
}

/** Whether round r's result may be shown. */
export function isSeen(r: RoundView, seen: Set<number>): boolean {
  return seen.has(r.n);
}

/** Wins per seat, counting only rounds that have been watched. */
export function visibleWins(v: MatchView, seen: Set<number>): Record<Seat, number> {
  const wins: Record<Seat, number> = { a: 0, b: 0 };
  for (const r of v.rounds) if (r.winner && isSeen(r, seen)) wins[r.winner]++;
  return wins;
}

/** Whether every played round has been watched, so the match result may show. */
export function allSeen(v: MatchView, seen: Set<number>): boolean {
  return v.rounds.every((r) => isSeen(r, seen));
}

/** Shown in place of an unwatched round's result. */
export const UNSEEN_TEXT = 'not watched yet';
/** Shown in place of the match result while some rounds are unwatched. */
export const UNSEEN_MATCH_TEXT = 'Watch every round to see who won the match.';
