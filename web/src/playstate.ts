// Which screen the play page shows, as a pure function of the match state and
// the seat (if any) this browser holds.
import type { MatchView, Seat, SeatView } from './types';

export interface StoredSeat {
  seat: Seat;
  token: string;
  name: string;
}

export type PlayState =
  | { kind: 'notFound' }
  | { kind: 'spectator'; view: MatchView }
  | { kind: 'join'; view: MatchView }
  | { kind: 'waitingOpponent'; view: MatchView; seat: StoredSeat }
  | { kind: 'yourMove'; view: MatchView; seat: StoredSeat; round: number }
  | { kind: 'waitingTheirs'; view: MatchView; seat: StoredSeat; round: number }
  | { kind: 'finished'; view: MatchView; seat: StoredSeat | null };

export function seatView(v: MatchView, s: Seat): SeatView | null {
  return s === 'a' ? v.seats.a : v.seats.b;
}

export function otherSeat(s: Seat): Seat {
  return s === 'a' ? 'b' : 'a';
}

/** Whether a visitor can still take a seat (seat a, then seat b). */
export function hasFreeSeat(view: MatchView): boolean {
  return view.seats.a === null || view.seats.b === null;
}

export function playState(view: MatchView | null, seat: StoredSeat | null): PlayState {
  if (!view) return { kind: 'notFound' };
  if (view.status === 'finished') return { kind: 'finished', view, seat };
  const visitor = (): PlayState => (hasFreeSeat(view) ? { kind: 'join', view } : { kind: 'spectator', view });
  if (!seat) return visitor();
  const mine = seatView(view, seat.seat);
  // A stored seat this match does not know about (e.g. data reset): treat as a visitor.
  if (!mine) return visitor();
  if (seatView(view, otherSeat(seat.seat)) === null) return { kind: 'waitingOpponent', view, seat };
  return mine.submitted ? { kind: 'waitingTheirs', view, seat, round: view.next_round } : { kind: 'yourMove', view, seat, round: view.next_round };
}
