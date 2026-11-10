// Pure helpers for the live match page (GET /matches/{id}).
import type { MatchView, Seat, SeatView } from './types';

export const POLL_MS = 2000;
export const POLL_BACKOFF_MS = 5000;

export interface SeatRow {
  seat: Seat;
  name: string;
  wins: number;
  submitted: boolean;
  text: string;
}

export interface RoundRow {
  n: number;
  text: string;
  href: string;
}

export interface MatchRows {
  title: string;
  status: string;
  seats: SeatRow[];
  rounds: RoundRow[];
  outcome: string | null;
  finished: boolean;
}

const STATUS_TEXT: Record<MatchView['status'], string> = {
  waiting_for_opponent: 'waiting for an opponent to join',
  awaiting_scripts: 'waiting for scripts',
  finished: 'finished',
};

function seatName(v: MatchView, seat: Seat): string {
  const s = seat === 'a' ? v.seats.a : v.seats.b;
  return s ? `seat ${seat} (${s.name})` : `seat ${seat}`;
}

function seatRow(seat: Seat, s: SeatView | null, finished: boolean): SeatRow {
  if (!s) {
    return { seat, name: '', wins: 0, submitted: false, text: `seat ${seat}: not joined yet` };
  }
  const sub = finished ? '' : s.submitted ? ', next script submitted' : ', next script not submitted';
  return { seat, name: s.name, wins: s.wins, submitted: s.submitted, text: `seat ${seat} — ${s.name}: ${s.wins} win${s.wins === 1 ? '' : 's'}${sub}` };
}

/** Text rows describing a match, with links to each played round's player. */
export function matchRows(v: MatchView): MatchRows {
  const finished = v.status === 'finished';
  const rounds = v.rounds.map((r) => {
    const result = r.winner ? `${seatName(v, r.winner)} won` : 'draw';
    return {
      n: r.n,
      text: `Round ${r.n}: ${seatName(v, r.p1)} was p1 — ${result} by ${r.reason} at f${r.end_frame} (hp a ${r.hp.a}, b ${r.hp.b})`,
      href: `/view/matches/${encodeURIComponent(v.match_id)}/rounds/${r.n}`,
    };
  });
  let outcome: string | null = null;
  if (finished) {
    outcome = v.winner ? `${seatName(v, v.winner)} won the match` : 'the match was a draw';
  }
  return {
    title: `Match ${v.match_id}`,
    status: finished ? STATUS_TEXT.finished : `${STATUS_TEXT[v.status]} (next: round ${v.next_round})`,
    seats: [seatRow('a', v.seats.a, finished), seatRow('b', v.seats.b, finished)],
    rounds,
    outcome,
    finished,
  };
}

/**
 * The round to auto-open after a poll: the highest round number that is new
 * since prev. Returns null on the first load (prev null) or when nothing new.
 */
export function newRoundToOpen(prev: MatchView | null, cur: MatchView): number | null {
  if (!prev || cur.rounds.length <= prev.rounds.length) {
    return null;
  }
  const last = cur.rounds[cur.rounds.length - 1];
  return last ? last.n : null;
}
