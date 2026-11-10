import { describe, expect, it } from 'vitest';
import { byteColToOffset, toDiagnostics } from '../src/diagnostics';
import { ACTIONS, classifyWord, OVERLAY_ATTACKS, pfToken, SENSORS, type PfToken } from '../src/pflang';

const samples = import.meta.glob<string>('../../scripts/*.pf', { eager: true, query: '?raw', import: 'default' });
const guide = Object.values(import.meta.glob<string>('../../docs/PLAYER_GUIDE.md', { eager: true, query: '?raw', import: 'default' }))[0]!;

describe('toDiagnostics', () => {
  it('maps ASCII line/col to a word range', () => {
    const text = 'punch\n\n    kik\nwait 1';
    const [d] = toDiagnostics(text, [{ line: 3, col: 5, message: 'unknown action "kik"' }]);
    expect(text.slice(d!.from, d!.to)).toBe('kik');
    expect(d!.severity).toBe('error');
    expect(d!.message).toBe('unknown action "kik"');
  });
  it('treats columns as UTF-8 byte offsets', () => {
    // "é" is 2 bytes and "🥊" 4 bytes in UTF-8 but 1 and 2 UTF-16 units.
    const line = 'if é🥊 kik';
    expect(byteColToOffset(line, 1)).toBe(0);
    const col = new TextEncoder().encode('if é🥊 ').length + 1;
    const [d] = toDiagnostics(line, [{ line: 1, col, message: 'm' }]);
    expect(line.slice(d!.from, d!.to)).toBe('kik');
  });
  it('clamps errors past the end of the document', () => {
    const text = 'loop { punch';
    const [d] = toDiagnostics(text, [{ line: 9, col: 1, message: 'missing }' }]);
    expect(d!.from).toBe(text.length);
    expect(d!.to).toBe(text.length);
  });
  it('handles an end-of-file error on the last line', () => {
    const text = 'loop { punch';
    const [d] = toDiagnostics(text, [{ line: 1, col: 13, message: 'missing }' }]);
    expect(d!.from).toBe(text.length);
    expect(d!.to).toBe(text.length);
  });
  it('marks a single symbol when no word follows', () => {
    const [d] = toDiagnostics('kick;', [{ line: 1, col: 5, message: '; not needed' }]);
    expect(d!.from).toBe(4);
    expect(d!.to).toBe(5);
  });
});

/** Minimal StringStream for driving pfToken over one line. */
function tokenize(line: string): { text: string; tok: PfToken }[] {
  let pos = 0;
  const out: { text: string; tok: PfToken }[] = [];
  const stream = {
    eatSpace() {
      const s = pos;
      while (pos < line.length && /\s/.test(line[pos]!)) pos++;
      return pos > s;
    },
    match(re: RegExp) {
      const m = re.exec(line.slice(pos));
      if (!m || m.index !== 0) return null;
      pos += m[0].length;
      return m;
    },
    skipToEnd() {
      pos = line.length;
    },
    next() {
      return line[pos++];
    },
    peek() {
      return line[pos];
    },
  };
  while (pos < line.length) {
    const start = pos;
    const tok = pfToken(stream);
    if (tok) out.push({ text: line.slice(start, pos), tok });
  }
  return out;
}

describe('pf tokenizer', () => {
  it('classifies every token in every sample bot', () => {
    expect(Object.keys(samples)).toHaveLength(6);
    for (const [path, src] of Object.entries(samples)) {
      for (const line of src.split('\n')) {
        for (const { text, tok } of tokenize(line)) {
          expect(tok, `${path}: ${JSON.stringify(text)} in ${JSON.stringify(line)}`).not.toBe('unknown');
        }
      }
    }
  });
  it('recognises the token kinds', () => {
    expect(tokenize('loop { if in_range(sweep) and not enemy.airborne { move_forward x3 } } on_hit { duck hold 10 } # c').map((x) => x.tok)).toEqual([
      'keyword', 'bracket', 'keyword', 'sensor', 'bracket', 'action', 'bracket', 'keyword', 'keyword', 'sensor', 'bracket', 'action', 'number', 'bracket', 'bracket',
      'handler', 'bracket', 'action', 'keyword', 'number', 'bracket', 'comment',
    ]);
    expect(classifyWord('jump_kick')).toBe('attack');
    expect(classifyWord('kik')).toBe('unknown');
    expect(tokenize('distance <= 12').map((x) => x.tok)).toEqual(['sensor', 'operator', 'number']);
  });
  it('matches the action and sensor tables in the player guide', () => {
    const section = (title: string) => {
      const start = guide.indexOf(title);
      return guide.slice(start, guide.indexOf('\n## ', start + 1));
    };
    const firstCellNames = (md: string) =>
      md
        .split('\n')
        .filter((l) => l.startsWith('| `'))
        .flatMap((l) => [...(l.split('|')[1] ?? '').matchAll(/`([a-z_.]+)[^`]*`/g)].map((m) => m[1]!));
    expect(new Set(firstCellNames(section('## 3. Actions')))).toEqual(new Set(ACTIONS));
    expect(new Set(firstCellNames(section('## 4. Sensors and conditions')))).toEqual(new Set(SENSORS));
    for (const a of OVERLAY_ATTACKS) expect(guide).toContain('`' + a + '`');
  });
});
