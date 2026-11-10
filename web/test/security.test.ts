import { describe, expect, it } from 'vitest';

// Every viewer source file, as text.
const sources = import.meta.glob<string>('../src/*.ts', { eager: true, query: '?raw', import: 'default' });

// APIs that turn strings into markup or code. Player-supplied text (scripts,
// names) must only ever reach the DOM through textContent / createTextNode.
const SINKS = [/\.innerHTML\b/, /\.outerHTML\b/, /insertAdjacentHTML/, /document\.write/, /\beval\s*\(/, /\bnew\s+Function\s*\(/, /\bsetTimeout\s*\(\s*['"`]/, /\bsetInterval\s*\(\s*['"`]/];

describe('no HTML or code sinks', () => {
  it('scans the viewer sources', () => {
    expect(Object.keys(sources).length).toBeGreaterThanOrEqual(4);
  });
  for (const [path, text] of Object.entries(sources)) {
    it(path, () => {
      for (const re of SINKS) {
        expect(re.test(text), `${path} uses ${re}`).toBe(false);
      }
    });
  }
});
