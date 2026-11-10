// A 5×7 pixel font for every string drawn on the stage. textMask() renders a
// string at 1 pixel per font pixel with a one-pixel outline; the stage scales
// the result up with nearest-neighbour sampling. Text is only ever looked up
// glyph by glyph, so player names cannot inject anything.

export const GLYPH_W = 5;
export const GLYPH_H = 7;
/** Horizontal distance between glyph origins, in font pixels. */
export const ADVANCE = GLYPH_W + 1;

const rows = (...r: string[]): string => r.join('');

/** Glyphs as 35 characters, row by row; '#' is ink. */
const GLYPHS: Record<string, string> = {
  A: rows('.###.', '#...#', '#...#', '#####', '#...#', '#...#', '#...#'),
  B: '####.#...##...#####.#...##...#####.', C: '.###.#...##....#....#....#...#.###.',
  D: '####.#...##...##...##...##...#####.', E: '######....#....####.#....#....#####', F: '######....#....####.#....#....#....',
  G: '.###.#...##....#.####...##...#.###.', H: '#...##...##...#######...##...##...#', I: '#####..#....#....#....#....#..#####',
  J: rows('..###', '...#.', '...#.', '...#.', '#..#.', '#..#.', '.##..'),
  K: '#...##..#.#.#..##...#.#..#..#.#...#', L: '#....#....#....#....#....#....#####', M: '#...###.###.#.##.#.##...##...##...#',
  N: '#...###..##.#.##..###...##...##...#', O: '.###.#...##...##...##...##...#.###.', P: '####.#...##...#####.#....#....#....',
  Q: rows('.###.', '#...#', '#...#', '#...#', '#.#.#', '#..#.', '.##.#'),
  R: '####.#...##...#####.#.#..#..#.#...#', S: '.#####....#.....###.....#....#####.', T: '#####..#....#....#....#....#....#..',
  U: '#...##...##...##...##...##...#.###.',
  V: rows('#...#', '#...#', '#...#', '#...#', '#...#', '.#.#.', '..#..'),
  W: '#...##...##...##.#.##.#.###.###...#',
  X: rows('#...#', '#...#', '.#.#.', '..#..', '.#.#.', '#...#', '#...#'),
  Y: '#...##...#.#.#...#....#....#....#..',
  Z: rows('#####', '....#', '...#.', '..#..', '.#...', '#....', '#####'),
  0: '.###.#...##..###.#.###..##...#.###.', 1: '..#...##....#....#....#....#...###.', 2: '.###.#...#....#...#...#...#...#####',
  3: '####.....#....#.###.....#....#####.', 4: '...#...##..#.#.#..#.#####...#....#.', 5: '######....####.....#....##...#.###.',
  6: '.###.#....#....####.#...##...#.###.', 7: '#####....#...#...#...#....#....#...', 8: '.###.#...##...#.###.#...##...#.###.',
  9: '.###.#...##...#.####....#....#.###.',
  ' ': '.'.repeat(35),
  '.': rows('.....', '.....', '.....', '.....', '.....', '.##..', '.##..'),
  ',': rows('.....', '.....', '.....', '.....', '..#..', '..#..', '.#...'),
  '!': rows('..#..', '..#..', '..#..', '..#..', '..#..', '.....', '..#..'),
  '?': rows('.###.', '#...#', '....#', '...#.', '..#..', '.....', '..#..'),
  ':': rows('.....', '..#..', '..#..', '.....', '..#..', '..#..', '.....'),
  ';': rows('.....', '..#..', '..#..', '.....', '..#..', '..#..', '.#...'),
  "'": rows('..#..', '..#..', '.#...', '.....', '.....', '.....', '.....'),
  '"': rows('.#.#.', '.#.#.', '.....', '.....', '.....', '.....', '.....'),
  '-': rows('.....', '.....', '.....', '#####', '.....', '.....', '.....'),
  '_': rows('.....', '.....', '.....', '.....', '.....', '.....', '#####'),
  '+': rows('.....', '..#..', '..#..', '#####', '..#..', '..#..', '.....'),
  '=': rows('.....', '.....', '#####', '.....', '#####', '.....', '.....'),
  '*': rows('.....', '#.#.#', '.###.', '#####', '.###.', '#.#.#', '.....'),
  '/': rows('....#', '....#', '...#.', '..#..', '.#...', '#....', '#....'),
  '(': rows('...#.', '..#..', '.#...', '.#...', '.#...', '..#..', '...#.'),
  ')': rows('.#...', '..#..', '...#.', '...#.', '...#.', '..#..', '.#...'),
  '<': rows('...#.', '..#..', '.#...', '#....', '.#...', '..#..', '...#.'),
  '>': rows('.#...', '..#..', '...#.', '....#', '...#.', '..#..', '.#...'),
  '#': rows('.#.#.', '.#.#.', '#####', '.#.#.', '#####', '.#.#.', '.#.#.'),
  '&': rows('.##..', '#..#.', '#.#..', '.#...', '#.#.#', '#..#.', '.##.#'),
  '%': rows('##..#', '##..#', '...#.', '..#..', '.#...', '#..##', '#..##'),
  '·': rows('.....', '.....', '.....', '..#..', '.....', '.....', '.....'),
  '×': rows('.....', '#...#', '.#.#.', '..#..', '.#.#.', '#...#', '.....'),
};

/** Every glyph, for tests. */
export const ALL_GLYPHS: Readonly<Record<string, string>> = GLYPHS;

/** Drawn for characters the font does not have. */
export const MISSING_GLYPH = rows('#####', '#...#', '#...#', '#...#', '#...#', '#...#', '#####');

/** The glyph for one character: lower case is drawn as capitals; unknown characters get a box. */
export function glyphFor(ch: string): string {
  return GLYPHS[ch] ?? GLYPHS[ch.toUpperCase()] ?? MISSING_GLYPH;
}

/** Width in font pixels of text, without the outline. */
export function textWidth(text: string): number {
  const n = [...text].length;
  return n === 0 ? 0 : n * ADVANCE - 1;
}

export interface TextMask {
  width: number;
  height: number;
  /** 0 empty, 1 ink, 2 outline. */
  bits: Uint8Array;
}

/** Renders text with a one-pixel outline (8-neighbour), so it reads on any background. */
export function textMask(text: string): TextMask {
  const chars = [...text];
  const width = textWidth(text) + 2;
  const height = GLYPH_H + 2;
  const bits = new Uint8Array(Math.max(width, 0) * height);
  chars.forEach((ch, i) => {
    const g = glyphFor(ch);
    for (let r = 0; r < GLYPH_H; r++)
      for (let c = 0; c < GLYPH_W; c++) if (g[r * GLYPH_W + c] === '#') bits[(r + 1) * width + 1 + i * ADVANCE + c] = 1;
  });
  for (let y = 0; y < height; y++)
    for (let x = 0; x < width; x++) {
      if (bits[y * width + x] !== 0) continue;
      outer: for (let dy = -1; dy <= 1; dy++)
        for (let dx = -1; dx <= 1; dx++) {
          const xx = x + dx;
          const yy = y + dy;
          if (xx >= 0 && yy >= 0 && xx < width && yy < height && bits[yy * width + xx] === 1) {
            bits[y * width + x] = 2;
            break outer;
          }
        }
    }
  return { width, height, bits };
}

/** Truncates text to at most maxChars characters, ending in "." when cut. */
export function fitText(text: string, maxChars: number): string {
  const chars = [...text];
  return chars.length <= maxChars ? text : chars.slice(0, Math.max(0, maxChars - 1)).join('') + '.';
}
