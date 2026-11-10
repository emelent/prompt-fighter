// .pf syntax highlighting for CodeMirror: a small StreamLanguage tokenizer.
// The word lists mirror sim/dsl/vocab.go; tests check them against the
// player guide, whose own Go test keeps it in sync with the engine.
import { HighlightStyle, StreamLanguage, type StreamParser } from '@codemirror/language';
import { tags as t } from '@lezer/highlight';

export const ACTIONS = ['punch', 'kick', 'sweep', 'throw', 'jump', 'jump_forward', 'jump_back', 'duck', 'high_block', 'low_block', 'move_forward', 'move_back', 'wait'] as const;
export const SENSORS = ['distance', 'health', 'enemy.health', 'guard', 'passive', 'enemy.passive', 'repeats', 'enemy.airborne', 'enemy.ducking', 'enemy.blocking', 'enemy.attacking', 'enemy.knocked_down', 'in_range'] as const;
export const OVERLAY_ATTACKS = ['jump_kick', 'dive_punch', 'low_jab'] as const;
export const KEYWORDS = ['if', 'elif', 'else', 'loop', 'repeat', 'hold', 'and', 'or', 'not'] as const;
export const HANDLERS = ['on_hit', 'on_block'] as const;

export type PfToken = 'comment' | 'number' | 'keyword' | 'action' | 'sensor' | 'attack' | 'handler' | 'bracket' | 'operator' | 'unknown';

const sets: [PfToken, ReadonlySet<string>][] = [
  ['keyword', new Set(KEYWORDS)],
  ['action', new Set(ACTIONS)],
  ['sensor', new Set(SENSORS)],
  ['attack', new Set(OVERLAY_ATTACKS)],
  ['handler', new Set(HANDLERS)],
];

/** Classifies one identifier (possibly dotted). */
export function classifyWord(w: string): PfToken {
  if (/^x\d+$/.test(w)) return 'number';
  for (const [tok, set] of sets) if (set.has(w)) return tok;
  return 'unknown';
}

/** The stream-parser step: consumes one token and returns its type (null for whitespace). */
export function pfToken(stream: { eatSpace(): boolean; match(re: RegExp): RegExpMatchArray | null | boolean; skipToEnd(): void; next(): string | void; peek(): string | undefined }): PfToken | null {
  if (stream.eatSpace()) return null;
  if (stream.peek() === '#') {
    stream.skipToEnd();
    return 'comment';
  }
  if (stream.match(/^\d+/)) return 'number';
  const m = stream.match(/^[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)?/);
  if (m && typeof m !== 'boolean') return classifyWord(m[0]);
  if (stream.match(/^[{}()]/)) return 'bracket';
  if (stream.match(/^(?:<=|>=|==|!=|<|>|=|!|&&|\|\||;)/)) return 'operator';
  stream.next();
  return 'unknown';
}

const parser: StreamParser<null> = {
  name: 'pf',
  token: (stream) => pfToken(stream),
  tokenTable: {
    comment: t.lineComment,
    number: t.number,
    keyword: t.controlKeyword,
    action: t.function(t.variableName),
    sensor: t.propertyName,
    attack: t.atom,
    handler: t.definitionKeyword,
    bracket: t.bracket,
    operator: t.operator,
    unknown: t.invalid,
  },
  languageData: { commentTokens: { line: '#' } },
};

export const pfLanguage = StreamLanguage.define(parser);

export const pfHighlight = HighlightStyle.define([
  { tag: t.lineComment, color: '#6f7d8b', fontStyle: 'italic' },
  { tag: t.number, color: '#f2b134' },
  { tag: t.controlKeyword, color: '#c792ea', fontWeight: 'bold' },
  { tag: t.function(t.variableName), color: '#82aaff' },
  { tag: t.propertyName, color: '#7fdbca' },
  { tag: t.atom, color: '#f78c6c' },
  { tag: t.definitionKeyword, color: '#ff6b9b', fontWeight: 'bold' },
  { tag: t.operator, color: '#89ddff' },
  { tag: t.invalid, color: '#ff5c5c', textDecoration: 'underline wavy' },
]);
