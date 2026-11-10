import { describe, expect, it } from 'vitest';
import { Playback } from '../src/playback';

describe('Playback', () => {
  it('plays 60 frames per second at 1x and scales with speed', () => {
    const p = new Playback(599);
    p.play();
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
    p.play();
    for (let i = 0; i < 10; i++) p.tick(1000 / 120); // half a frame each
    expect(p.frame).toBe(5);
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
});
