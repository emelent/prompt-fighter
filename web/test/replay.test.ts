import { describe, expect, it } from 'vitest';
import { clampFrame, eventLines, faster, frameHash, parseFrameHash, parseLog, resultLine, slower, validateLog } from '../src/replay';
import { goldenLogs, textSnapshots } from './fixtures';

const anyLog = goldenLogs.rushdown_vs_turtle!;

describe('validateLog', () => {
  it('accepts every golden log', () => {
    for (const log of Object.values(goldenLogs)) {
      expect(validateLog(log)).toBeNull();
    }
  });
  it('rejects non-objects', () => {
    expect(validateLog(null)).toMatch(/expected a JSON object/);
    expect(validateLog([1, 2])).toMatch(/expected a JSON object/);
    expect(validateLog('x')).toMatch(/expected a JSON object/);
  });
  it('names an unsupported format version', () => {
    expect(validateLog({ ...anyLog, format_version: '2' })).toBe('log format version 2 is not supported by this viewer (it reads version 1)');
    expect(validateLog({ ...anyLog, format_version: undefined })).toMatch(/version missing/);
  });
  it('rejects empty frames and wrong player counts', () => {
    expect(validateLog({ ...anyLog, frames: [] })).toBe('the log has no frames to play');
    expect(validateLog({ ...anyLog, players: [anyLog.players[0]] })).toMatch(/exactly two players/);
    expect(validateLog({ ...anyLog, frames: anyLog.frames.slice(1) })).toMatch(/numbered/);
    expect(validateLog({ ...anyLog, events: undefined })).toMatch(/missing events/);
  });
  it('parseLog reports invalid JSON', () => {
    const r = parseLog('{"format_version": ');
    expect('error' in r && r.error).toMatch(/^the file is not valid JSON/);
    const ok = parseLog(JSON.stringify(anyLog));
    expect('log' in ok).toBe(true);
  });
});

describe('event text parity with pf sim', () => {
  const pairs = Object.keys(goldenLogs);
  it('covers all 12 snapshots', () => {
    expect(Object.keys(textSnapshots)).toHaveLength(12);
  });
  for (const pair of pairs) {
    it(`${pair}: lines and result equal pf sim (without and with checks)`, () => {
      const log = goldenLogs[pair]!;
      for (const [key, checks] of [[pair, false], [`${pair}.v`, true]] as const) {
        const snap = textSnapshots[key];
        expect(snap, key).toBeDefined();
        const lines = snap!.trimEnd().split('\n');
        const result = lines.pop();
        expect(eventLines(log, checks)).toEqual(lines);
        expect(resultLine(log)).toBe(result);
      }
    });
  }
});

describe('frames, hashes, speeds', () => {
  it('clamps frames', () => {
    expect(clampFrame(999, 300)).toBe(300);
    expect(clampFrame(-5, 300)).toBe(0);
    expect(clampFrame(12.6, 300)).toBe(13);
    expect(clampFrame(Number.NaN, 300)).toBe(0);
  });
  it('parses and builds frame hashes', () => {
    expect(parseFrameHash('#f=412')).toBe(412);
    expect(parseFrameHash('#x=1&f=7')).toBe(7);
    expect(parseFrameHash('')).toBeNull();
    expect(parseFrameHash('#f=abc')).toBeNull();
    expect(parseFrameHash('#f=-3')).toBeNull();
    expect(frameHash(42)).toBe('#f=42');
  });
  it('steps speeds within 0.25x-4x', () => {
    expect(slower(1)).toBe(0.5);
    expect(slower(0.25)).toBe(0.25);
    expect(faster(1)).toBe(2);
    expect(faster(4)).toBe(4);
  });
});
