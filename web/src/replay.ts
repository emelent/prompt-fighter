// Pure replay helpers: log validation, event text (a line-for-line port of
// sim/textlog.go), frame clamping, frame-link hashes and playback speeds.
import type { GameEvent, Log } from './types';

export const SUPPORTED_FORMAT = '1';

/** Returns a human-readable problem with `data`, or null if it is a playable log. */
export function validateLog(data: unknown): string | null {
  if (typeof data !== 'object' || data === null || Array.isArray(data)) {
    return 'not a replay log: expected a JSON object';
  }
  const d = data as Record<string, unknown>;
  if (d.format_version !== SUPPORTED_FORMAT) {
    const v = d.format_version === undefined ? 'missing' : String(d.format_version);
    return `log format version ${v} is not supported by this viewer (it reads version ${SUPPORTED_FORMAT})`;
  }
  const players = d.players;
  if (!Array.isArray(players) || players.length !== 2 || !players.every((p) => typeof p === 'object' && p !== null && typeof (p as Record<string, unknown>).script === 'string' && typeof (p as Record<string, unknown>).name === 'string')) {
    return 'not a replay log: expected exactly two players with names and scripts';
  }
  const frames = d.frames;
  if (!Array.isArray(frames) || frames.length === 0) {
    return 'the log has no frames to play';
  }
  const first = frames[0] as { f?: unknown } | undefined;
  const last = frames[frames.length - 1] as { f?: unknown } | undefined;
  if (first?.f !== 0 || last?.f !== frames.length - 1) {
    return 'not a replay log: frames are not numbered 0..N-1';
  }
  if (!Array.isArray(d.events)) {
    return 'not a replay log: missing events';
  }
  if (typeof d.result !== 'object' || d.result === null) {
    return 'not a replay log: missing result';
  }
  return null;
}

/** Parses and validates log text, e.g. from a file the user selected. */
export function parseLog(text: string): { log: Log } | { error: string } {
  let data: unknown;
  try {
    data = JSON.parse(text);
  } catch (e) {
    return { error: `the file is not valid JSON (${e instanceof Error ? e.message : String(e)})` };
  }
  const problem = validateLog(data);
  return problem === null ? { log: data as Log } : { error: problem };
}

function pad3(n: number): string {
  return String(n).padStart(3, '0');
}

/** The description part of a `pf sim` event line (describeEvent in Go). */
export function describeEvent(e: GameEvent): string {
  const d = e.detail;
  switch (e.type) {
    case 'hit':
      return `${d.attack} hits ${e.target} for ${d.damage} (combo ${d.combo ?? 0}, hp ${d.hp})`;
    case 'block':
      return `${d.attack} blocked by ${e.target}`;
    case 'whiff':
      return `${d.attack} whiffs (${d.reason})`;
    case 'throw':
      if (d.outcome === 'tech') {
        return 'throw teched';
      }
      return `throws ${e.target} for ${d.damage} (hp ${d.hp})`;
    case 'trade':
      return `trades ${d.attacks?.[0]} for ${d.attacks?.[1]} (${d.damages?.[0]}/${d.damages?.[1]})`;
    case 'knockdown':
      return `knocked down (${d.reason})`;
    case 'ko':
      return 'KO';
    case 'check':
      return `checks ${d.sensor} = ${JSON.stringify(d.value)} (line ${d.line})`;
    case 'script_end':
      return 'script ended';
    case 'handler_enter':
      return `enters ${d.handler}`;
  }
  return e.type;
}

/** A `pf sim` event line: "[f NNN] <actor> <description>". */
export function formatEvent(e: GameEvent): string {
  return `[f ${pad3(e.frame)}] ${e.actor} ${describeEvent(e)}`;
}

/** Event lines as printed by `pf sim` (check events only when includeChecks). */
export function eventLines(log: Log, includeChecks: boolean): string[] {
  return log.events.filter((e) => includeChecks || e.type !== 'check').map(formatEvent);
}

/** The `pf sim` result line. */
export function resultLine(log: Log): string {
  const r = log.result;
  const hp = `hp ${r.hp[0]}-${r.hp[1]}`;
  if (r.winner === null) {
    return `result: draw by ${r.reason} at f${r.end_frame} (${hp})`;
  }
  return `result: ${r.winner} wins by ${r.reason} at f${r.end_frame} (${hp})`;
}

/** The last frame index of a log. */
export function lastFrame(log: Log): number {
  return log.frames.length - 1;
}

/** Clamps f to 0..last and rounds it to a whole frame (NaN → 0). */
export function clampFrame(f: number, last: number): number {
  if (!Number.isFinite(f)) {
    return 0;
  }
  return Math.min(Math.max(Math.round(f), 0), Math.max(last, 0));
}

/** Parses a "#f=N" fragment; returns null when absent or malformed. */
export function parseFrameHash(hash: string): number | null {
  const m = /^#?(?:.*&)?f=(\d{1,6})(?:&.*)?$/.exec(hash);
  return m?.[1] !== undefined ? Number(m[1]) : null;
}

export function frameHash(f: number): string {
  return `#f=${f}`;
}

/** Playback speeds; 1× plays 60 frames per second. */
export const SPEEDS = [0.25, 0.5, 1, 2, 4] as const;
export type Speed = (typeof SPEEDS)[number];
export const FRAMES_PER_SECOND = 60;

export function slower(s: Speed): Speed {
  const i = SPEEDS.indexOf(s);
  return SPEEDS[Math.max(i - 1, 0)] ?? s;
}

export function faster(s: Speed): Speed {
  const i = SPEEDS.indexOf(s);
  return SPEEDS[Math.min(i + 1, SPEEDS.length - 1)] ?? s;
}
