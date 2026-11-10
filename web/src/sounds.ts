// Which sound effects play as playback moves from one frame to the next. Pure
// (log, from, to) → cues, so it is unit-tested directly; src/stage.ts plays them.
import type { Log } from './types';

export type SoundKey = 'punch_hit' | 'punch_miss' | 'kick_hit' | 'kick_miss' | 'block' | 'jump';

/** Every sound effect, for preloading. The background music is separate. */
export const SOUND_KEYS: readonly SoundKey[] = ['punch_hit', 'punch_miss', 'kick_hit', 'kick_miss', 'block', 'jump'];

/** Frames playback may advance in one step and still play sounds; bigger jumps are seeks. */
export const MAX_SOUND_STEP = 12;

/** Attacks that use the kick sounds; every other attack uses the punch sounds. */
const KICKS = new Set(['kick', 'sweep', 'jump_kick']);

const family = (attack: string | undefined): 'punch' | 'kick' => (attack && KICKS.has(attack) ? 'kick' : 'punch');

export interface SoundCue {
  frame: number;
  key: SoundKey;
}

/**
 * The cues for frames after `from` up to and including `to`, in frame order,
 * with each sound at most once per frame (a trade's two hits sound once).
 * Moving backwards, or forwards by more than MAX_SOUND_STEP, plays nothing.
 */
export function soundsBetween(log: Log, from: number, to: number): SoundCue[] {
  if (to <= from || to - from > MAX_SOUND_STEP) return [];
  const cues: SoundCue[] = [];
  const add = (frame: number, key: SoundKey) => {
    if (!cues.some((c) => c.frame === frame && c.key === key)) cues.push({ frame, key });
  };
  for (let f = from + 1; f <= to; f++) {
    const fr = log.frames[f];
    const prev = log.frames[f - 1];
    if (!fr) break;
    // A fighter leaving the ground.
    if (prev && ((fr.p1.state === 'airborne' && prev.p1.state !== 'airborne') || (fr.p2.state === 'airborne' && prev.p2.state !== 'airborne'))) {
      add(f, 'jump');
    }
  }
  for (const e of log.events) {
    if (e.frame <= from) continue;
    if (e.frame > to) break; // events are ordered by frame
    switch (e.type) {
      case 'hit':
        add(e.frame, `${family(e.detail.attack)}_hit`);
        break;
      case 'throw':
        if (e.detail.outcome === 'success') add(e.frame, 'kick_hit');
        break;
      case 'whiff':
        add(e.frame, `${family(e.detail.attack)}_miss`);
        break;
      case 'block':
      case 'guard_break':
        add(e.frame, 'block');
        break;
    }
  }
  return cues.sort((a, b) => a.frame - b.frame);
}

/**
 * Picks one fight-music track at random (rand returns [0, 1), like
 * Math.random). Tracks are sorted first, so the pick depends only on rand.
 * Returns null when there are none.
 */
export function pickTrack(urls: readonly string[], rand: () => number = Math.random): string | null {
  if (urls.length === 0) return null;
  const sorted = [...urls].sort();
  return sorted[Math.min(sorted.length - 1, Math.floor(rand() * sorted.length))]!;
}
