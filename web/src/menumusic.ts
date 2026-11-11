// The menu music: loops on the lobby and on the spectator's waiting screen
// until a round opens (a new page, where the fight music takes over). Browsers
// block audio until the first click or key press, so it starts on the first
// gesture if it cannot start right away. Its sound toggle shares the replay
// player's sound setting.
import { el } from './dom';
import { loadSimpleSettings, saveSimpleSettings } from './spectate';
import menuMusicUrl from './audio/menu.mp3';

let menuAudio: HTMLAudioElement | null = null;

/** Starts the menu music (once per page) and returns its sound toggle. */
export function menuMusic(): HTMLElement {
  const settings = loadSimpleSettings();
  const audio = menuAudio ?? new Audio(menuMusicUrl);
  audio.loop = true;
  audio.volume = 0.4;
  menuAudio = audio;
  const tryPlay = () => {
    if (settings.sound && audio.paused) void audio.play().catch(() => {});
  };
  const gesture = () => {
    tryPlay();
    if (!audio.paused || !settings.sound) {
      document.removeEventListener('pointerdown', gesture);
      document.removeEventListener('keydown', gesture);
    }
  };
  document.addEventListener('pointerdown', gesture);
  document.addEventListener('keydown', gesture);
  tryPlay(); // works when the browser already allows audio on this site
  const box = el('input', { type: 'checkbox', 'aria-label': 'Play the menu music' });
  box.checked = settings.sound;
  box.addEventListener('change', () => {
    const latest = loadSimpleSettings();
    latest.sound = settings.sound = box.checked;
    saveSimpleSettings(latest);
    if (box.checked) tryPlay();
    else audio.pause();
  });
  return el('label', { class: 'menu-sound' }, box, ' sound');
}

/** Stops the menu music (when the page switches views without navigating). */
export function stopMenuMusic(): void {
  menuAudio?.pause();
  menuAudio = null;
}
