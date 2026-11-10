// Event callouts ("BLOCKED", "THROW", "TECH", "TRADE", "K.O.") and the frame
// countdown. Pure functions of (log, frame), so stepping, scrubbing and frame
// links show exactly what playing up to that frame shows.
import type { Log, PlayerID } from './types';

/** Frames a callout stays up after its event (K.O. lasts to the end). */
export const CALLOUT_FRAMES = 30;

export type CalloutKind = 'block' | 'throw' | 'tech' | 'trade' | 'ko';

export interface Callout {
  text: string;
  kind: CalloutKind;
  /** Above a fighter, between the fighters, or large in the centre band. */
  anchor: PlayerID | 'mid' | 'center';
  /** Frames since the event (0 on the event frame). */
  age: number;
  frame: number;
}

/** The callouts visible on frame f, oldest first. */
export function calloutsAt(log: Log, f: number): Callout[] {
  const out: Callout[] = [];
  let techFrame = -1;
  for (const e of log.events) {
    if (e.frame > f) break; // events are ordered by frame
    const age = f - e.frame;
    if (e.type === 'ko') {
      if (!out.some((c) => c.kind === 'ko')) out.push({ text: 'K.O.', kind: 'ko', anchor: 'center', age, frame: e.frame });
      continue;
    }
    if (age >= CALLOUT_FRAMES) continue;
    if (e.type === 'block' && e.target) {
      out.push({ text: 'BLOCKED', kind: 'block', anchor: e.target, age, frame: e.frame });
    } else if (e.type === 'throw' && e.detail.outcome === 'success' && e.target) {
      out.push({ text: 'THROW', kind: 'throw', anchor: e.target, age, frame: e.frame });
    } else if (e.type === 'throw' && e.detail.outcome === 'tech') {
      // Both fighters emit a tech event on the same frame; show it once.
      if (techFrame !== e.frame) out.push({ text: 'TECH', kind: 'tech', anchor: 'mid', age, frame: e.frame });
      techFrame = e.frame;
    } else if (e.type === 'trade') {
      out.push({ text: 'TRADE', kind: 'trade', anchor: 'mid', age, frame: e.frame });
    }
  }
  return out;
}

/** Rounds per match: best of 3, so round 3 is the deciding round. */
export const MAX_ROUNDS = 3;

/** The intro's round word for stored round n, or null when the round is unknown (local files). */
export function roundLabel(n: number | null): string | null {
  if (n === null || !Number.isInteger(n) || n < 1) return null;
  return n === MAX_ROUNDS ? 'FINAL ROUND' : `ROUND ${n}`;
}

/** Frames left in the round on frame f, zero-padded to at least three digits. */
export function countdownAt(log: Log, f: number): string {
  const left = Math.max(0, log.frame_budget - 1 - f);
  const width = Math.max(3, String(Math.max(0, log.frame_budget - 1)).length);
  return String(left).padStart(width, '0');
}
