import { describe, expect, it } from 'vitest';
import { BACKDROP_H, BACKDROP_W, backdropRGBA } from '../src/backdrop';
import { ADVANCE, ALL_GLYPHS, fitText, GLYPH_H, glyphFor, MISSING_GLYPH, textMask, textWidth } from '../src/pixelfont';

describe('pixel font', () => {
  it('measures strings', () => {
    expect(textWidth('')).toBe(0);
    expect(textWidth('K.O.')).toBe(4 * ADVANCE - 1);
    const m = textMask('BLOCKED');
    expect(m.width).toBe(textWidth('BLOCKED') + 2);
    expect(m.height).toBe(GLYPH_H + 2);
  });
  it('defines every glyph as exactly 5×7', () => {
    for (const [ch, g] of Object.entries(ALL_GLYPHS)) expect(g.length, ch).toBe(35);
    expect(MISSING_GLYPH.length).toBe(35);
  });
  it('has every letter, digit and the callout punctuation', () => {
    for (const ch of 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.!-:') expect(glyphFor(ch), ch).not.toBe(MISSING_GLYPH);
    expect(glyphFor('k')).toBe(glyphFor('K'));
  });
  it('draws unknown characters as a box without throwing', () => {
    expect(glyphFor('é')).toBe(MISSING_GLYPH);
    expect(glyphFor('🙂')).toBe(MISSING_GLYPH);
    expect(textMask('🙂').width).toBe(textWidth('x') + 2);
  });
  it('draws markup as plain glyphs', () => {
    const m = textMask('<img src=x>');
    expect(m.width).toBe(textWidth('<img src=x>') + 2);
    expect([...m.bits].filter((b) => b === 1).length).toBeGreaterThan(40);
  });
  it('outlines the ink', () => {
    const m = textMask('A');
    for (let y = 0; y < m.height; y++)
      for (let x = 0; x < m.width; x++) {
        if (m.bits[y * m.width + x] !== 1) continue;
        for (const [dx, dy] of [[1, 0], [-1, 0], [0, 1], [0, -1]] as const) expect(m.bits[(y + dy) * m.width + x + dx]).not.toBe(0);
      }
  });
  it('truncates long names', () => {
    expect(fitText('short', 10)).toBe('short');
    expect(fitText('a very long fighter name', 10)).toBe('a very lo.');
  });
});

describe('backdrop', () => {
  it('is 320×180 and fully opaque', () => {
    const b = backdropRGBA();
    expect([b.width, b.height]).toEqual([BACKDROP_W, BACKDROP_H]);
    for (let i = 3; i < b.data.length; i += 4) if (b.data[i] !== 255) throw new Error(`transparent pixel ${(i - 3) / 4}`);
  });
  it('is the same every time', () => {
    expect(backdropRGBA().data).toEqual(backdropRGBA().data);
  });
});
