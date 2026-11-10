// The playback state shared by the DOM controls and the Phaser stage scene.
import { clampFrame, FRAMES_PER_SECOND, faster, slower, type Speed } from './replay';

export type Listener = (p: Playback) => void;

export class Playback {
  frame = 0;
  playing = false;
  speed: Speed = 1;
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
    if (next === this.last && this.playing) this.playing = false;
    if (next !== this.frame || !this.playing) {
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
    this.emit();
  }

  pause(): void {
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
