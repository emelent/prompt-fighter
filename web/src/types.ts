// Types for the event log (format "1", docs/RULES.md §7) and the match state
// returned by GET /matches/{id} (docs/API.md).

export type PlayerID = 'p1' | 'p2';

export interface Player {
  id: PlayerID;
  name: string;
  script: string;
}

export interface Result {
  winner: PlayerID | null;
  reason: 'ko' | 'double_ko' | 'timeout' | string;
  end_frame: number;
  hp: [number, number];
}

export interface FighterFrame {
  x: number;
  y: number;
  facing: 1 | -1;
  state: string;
  anim: string;
  anim_frame: number;
  phase: '' | 'startup' | 'active' | 'recovery';
  hp: number;
  invulnerable: boolean;
  combo: number;
  script_line: number;
}

export interface Frame {
  f: number;
  p1: FighterFrame;
  p2: FighterFrame;
}

export interface EventDetail {
  attack?: string;
  damage?: number;
  combo?: number;
  hp?: number;
  reason?: string;
  outcome?: string;
  sensor?: string;
  value?: number | boolean;
  line?: number;
  cost?: number;
  handler?: string;
  attacks?: string[];
  damages?: number[];
  /** hit/throw: damage percent from the target's passivity, when above 100. */
  boost?: number;
  /** stale: starts of this attack in a row. */
  repeats?: number;
  /** stale: the reduced range. */
  range?: number;
}

export interface GameEvent {
  frame: number;
  type: string;
  actor: PlayerID;
  target?: PlayerID;
  detail: EventDetail;
}

export interface Log {
  format_version: string;
  config_version: string;
  frame_budget: number;
  players: [Player, Player];
  result: Result;
  frames: Frame[];
  events: GameEvent[];
}

export type Seat = 'a' | 'b';

export interface SeatView {
  name: string;
  wins: number;
  submitted: boolean;
}

export interface RoundView {
  n: number;
  p1: Seat;
  winner: Seat | null;
  reason: string;
  end_frame: number;
  hp: { a: number; b: number };
  config_version: string;
  log: string;
}

export interface MatchView {
  match_id: string;
  status: 'waiting_for_opponent' | 'awaiting_scripts' | 'finished';
  config_version: string;
  next_round: number;
  seats: { a: SeatView | null; b: SeatView | null };
  rounds: RoundView[];
  winner: Seat | null;
}
