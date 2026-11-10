// The playback state shared by the DOM controls and the Phaser stage scene.
import { clampFrame, FRAMES_PER_SECOND, faster, slower, type Speed } from './replay';

export type Listener = (p: Playback) => void;

/** Length of the round intro ("ROUND n" then "FIGHT!") in wall-clock ms. */
export const INTRO_MS = 1500;
/** When the intro switches from the round word to "FIGHT!". */
export const INTRO_FIGHT_MS = 800;

export interface Intro {
  elapsedMs: number;
}

export class Playback {
  frame = 0;
  playing = false;
  speed: Speed = 1;
  /** Non-null while the round intro plays; the frame stays 0 meanwhile. */
  intro: Intro | null = null;
  private carry = 0; // fractional frames accumulated while playing
  private readonly listeners: Listener[] = [];

  constructor(readonly last: number) {}

  onChange(l: Listener): void {
    this.listeners.push(l);
  }

  private emit(): void {
    for (const l of this.listeners) l(this);
  }

  seek(f: number): void {
    const next = clampFrame(f, this.last);
    this.carry = 0;
    const endedIntro = this.intro !== null;
    this.intro = null;
    if (next === this.last && this.playing) this.playing = false;
    if (endedIntro || next !== this.frame || !this.playing) {
      this.frame = next;
      this.emit();
    }
  }

  step(n: number): void {
    this.playing = false;
    this.seek(this.frame + n);
  }

  play(): void {
    if (this.frame >= this.last) this.frame = 0; // replay from the start
    this.playing = true;
    this.carry = 0;
    this.intro = this.frame === 0 ? { elapsedMs: 0 } : null;
    this.emit();
  }

  pause(): void {
    this.intro = null;
    this.playing = false;
    this.carry = 0;
    this.emit();
  }

  toggle(): void {
    if (this.playing) this.pause();
    else this.play();
  }

  setSpeed(s: Speed): void {
    this.speed = s;
    this.emit();
  }

  slower(): void {
    this.setSpeed(slower(this.speed));
  }

  faster(): void {
    this.setSpeed(faster(this.speed));
  }

  /** Advances by real elapsed time (ms); stops and pauses on the last frame. */
  tick(deltaMs: number): void {
    if (!this.playing || deltaMs <= 0) return;
    if (this.intro) {
      // The intro runs in wall-clock time at any speed; leftover time plays frames.
      const left = INTRO_MS - this.intro.elapsedMs;
      if (deltaMs < left) {
        this.intro.elapsedMs += deltaMs;
        this.emit();
        return;
      }
      this.intro = null;
      deltaMs -= left;
      if (deltaMs <= 0) {
        this.emit();
        return;
      }
    }
    this.carry += (deltaMs / 1000) * FRAMES_PER_SECOND * this.speed;
    const whole = Math.floor(this.carry);
    if (whole === 0) return;
    this.carry -= whole;
    const next = Math.min(this.frame + whole, this.last);
    this.frame = next;
    if (next >= this.last) {
      this.playing = false;
      this.carry = 0;
    }
    this.emit();
  }
}
