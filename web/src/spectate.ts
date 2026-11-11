// Pure logic for watching a match: the waiting screen's state, which round to
// watch next, the button at the end of a replay, the countdown between rounds,
// and the simple round player's remembered settings.
import { SPEEDS, type Speed } from './replay';
import { allSeen, visibleWins } from './seen';
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

export interface SetupInfo {
  /** Names of the players who have joined, seat a first. */
  joined: string[];
  /** The one game link both players use: /play/matches/{id} on this server. */
  gameLink: string;
  /** This server's origin, for the pf mcp --server setup. */
  origin: string;
  /** Claude Code command that adds the plugin's marketplace (run first, on its own). */
  marketplaceCommand: string;
  /** Claude Code command that installs the pf plugin (run second). */
  installCommand: string;
  /** The plugin command that joins this match. */
  joinCommand: string;
  /** The plugin command that writes the next round's script (roundN.pf). */
  scriptCommand: string;
  /** The plugin command that submits the next round's script. */
  submitCommand: string;
}

/** Where the pf plugin's marketplace lives (this repository). */
export const PLUGIN_MARKETPLACE = 'emelent/prompt-fighter';

export type WaitingState =
  | { kind: 'not_found' }
  | { kind: 'setup'; setup: SetupInfo; score: Score }
  | { kind: 'waiting'; round: number; score: Score; seats: SeatStatus[]; waitingFor: string[]; rounds: RoundView[] }
  | { kind: 'finished'; winner: string | null; /** every round watched, so the winner may show */ revealed: boolean; score: Score; rounds: RoundView[] };

function score(v: MatchView, seen: Set<number>): Score {
  const a = v.seats.a;
  const b = v.seats.b;
  const wins = visibleWins(v, seen);
  return { a: { name: a?.name ?? 'seat a', wins: wins.a }, b: b ? { name: b.name, wins: wins.b } : null };
}

/** How players join a match: the game link and LLM instructions for this server. */
export function setupInfo(v: MatchView, origin: string): SetupInfo {
  const gameLink = `${origin}/play/matches/${encodeURIComponent(v.match_id)}`;
  const joined = [v.seats.a?.name, v.seats.b?.name].filter((n): n is string => !!n);
  return {
    joined,
    gameLink,
    origin,
    marketplaceCommand: `/plugin marketplace add ${PLUGIN_MARKETPLACE}`,
    installCommand: '/plugin install pf@prompt-fighter',
    joinCommand: `/pf:join ${gameLink}`,
    scriptCommand: '/pf:script <how you want to fight>',
    submitCommand: '/pf:submit',
  };
}

/**
 * What the waiting screen shows for a match (null: the match does not exist).
 * origin is this server's origin (location.origin), used for the game link.
 */
export function waitingState(v: MatchView | null, origin = '', seenRounds?: Set<number>): WaitingState {
  if (!v) return { kind: 'not_found' };
  // Rounds not watched yet keep their results (and the score) hidden.
  const seen = seenRounds ?? new Set(v.rounds.map((r) => r.n));
  if (v.status === 'waiting_for_players' || v.status === 'waiting_for_opponent') {
    return { kind: 'setup', setup: setupInfo(v, origin), score: score(v, seen) };
  }
  if (v.status === 'finished') {
    const w = v.winner === null ? null : (v.winner === 'a' ? v.seats.a : v.seats.b)?.name ?? `seat ${v.winner}`;
    return { kind: 'finished', winner: w, revealed: allSeen(v, seen), score: score(v, seen), rounds: v.rounds };
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
  return { kind: 'waiting', round: v.next_round, score: score(v, seen), seats, waitingFor, rounds: v.rounds };
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
