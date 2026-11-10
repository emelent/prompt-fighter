// The stage backdrop: a rooftop at dusk, generated as a 320×180 RGBA image
// and drawn at 3× (960×540). Pure and seeded, so it is identical every load.
import { hexRGB } from './fighterart';

export const BACKDROP_W = 320;
export const BACKDROP_H = 180;
/** Backdrop row where the fighters' feet stand. */
export const BACKDROP_GROUND = 162;

class Raster {
  readonly data: Uint8ClampedArray;
  constructor(readonly w: number, readonly h: number) {
    this.data = new Uint8ClampedArray(w * h * 4);
  }
  fill(x: number, y: number, w: number, h: number, hex: string): void {
    const [r, g, b] = hexRGB(hex);
    const x0 = Math.max(0, Math.floor(x));
    const y0 = Math.max(0, Math.floor(y));
    const x1 = Math.min(this.w, Math.floor(x + w));
    const y1 = Math.min(this.h, Math.floor(y + h));
    for (let yy = y0; yy < y1; yy++)
      for (let xx = x0; xx < x1; xx++) {
        const i = (yy * this.w + xx) * 4;
        this.data[i] = r;
        this.data[i + 1] = g;
        this.data[i + 2] = b;
        this.data[i + 3] = 255;
      }
  }
}

function mulberry(seed: number): () => number {
  return () => {
    seed = (seed + 0x6d2b79f5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

const SKY = ['#1c1430', '#2a1a40', '#45204e', '#6b2a52', '#a23f52', '#d8664a', '#f29a4a'];
const FLOOR_TOP = 150;

/** Generates the backdrop. Every pixel is opaque. */
export function backdropRGBA(): { width: number; height: number; data: Uint8ClampedArray } {
  const R = new Raster(BACKDROP_W, BACKDROP_H);
  const W = BACKDROP_W;
  // Sky: the whole image first (so nothing is left transparent), then hard
  // bands with a dithered seam between each pair.
  R.fill(0, 0, W, BACKDROP_H, SKY[SKY.length - 1]!);
  const bh = 17;
  SKY.forEach((c, i) => R.fill(0, i * bh, W, bh, c));
  for (let i = 1; i < SKY.length; i++)
    for (let y = i * bh - 3; y < i * bh; y++) for (let x = y % 2; x < W; x += 2) R.fill(x, y, 1, 1, SKY[i]!);
  // Sun, with horizon stripes.
  for (let dy = -12; dy <= 12; dy++) {
    const w = Math.round(Math.sqrt(144 - dy * dy));
    R.fill(242 - w, 100 + dy, w * 2, 1, '#ffd36a');
  }
  for (let dy = 4; dy < 12; dy += 3) R.fill(226, 100 + dy, 32, 1, '#f29a4a');
  // Two skyline layers; the near one has lit windows.
  const rnd = mulberry(7);
  const skyline = (base: number, col: string, lit: boolean, minW: number, maxW: number, minH: number, maxH: number): void => {
    let px = -4;
    while (px < W) {
      const w = Math.floor(minW + rnd() * (maxW - minW));
      const h = Math.floor(minH + rnd() * (maxH - minH));
      R.fill(px, base - h, w, h, col);
      if (lit)
        for (let wy = base - h + 4; wy < base - 4; wy += 5)
          for (let wx = px + 2; wx < px + w - 2; wx += 4) if (rnd() < 0.28) R.fill(wx, wy, 2, 2, rnd() < 0.5 ? '#ffd36a' : '#f6a85a');
      px += w + Math.floor(rnd() * 3);
    }
  };
  skyline(126, '#3a2246', false, 12, 26, 26, 60);
  skyline(136, '#221630', true, 15, 34, 20, 52);
  // Water tower on the roof.
  R.fill(40, 104, 22, 18, '#120c18');
  R.fill(37, 100, 28, 5, '#120c18');
  R.fill(42, 122, 3, 18, '#120c18');
  R.fill(57, 122, 3, 18, '#120c18');
  R.fill(47, 95, 8, 5, '#120c18');
  for (let y = 107; y < 120; y += 4) R.fill(42, y, 18, 1, '#2a1d36');
  // Parapet wall, then the railing in front of it.
  R.fill(0, 140, W, 10, '#1a1226');
  R.fill(0, 140, W, 1, '#2a1d36');
  R.fill(0, 136, W, 2, '#120c18');
  for (let x = 4; x < W; x += 10) R.fill(x, 136, 2, 14, '#120c18');
  // Rooftop floor with perspective seams.
  R.fill(0, FLOOR_TOP, W, BACKDROP_H - FLOOR_TOP, '#3b3046');
  R.fill(0, FLOOR_TOP, W, 3, '#4a3d57');
  const top = FLOOR_TOP + 3;
  for (let i = -12; i <= 12; i++) {
    const a = W / 2 + i * 18;
    const b = W / 2 + i * 36;
    for (let y = top; y < BACKDROP_H; y++) R.fill(Math.round(a + ((b - a) * (y - top)) / (BACKDROP_H - top)), y, 1, 1, '#2c2336');
  }
  R.fill(0, 166, W, 1, '#2c2336');
  R.fill(0, 174, W, 1, '#2c2336');
  return { width: R.w, height: R.h, data: R.data };
}
