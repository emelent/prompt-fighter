// Event callouts ("BLOCKED", "THROW", "TECH", "TRADE", "K.O.") and the frame
// countdown. Pure functions of (log, frame), so stepping, scrubbing and frame
// links show exactly what playing up to that frame shows.
import { boostMultiplier } from './replay';
import type { Log, PlayerID } from './types';

/** Frames a callout stays up after its event (K.O. lasts to the end). */
export const CALLOUT_FRAMES = 30;

export type CalloutKind = 'block' | 'throw' | 'tech' | 'trade' | 'ko' | 'guard_break' | 'boost' | 'stale';

export interface Callout {
  text: string;
  kind: CalloutKind;
  /** Above a fighter, between the fighters, or large in the centre band. */
  anchor: PlayerID | 'mid' | 'center';
  /** Frames since the event (0 on the event frame). */
  age: number;
  frame: number;
  /** Opacity: 1 on the event frame, fading to 0 as the callout expires (K.O. stays 1). */
  alpha: number;
}

/** The callouts visible on frame f, oldest first. */
export function calloutsAt(log: Log, f: number): Callout[] {
  const out: Callout[] = [];
  let techFrame = -1;
  for (const e of log.events) {
    if (e.frame > f) break; // events are ordered by frame
    const age = f - e.frame;
    if (e.type === 'ko') {
      if (!out.some((c) => c.kind === 'ko')) out.push({ text: 'K.O.', kind: 'ko', anchor: 'center', age, frame: e.frame, alpha: 1 });
      continue;
    }
    if (age >= CALLOUT_FRAMES) continue;
    const push = (text: string, kind: CalloutKind, anchor: Callout['anchor']) =>
      out.push({ text, kind, anchor, age, frame: e.frame, alpha: 1 - age / CALLOUT_FRAMES });
    if (e.type === 'block' && e.target) {
      push('BLOCKED', 'block', e.target);
    } else if (e.type === 'guard_break' && e.target) {
      push('GUARD BREAK!', 'guard_break', e.target);
    } else if (e.type === 'throw' && e.detail.outcome === 'success' && e.target) {
      push('THROW', 'throw', e.target);
    } else if (e.type === 'throw' && e.detail.outcome === 'tech') {
      // Both fighters emit a tech event on the same frame; show it once.
      if (techFrame !== e.frame) push('TECH', 'tech', 'mid');
      techFrame = e.frame;
    } else if (e.type === 'trade') {
      push('TRADE', 'trade', 'mid');
    } else if (e.type === 'stale') {
      push('STALE', 'stale', e.actor);
    }
    // A boosted hit or throw also shows its multiplier above the target.
    if ((e.type === 'hit' || e.type === 'throw') && e.target && (e.detail.boost ?? 0) > 100) {
      push(`×${boostMultiplier(e.detail.boost!)}`, 'boost', e.target);
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
