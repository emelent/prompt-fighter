// Pure logic for watching a match: the waiting screen's state, which round to
// watch next, the button at the end of a replay, the countdown between rounds,
// and the simple round player's remembered settings.
import { SPEEDS, type Speed } from './replay';
import type { MatchView, RoundView, Seat } from './types';

/** Seconds counted down before the next round opens. */
export const COUNTDOWN_SECONDS = 5;

export interface SeatStatus {
  seat: Seat;
  name: string;
  wins: number;
  submitted: boolean;
}

export interface Score {
  a: { name: string; wins: number };
  b: { name: string; wins: number } | null;
}

export type WaitingState =
  | { kind: 'not_found' }
  | { kind: 'waiting'; round: number; score: Score; seats: SeatStatus[]; waitingFor: string[]; rounds: RoundView[] }
  | { kind: 'finished'; winner: string | null; score: Score; rounds: RoundView[] };

function score(v: MatchView): Score {
  const a = v.seats.a;
  const b = v.seats.b;
  return { a: { name: a?.name ?? 'seat a', wins: a?.wins ?? 0 }, b: b ? { name: b.name, wins: b.wins } : null };
}

/** What the waiting screen shows for a match (null: the match does not exist). */
export function waitingState(v: MatchView | null): WaitingState {
  if (!v) return { kind: 'not_found' };
  if (v.status === 'finished') {
    const w = v.winner === null ? null : (v.winner === 'a' ? v.seats.a : v.seats.b)?.name ?? `seat ${v.winner}`;
    return { kind: 'finished', winner: w, score: score(v), rounds: v.rounds };
  }
  const seats: SeatStatus[] = [];
  const waitingFor: string[] = [];
  for (const seat of ['a', 'b'] as const) {
    const s = seat === 'a' ? v.seats.a : v.seats.b;
    if (!s) {
      waitingFor.push('an opponent to join');
      continue;
    }
    seats.push({ seat, name: s.name, wins: s.wins, submitted: s.submitted });
    if (!s.submitted) waitingFor.push(s.name);
  }
  return { kind: 'waiting', round: v.next_round, score: score(v), seats, waitingFor, rounds: v.rounds };
}

/** The highest round number already played (0 before round 1). */
export function lastPlayed(v: MatchView): number {
  return v.rounds.reduce((m, r) => Math.max(m, r.n), 0);
}

/**
 * The next round to watch: the lowest played round after `watched`, or null.
 * `watched` is null for a waiting screen opened directly, which counts every
 * round played so far as watched.
 */
export function nextToWatch(v: MatchView, watched: number | null, openedWith: number): number | null {
  const after = watched ?? openedWith;
  const next = v.rounds.map((r) => r.n).filter((n) => n > after).sort((x, y) => x - y);
  return next[0] ?? null;
}

/** Parses the `after` query parameter (the round just watched), or null. */
export function parseAfter(search: string): number | null {
  const raw = new URLSearchParams(search).get('after');
  if (raw === null || !/^\d+$/.test(raw)) return null;
  return Number(raw);
}

export type EndButton = { kind: 'next'; round: number; href: string } | { kind: 'done'; href: string };

/** The button shown at the end of a spectator's replay of round n. */
export function endButton(v: MatchView, n: number): EndButton {
  const id = encodeURIComponent(v.match_id);
  if (v.rounds.some((r) => r.n === n + 1)) {
    return { kind: 'next', round: n + 1, href: `/watch/matches/${id}/rounds/${n + 1}?autoplay=1` };
  }
  return { kind: 'done', href: `/watch/matches/${id}?after=${n}` };
}

/** Timer functions, injectable for tests. */
export interface Timers {
  setInterval(fn: () => void, ms: number): unknown;
  clearInterval(handle: unknown): void;
}

const realTimers: Timers = {
  setInterval: (fn, ms) => globalThis.setInterval(fn, ms),
  clearInterval: (h) => globalThis.clearInterval(h as ReturnType<typeof setInterval>),
};

/** Counts down once per second: onTick(5), onTick(4) … onTick(1), then onDone(). */
export class Countdown {
  private handle: unknown = null;

  constructor(private readonly timers: Timers = realTimers) {}

  get running(): boolean {
    return this.handle !== null;
  }

  start(seconds: number, onTick: (left: number) => void, onDone: () => void): void {
    this.cancel();
    let left = seconds;
    onTick(left);
    this.handle = this.timers.setInterval(() => {
      left--;
      if (left > 0) {
        onTick(left);
        return;
      }
      this.cancel();
      onDone();
    }, 1000);
  }

  cancel(): void {
    if (this.handle !== null) this.timers.clearInterval(this.handle);
    this.handle = null;
  }
}

// --- simple round player settings ----------------------------------------------

export interface SimpleSettings {
  /** Show both script panels. */
  scripts: boolean;
  /** Show the event feed. */
  feed: boolean;
  /** Show the scrub bar, step buttons and frame counter. */
  timeline: boolean;
  /** Show the speed control. */
  speed: boolean;
  /** Show the checks and ranges toggles. */
  toggles: boolean;
  /** Sound effects and music on. Shared with the lobby's menu music. */
  sound: boolean;
  playbackSpeed: Speed;
}

export const DEFAULT_SETTINGS: Readonly<SimpleSettings> = { scripts: false, feed: false, timeline: false, speed: false, toggles: false, sound: true, playbackSpeed: 0.5 };

export const SETTINGS_KEY = 'pf.simple';

/** Storage functions, injectable for tests (defaults to localStorage). */
export interface Store {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

function defaultStore(): Store | null {
  try {
    return globalThis.localStorage ?? null;
  } catch {
    return null;
  }
}

/** The remembered settings, falling back to the defaults field by field. */
export function loadSimpleSettings(store: Store | null = defaultStore()): SimpleSettings {
  const out: SimpleSettings = { ...DEFAULT_SETTINGS };
  let raw: unknown = null;
  try {
    const text = store?.getItem(SETTINGS_KEY);
    raw = text ? JSON.parse(text) : null;
  } catch {
    raw = null;
  }
  if (typeof raw !== 'object' || raw === null) return out;
  const r = raw as Record<string, unknown>;
  for (const k of ['scripts', 'feed', 'timeline', 'speed', 'toggles', 'sound'] as const) {
    if (typeof r[k] === 'boolean') out[k] = r[k] as boolean;
  }
  if ((SPEEDS as readonly unknown[]).includes(r.playbackSpeed)) out.playbackSpeed = r.playbackSpeed as Speed;
  return out;
}

/** Remembers the settings; storage failures are ignored (the defaults apply next time). */
export function saveSimpleSettings(s: SimpleSettings, store: Store | null = defaultStore()): void {
  try {
    store?.setItem(SETTINGS_KEY, JSON.stringify(s));
  } catch {
    // Storage blocked: the settings last for this page only.
  }
}

/** The match id in a pasted id or link (e.g. a /play or /watch link), or null. */
export function matchIdFrom(text: string): string | null {
  const t = text.trim();
  const inLink = /\/(?:play|watch|view)\/matches\/([0-9a-f]{16})(?:[/?#]|$)/i.exec(t);
  if (inLink) return inLink[1]!.toLowerCase();
  return /^[0-9a-f]{16}$/i.test(t) ? t.toLowerCase() : null;
}
