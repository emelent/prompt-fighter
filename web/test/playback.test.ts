import { describe, expect, it } from 'vitest';
import { INTRO_MS, Playback } from '../src/playback';

describe('Playback', () => {
  it('plays 60 frames per second at 1x and scales with speed', () => {
    const p = new Playback(599);
    p.play();
    p.tick(INTRO_MS); // the round intro comes first
    p.tick(1000);
    expect(p.frame).toBe(60);
    p.setSpeed(0.25);
    p.tick(1000);
    expect(p.frame).toBe(75);
    p.setSpeed(2);
    p.tick(500);
    expect(p.frame).toBe(135);
  });
  it('accumulates fractional frames', () => {
    const p = new Playback(599);
    p.seek(1);
    p.play();
    for (let i = 0; i < 10; i++) p.tick(1000 / 120); // half a frame each
    expect(p.frame).toBe(6);
  });
  it('stops and pauses on the last frame', () => {
    const p = new Playback(300);
    p.seek(290);
    p.play();
    p.tick(10_000);
    expect(p.frame).toBe(300);
    expect(p.playing).toBe(false);
    p.play(); // restarts from 0 at the end
    expect(p.frame).toBe(0);
  });
  it('steps and seeks with clamping, pausing playback', () => {
    const p = new Playback(300);
    p.play();
    p.step(3);
    expect(p.playing).toBe(false);
    expect(p.frame).toBe(3);
    p.step(-10);
    expect(p.frame).toBe(0);
    p.seek(999);
    expect(p.frame).toBe(300);
  });
  it('notifies listeners and steps speeds', () => {
    const p = new Playback(10);
    const seen: number[] = [];
    p.onChange((x) => seen.push(x.frame));
    p.seek(4);
    p.faster();
    expect(p.speed).toBe(2);
    p.slower();
    p.slower();
    expect(p.speed).toBe(0.5);
    expect(seen[0]).toBe(4);
  });

  describe('round intro', () => {
    it('holds frame 0 for the intro at any speed, then plays', () => {
      for (const speed of [1, 0.25] as const) {
        const p = new Playback(599);
        p.setSpeed(speed);
        p.play();
        expect(p.intro).toEqual({ elapsedMs: 0 });
        p.tick(INTRO_MS - 100);
        expect(p.frame).toBe(0);
        expect(p.intro?.elapsedMs).toBe(INTRO_MS - 100);
        p.tick(100 + 1000);
        expect(p.intro).toBeNull();
        expect(p.frame).toBe(60 * speed);
      }
    });
    it('plays again when replaying from the end', () => {
      const p = new Playback(300);
      p.seek(300);
      p.play();
      expect(p.frame).toBe(0);
      expect(p.intro).not.toBeNull();
    });
    it('ends on stepping, seeking or pausing', () => {
      const p = new Playback(300);
      const seen: (number | null)[] = [];
      p.onChange((x) => seen.push(x.intro ? -1 : x.frame));
      p.play();
      p.tick(200);
      p.step(1);
      expect(p.intro).toBeNull();
      expect(p.playing).toBe(false);
      expect(p.frame).toBe(1);
      p.seek(0);
      p.play();
      p.seek(0); // seeking to the same frame still ends it, and notifies
      expect(p.intro).toBeNull();
      expect(seen.at(-1)).toBe(0);
      p.play();
      p.pause();
      expect(p.intro).toBeNull();
    });
    it('does not play when starting from a later frame', () => {
      const p = new Playback(300);
      p.seek(100);
      p.play();
      expect(p.intro).toBeNull();
      p.tick(1000);
      expect(p.frame).toBe(160);
    });
  });
});
