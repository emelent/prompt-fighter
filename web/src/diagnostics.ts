// Maps validator errors (1-based line, 1-based *byte* column, as produced by
// sim/dsl) to editor ranges in UTF-16 code units.
import type { Diagnostic } from '@codemirror/lint';

export interface ValidationError {
  line: number;
  col: number;
  message: string;
  text?: string;
}

function utf8Len(cp: number): number {
  return cp < 0x80 ? 1 : cp < 0x800 ? 2 : cp < 0x10000 ? 3 : 4;
}

/** Converts a 1-based byte column within `line` to a UTF-16 offset in it. */
export function byteColToOffset(line: string, col: number): number {
  let bytes = 0;
  let i = 0;
  while (i < line.length && bytes < col - 1) {
    const cp = line.codePointAt(i) ?? 0;
    bytes += utf8Len(cp);
    i += cp > 0xffff ? 2 : 1;
  }
  return i;
}

const WORD = /[A-Za-z0-9_.]/;

export function toDiagnostics(text: string, errors: readonly ValidationError[]): Diagnostic[] {
  const lines = text.split('\n');
  const starts: number[] = [];
  let pos = 0;
  for (const l of lines) {
    starts.push(pos);
    pos += l.length + 1;
  }
  return errors.map((e) => {
    if (e.line < 1 || e.line > lines.length) {
      return { from: text.length, to: text.length, severity: 'error', message: e.message };
    }
    const lineText = lines[e.line - 1] ?? '';
    const start = starts[e.line - 1] ?? 0;
    const off = byteColToOffset(lineText, Math.max(e.col, 1));
    let end = off;
    while (end < lineText.length && WORD.test(lineText[end] ?? '')) end++;
    if (end === off && off < lineText.length) end = off + 1; // mark a single symbol
    return { from: start + off, to: start + end, severity: 'error', message: e.message };
  });
}
