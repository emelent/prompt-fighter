// Viewer entry point: routes by location.pathname and wires the DOM.
// All player-supplied text is inserted with textContent only (see el()).
import './style.css';
import { matchRows, newRoundToOpen, POLL_BACKOFF_MS, POLL_MS } from './match';
import { Playback } from './playback';
import { eventLines, frameHash, lastFrame, parseFrameHash, parseLog, SPEEDS, type Speed, validateLog } from './replay';
import { roundLabel } from './callouts';
import { COUNTDOWN_SECONDS, Countdown, endButton, loadSimpleSettings, saveSimpleSettings, type EndButton } from './spectate';
import { buildScene } from './scene';
import { mountStage } from './stage';
import type { GameEvent, Log, MatchView } from './types';
import { alertBox, el, fetchJSON, show } from './dom';
import { lobbyPage, playPage } from './play';
import { watchPage } from './watch';

// --- round player -----------------------------------------------------------

interface PlayerContext {
  heading: string;
  matchLink?: string;
  backLabel?: string;
  /** Start playing on load unless a #f= frame link is present. */
  autoplay?: boolean;
  /** The stored round number, for the intro ("ROUND n"); absent for local files. */
  round?: number;
  /**
   * full: the analysis viewer. simple: a player's own round, opened from the
   * play page. spectator: a watch-page round. The simple modes hide everything
   * but play/pause and rewind behind a settings menu.
   */
  mode?: 'full' | 'simple' | 'spectator';
  /** The match, for the spectator's end-of-replay button. */
  match?: MatchView;
}

/** One part of the simple player that the settings menu can show or hide. */
type Part = 'scripts' | 'feed' | 'timeline' | 'speed' | 'toggles';

const PART_LABELS: Record<Part | 'sound', string> = {
  scripts: 'Scripts',
  feed: 'Event feed',
  timeline: 'Timeline and frame steps',
  speed: 'Playback speed',
  toggles: 'Checks and ranges',
  sound: 'Sound',
};

const SOUND_PREF = 'pf.sound';

/** Whether sound is on: on unless the viewer turned it off before. */
function loadSoundPref(): boolean {
  try {
    return localStorage.getItem(SOUND_PREF) !== 'off';
  } catch {
    return true;
  }
}

function saveSoundPref(on: boolean): void {
  try {
    localStorage.setItem(SOUND_PREF, on ? 'on' : 'off');
  } catch {
    // Storage blocked (private window): the choice lasts for this page only.
  }
}

/** Scrolls box (not the page) the least amount that shows item. */
function scrollWithin(box: HTMLElement, item: HTMLElement): void {
  const b = box.getBoundingClientRect();
  const r = item.getBoundingClientRect();
  if (r.top < b.top) box.scrollTop -= b.top - r.top;
  else if (r.bottom > b.bottom) box.scrollTop += r.bottom - b.bottom;
}

function isTextInput(t: EventTarget | null): boolean {
  if (!(t instanceof HTMLElement)) return false;
  if (t instanceof HTMLTextAreaElement || t instanceof HTMLSelectElement) return true;
  return t instanceof HTMLInputElement && !['range', 'checkbox', 'button'].includes(t.type);
}

function renderPlayer(log: Log, ctx: PlayerContext): void {
  // Read the frame link before anything rewrites the hash.
  const start = parseFrameHash(location.hash);
  const last = lastFrame(log);
  const pb = new Playback(last);
  const mode = ctx.mode ?? 'full';
  const simple = mode !== 'full';
  const settings = simple ? loadSimpleSettings() : null;
  let showChecks = false;
  let showRanges = false;

  // Header
  const header = el('div', { class: 'player-head' }, el('h1', {}, ctx.heading), ctx.matchLink ? el('a', { href: ctx.matchLink }, ctx.backLabel ?? '← back to match') : null);

  // Stage
  const stageBox = el('div', { class: 'stage', 'aria-label': 'Replay stage' });
  // Full screen shows just the stage; Esc (or the button again) leaves it.
  const toggleFullscreen = () => {
    if (document.fullscreenElement) void document.exitFullscreen();
    else void stageBox.requestFullscreen?.().catch(() => undefined);
  };
  const fsBtn = el('button', { type: 'button', class: 'stage-fs', 'aria-label': 'Full screen (F)', title: 'Full screen (F)' }, '⛶');
  fsBtn.addEventListener('click', toggleFullscreen);
  stageBox.append(fsBtn);

  // Controls
  const btn = (label: string, aria: string, onClick: () => void) => {
    const b = el('button', { type: 'button', 'aria-label': aria, title: aria }, label);
    b.addEventListener('click', onClick);
    return b;
  };
  const playBtn = btn('▶', 'Play', () => pb.toggle());
  const scrub = el('input', { type: 'range', min: '0', max: String(last), value: '0', step: '1', 'aria-label': 'Frame' });
  scrub.addEventListener('input', () => pb.seek(Number(scrub.value)));
  const speed = el('select', { 'aria-label': 'Playback speed' }, ...SPEEDS.map((s) => el('option', { value: String(s) }, `${s}×`)));
  speed.value = '1';
  speed.addEventListener('change', () => pb.setSpeed(Number(speed.value) as Speed));
  if (settings) pb.setSpeed(settings.playbackSpeed);
  const frameOut = el('output', { class: 'frame-out', 'aria-label': 'Current frame' });
  const checksBox = el('input', { type: 'checkbox', 'aria-label': 'Show sensor checks in the event feed' });
  const rangesBox = el('input', { type: 'checkbox', 'aria-label': 'Show attack ranges' });
  const soundBox = el('input', { type: 'checkbox', 'aria-label': 'Play sound effects and music' });
  soundBox.checked = loadSoundPref();
  soundBox.addEventListener('change', () => saveSoundPref(soundBox.checked));
  // The simple modes group the controls so the settings menu can show or hide
  // each group; the full mode keeps its single row.
  const groups: Record<Part, HTMLElement | null> = { scripts: null, feed: null, timeline: null, speed: null, toggles: null };
  let settingsBox: HTMLElement | null = null;
  let controls: HTMLElement;
  if (!simple) {
    controls = el(
      'div',
      { class: 'controls', role: 'toolbar', 'aria-label': 'Playback controls' },
      btn('⏮', 'First frame', () => pb.step(-last)),
      btn('−10', 'Back 10 frames', () => pb.step(-10)),
      btn('−1', 'Back 1 frame', () => pb.step(-1)),
      playBtn,
      btn('+1', 'Forward 1 frame', () => pb.step(1)),
      btn('+10', 'Forward 10 frames', () => pb.step(10)),
      btn('⏭', 'Last frame', () => pb.step(last)),
      scrub,
      frameOut,
      el('label', {}, 'speed ', speed),
      el('label', {}, checksBox, ' checks'),
      el('label', {}, rangesBox, ' ranges'),
      el('label', {}, soundBox, ' sound'),
    );
  } else {
    // Rewind returns to the start and keeps playing if it was playing.
    const rewind = btn('⏮', 'Rewind to the start', () => {
      const wasPlaying = pb.playing;
      pb.seek(0);
      if (wasPlaying && !pb.playing) pb.play();
    });
    groups.timeline = el(
      'span',
      { class: 'ctl-group ctl-timeline' },
      btn('−10', 'Back 10 frames', () => pb.step(-10)),
      btn('−1', 'Back 1 frame', () => pb.step(-1)),
      btn('+1', 'Forward 1 frame', () => pb.step(1)),
      btn('+10', 'Forward 10 frames', () => pb.step(10)),
      btn('⏭', 'Last frame', () => pb.step(last)),
      scrub,
      frameOut,
    );
    groups.speed = el('label', { class: 'ctl-group' }, 'speed ', speed);
    groups.toggles = el('span', { class: 'ctl-group' }, el('label', {}, checksBox, ' checks'), el('label', {}, rangesBox, ' ranges'));
    const gear = el('button', { type: 'button', class: 'gear', 'aria-label': 'Settings', title: 'Settings', 'aria-expanded': 'false', 'aria-controls': 'player-settings' }, '⚙');
    const menu = el('div', { class: 'settings-pop', id: 'player-settings', role: 'dialog', 'aria-label': 'Replay settings', hidden: 'hidden' });
    for (const part of [...(Object.keys(groups) as Part[]), 'sound'] as const) {
      const box = el('input', { type: 'checkbox' });
      box.checked = settings![part];
      box.addEventListener('change', () => {
        settings![part] = box.checked;
        saveSimpleSettings(settings!);
        applyParts();
      });
      menu.append(el('label', {}, box, ` ${PART_LABELS[part]}`));
    }
    const close = () => {
      menu.hidden = true;
      gear.setAttribute('aria-expanded', 'false');
    };
    gear.addEventListener('click', () => {
      menu.hidden = !menu.hidden;
      gear.setAttribute('aria-expanded', String(!menu.hidden));
      if (!menu.hidden) menu.querySelector('input')?.focus();
    });
    menu.addEventListener('keydown', (k) => {
      if (k.key === 'Escape') {
        close();
        gear.focus();
      }
    });
    document.addEventListener('pointerdown', (e) => {
      if (!menu.hidden && e.target instanceof Node && !menu.contains(e.target) && !gear.contains(e.target)) close();
    });
    settingsBox = el('span', { class: 'settings' }, gear, menu);
    controls = el(
      'div',
      { class: 'controls simple', role: 'toolbar', 'aria-label': 'Playback controls' },
      el('span', { class: 'ctl-group ctl-core' }, rewind, playBtn),
      groups.timeline,
      groups.speed,
      groups.toggles,
      settingsBox,
    );
  }
  const soundOn = () => (settings ? settings.sound : soundBox.checked);

  // Script panels
  const panels = log.players.map((p, i) => {
    const lines = p.script.split('\n');
    const items = lines.map((text, n) => el('li', { 'data-line': String(n + 1) }, el('span', { class: 'ln' }, String(n + 1)), el('code', {}, text === '' ? ' ' : text)));
    const list = el('ol', { class: 'script' }, ...items);
    const status = el('div', { class: 'script-status' });
    const panel = el('section', { class: `panel p${i + 1}`, 'aria-label': `${p.id} script` }, el('h2', {}, `${p.id} — `, el('span', {}, p.name)), status, list);
    return { panel, items, status, current: null as HTMLElement | null };
  });

  // Event feed
  const feedList = el('ol', { class: 'feed', 'aria-label': 'Events' });
  let feedItems: { e: GameEvent; li: HTMLLIElement }[] = [];
  const buildFeed = () => {
    const events = log.events.filter((e) => showChecks || e.type !== 'check');
    const lines = eventLines(log, showChecks);
    feedItems = events.map((e, i) => {
      const li = el('li', { tabindex: '0', 'data-frame': String(e.frame) }, lines[i] ?? '');
      const jump = () => pb.step(e.frame - pb.frame);
      li.addEventListener('click', jump);
      li.addEventListener('keydown', (k) => {
        if (k.key === 'Enter') jump();
      });
      return { e, li };
    });
    feedList.replaceChildren(...feedItems.map((x) => x.li));
    if (feedItems.length === 0) {
      feedList.append(el('li', { class: 'empty' }, showChecks ? 'No events in this round.' : 'No events besides sensor checks; tick "checks" to show them.'));
    }
    lastCurrent = null;
    sync();
  };
  checksBox.addEventListener('change', () => {
    showChecks = checksBox.checked;
    buildFeed();
  });

  let lastCurrent: HTMLElement | null = null;
  let stageScene: { invalidate(): void } | null = null;
  rangesBox.addEventListener('change', () => {
    showRanges = rangesBox.checked;
    stageScene?.invalidate();
  });

  const sync = () => {
    const f = pb.frame;
    playBtn.textContent = pb.playing ? '⏸' : '▶';
    playBtn.setAttribute('aria-label', pb.playing ? 'Pause' : 'Play');
    scrub.value = String(f);
    frameOut.textContent = `f ${f} / ${last}`;
    if (Number(speed.value) !== pb.speed) speed.value = String(pb.speed);
    if (settings && pb.speed !== settings.playbackSpeed) {
      settings.playbackSpeed = pb.speed;
      saveSimpleSettings(settings);
    }
    updateEnd();
    // Script highlighting
    const sc = buildScene(log, f);
    sc.fighters.forEach((fd, i) => {
      const p = panels[i]!;
      p.current?.classList.remove('current');
      p.current = fd.scriptLine > 0 ? (p.items[fd.scriptLine - 1] ?? null) : null;
      p.current?.classList.add('current');
      p.status.textContent = fd.scriptLine > 0 ? `line ${fd.scriptLine} · ${fd.state}` : `no line executing · ${fd.state}`;
    });
    // Feed: mark past events and keep the latest in view.
    let current: HTMLElement | null = null;
    for (const { e, li } of feedItems) {
      const past = e.frame <= f;
      li.classList.toggle('past', past);
      if (past) current = li;
    }
    if (current !== lastCurrent) {
      lastCurrent?.classList.remove('current');
      current?.classList.add('current');
      // Scroll only the feed box: scrollIntoView would also scroll the page
      // and pull the stage out of view.
      if (current) scrollWithin(feedList, current);
      lastCurrent = current;
    }
    if (!pb.playing && ready) {
      history.replaceState(null, '', location.pathname + location.search + frameHash(f));
    }
  };
  let ready = false; // no hash writes until the initial frame is applied
  pb.onChange(sync);

  const feedPanel = el('section', { class: 'panel events', 'aria-label': 'Event feed' }, el('h2', {}, 'Events'), feedList);
  const keysHint = el('p', { class: 'hint' }, 'Keys: Space play/pause · ←/→ 1 frame · Shift+←/→ 10 frames · Home/End · [ ] speed · F full screen');
  const endBar = el('div', { class: 'end-bar', role: 'status', 'aria-live': 'polite', hidden: 'hidden' });
  // The hint keeps its space when it fades out, so the first click (which
  // unlocks audio and hides it) does not shift the controls under the pointer.
  const soundHint = el('p', { class: 'hint sound-hint quiet' }, 'Click or press a key for sound.');
  const layout = el(
    'div',
    { class: simple ? 'player simple' : 'player' },
    header,
    stageBox,
    endBar,
    soundHint,
    controls,
    el('div', { class: simple ? 'panels simple' : 'panels' }, panels[0]!.panel, panels[1]!.panel, feedPanel),
    keysHint,
  );
  // Show or hide the simple player's optional parts from the settings.
  const applyParts = () => {
    if (!settings) return;
    for (const p of panels) p.panel.hidden = !settings.scripts;
    feedPanel.hidden = !settings.feed;
    for (const part of ['timeline', 'speed', 'toggles'] as const) groups[part]!.hidden = !settings[part];
    keysHint.hidden = !settings.timeline;
    soundHint.classList.toggle('quiet', !settings.sound || audioUnlocked);
  };
  let audioUnlocked = false;
  const unlock = () => {
    audioUnlocked = true;
    soundHint.classList.add('quiet');
  };
  document.addEventListener('pointerdown', unlock, { once: true });
  document.addEventListener('keydown', unlock, { once: true });

  // End of the replay (simple modes): Done, or for spectators Next round with
  // a countdown once the next round has been played. Polls while it waits.
  const countdown = new Countdown();
  let endKey = '';
  let matchView: MatchView | null = ctx.match ?? null;
  let poll: ReturnType<typeof setInterval> | null = null;
  const stopPoll = () => {
    if (poll !== null) clearInterval(poll);
    poll = null;
  };
  const startPoll = () => {
    if (poll !== null || !matchView) return;
    const id = matchView.match_id;
    poll = setInterval(() => {
      void fetchJSON<MatchView>(`/matches/${encodeURIComponent(id)}`).then((r) => {
        if (!r.ok) return;
        matchView = r.data;
        updateEnd();
      });
    }, POLL_MS);
  };
  const updateEnd = () => {
    if (!simple) return;
    const atEnd = pb.frame === last && !pb.playing && pb.intro === null;
    if (!atEnd) {
      if (endKey) {
        endKey = '';
        countdown.cancel();
        stopPoll();
        endBar.hidden = true;
        endBar.replaceChildren();
      }
      return;
    }
    const doneHref = ctx.matchLink ?? '/play';
    const spec: EndButton = mode === 'spectator' && matchView && ctx.round ? endButton(matchView, ctx.round) : { kind: 'done', href: doneHref };
    const key = `${spec.kind} ${spec.href}`;
    if (key === endKey) return;
    endKey = key;
    countdown.cancel();
    const go = el('button', { type: 'button', class: 'end-btn' }, spec.kind === 'next' ? 'Next round' : 'Done');
    go.addEventListener('click', () => location.assign(spec.href));
    const note = el('span', { class: 'countdown' });
    endBar.replaceChildren(go, note);
    endBar.hidden = false;
    if (spec.kind === 'next') {
      stopPoll();
      countdown.start(COUNTDOWN_SECONDS, (left) => (note.textContent = `Round ${spec.round} starting in ${left}`), () => location.assign(spec.href));
    } else if (mode === 'spectator' && matchView?.status !== 'finished') {
      startPoll();
    }
  };
  show(layout);
  applyParts();
  buildFeed();

  const { scene } = mountStage(stageBox, { log, playback: pb, showRanges: () => showRanges, roundLabel: roundLabel(ctx.round ?? null), soundOn });
  stageScene = scene;

  pb.seek(start ?? 0);
  // Start auto-play before enabling frame-link writes, so the URL does not
  // pick up "#f=0" (which would suppress auto-play on reload).
  if (ctx.autoplay && start === null) pb.play();
  ready = true;
  sync();
  // A frame link pasted into an already open tab only changes the hash.
  window.addEventListener('hashchange', () => {
    const f = parseFrameHash(location.hash);
    if (f !== null && f !== pb.frame) pb.step(f - pb.frame);
  });

  document.addEventListener('keydown', (k) => {
    if (isTextInput(k.target) || k.altKey || k.ctrlKey || k.metaKey) return;
    const n = k.shiftKey ? 10 : 1;
    switch (k.key) {
      case ' ':
        pb.toggle();
        break;
      case 'ArrowLeft':
        pb.step(-n);
        break;
      case 'ArrowRight':
        pb.step(n);
        break;
      case 'Home':
        pb.step(-last);
        break;
      case 'End':
        pb.step(last);
        break;
      case '[':
        pb.slower();
        break;
      case ']':
        pb.faster();
        break;
      case 'f':
      case 'F':
        toggleFullscreen();
        break;
      default:
        return;
    }
    k.preventDefault();
  });
}

async function roundPage(id: string, n: number, watching = false): Promise<void> {
  show(el('p', {}, `Loading match ${id}, round ${n}…`));
  const params = new URLSearchParams(location.search);
  const fromPlay = params.get('from') === 'play';
  const enc = encodeURIComponent(id);
  const matchURL = watching ? `/watch/matches/${enc}?after=${n}` : fromPlay ? `/play/matches/${enc}` : `/view/matches/${enc}`;
  const [logRes, matchRes] = await Promise.all([
    fetchJSON<unknown>(`/matches/${encodeURIComponent(id)}/rounds/${n}`),
    fetchJSON<MatchView>(`/matches/${encodeURIComponent(id)}`),
  ]);
  if (!matchRes.ok && matchRes.status === 404) {
    show(alertBox(`No match with id ${id}.`), el('p', {}, el('a', { href: '/view' }, 'Open another replay')));
    return;
  }
  if (!logRes.ok) {
    if (logRes.status === 404) {
      show(alertBox(`Round ${n} has not been played. `, el('a', { href: matchURL }, 'Go to the match page')));
    } else {
      show(alertBox(`Could not load round ${n}: ${logRes.message}`));
    }
    return;
  }
  const problem = validateLog(logRes.data);
  if (problem) {
    show(alertBox(problem));
    return;
  }
  const log = logRes.data as Log;
  let heading = `Match ${id} · round ${n}`;
  if (matchRes.ok) {
    const r = matchRes.data.rounds.find((x) => x.n === n);
    if (r) {
      const seat = r.p1 === 'a' ? matchRes.data.seats.a : matchRes.data.seats.b;
      heading += ` · seat ${r.p1}${seat ? ` (${seat.name})` : ''} played p1`;
    }
  }
  renderPlayer(log, {
    heading,
    matchLink: matchURL,
    backLabel: watching ? '← waiting screen' : fromPlay ? '← back to the match' : '← back to match',
    autoplay: params.get('autoplay') === '1',
    round: n,
    mode: watching ? 'spectator' : fromPlay ? 'simple' : 'full',
    ...(matchRes.ok ? { match: matchRes.data } : {}),
  });
}

// --- live match page ----------------------------------------------------------

function matchPage(id: string): void {
  const auto = el('input', { type: 'checkbox', 'aria-label': 'Open new rounds automatically' });
  auto.checked = true;
  const body = el('div', { class: 'match-body' }, el('p', {}, `Loading match ${id}…`));
  const notice = el('p', { class: 'notice', 'aria-live': 'polite' });
  show(el('div', { class: 'match' }, el('h1', {}, `Match ${id}`), el('label', { class: 'auto' }, auto, ' open new rounds automatically'), notice, body));

  let prev: MatchView | null = null;
  const poll = async (): Promise<void> => {
    const res = await fetchJSON<MatchView>(`/matches/${encodeURIComponent(id)}`);
    if (!res.ok) {
      if (res.status === 404) {
        body.replaceChildren(alertBox(`No match with id ${id}.`));
        return; // stop polling
      }
      notice.textContent = `Could not reach the server (${res.message}); retrying…`;
      setTimeout(() => void poll(), POLL_BACKOFF_MS);
      return;
    }
    notice.textContent = '';
    const v = res.data;
    const rows = matchRows(v);
    const parts: HTMLElement[] = [
      el('p', { class: 'status' }, `Status: ${rows.status}`),
      el('ul', { class: 'seats' }, ...rows.seats.map((s) => el('li', {}, s.text))),
    ];
    if (rows.outcome) parts.push(el('p', { class: 'outcome' }, rows.outcome));
    parts.push(rows.rounds.length ? el('ol', { class: 'rounds' }, ...rows.rounds.map((r) => el('li', {}, el('a', { href: r.href }, r.text)))) : el('p', {}, 'No rounds played yet.'));
    body.replaceChildren(...parts);
    const open = newRoundToOpen(prev, v);
    prev = v;
    if (open !== null && auto.checked) {
      location.assign(`/view/matches/${encodeURIComponent(id)}/rounds/${open}?from=match`);
      return;
    }
    if (!rows.finished) setTimeout(() => void poll(), POLL_MS);
  };
  void poll();
}

// --- home page ----------------------------------------------------------------

function homePage(): void {
  const idInput = el('input', { type: 'text', placeholder: '16-character match id', 'aria-label': 'Match id', autocomplete: 'off', spellcheck: 'false' });
  const form = el('form', { class: 'open-match' }, el('label', {}, 'Match ', idInput), el('button', { type: 'submit' }, 'Open match'));
  form.addEventListener('submit', (e) => {
    e.preventDefault();
    const id = idInput.value.trim();
    if (id) location.assign(`/view/matches/${encodeURIComponent(id)}`);
  });

  const errors = el('div', { class: 'file-errors', 'aria-live': 'polite' });
  const file = el('input', { type: 'file', accept: '.json,application/json', 'aria-label': 'Open a replay log file' });
  const drop = el('div', { class: 'drop', tabindex: '0' }, el('p', {}, 'Drop a replay log here (from ', el('code', {}, 'pf sim --json'), ' or a saved round), or choose a file: '), file);
  const load = (f: File) => {
    const reader = new FileReader();
    reader.onload = () => {
      const res = parseLog(String(reader.result));
      if ('error' in res) {
        errors.replaceChildren(alertBox(`${f.name}: ${res.error}`));
        return;
      }
      renderPlayer(res.log, { heading: `${f.name} · ${res.log.players[0].name} vs ${res.log.players[1].name}` });
    };
    reader.onerror = () => errors.replaceChildren(alertBox(`Could not read ${f.name}.`));
    reader.readAsText(f);
  };
  file.addEventListener('change', () => {
    const f = file.files?.[0];
    if (f) load(f);
  });
  drop.addEventListener('dragover', (e) => {
    e.preventDefault();
    drop.classList.add('over');
  });
  drop.addEventListener('dragleave', () => drop.classList.remove('over'));
  drop.addEventListener('drop', (e) => {
    e.preventDefault();
    drop.classList.remove('over');
    const f = e.dataTransfer?.files[0];
    if (f) load(f);
  });

  show(el('div', { class: 'home' }, el('h1', {}, 'Replay viewer'), form, drop, errors, el('p', { class: 'hint' }, 'Local files are read in your browser and never uploaded.')));
}

// --- routing --------------------------------------------------------------------

function route(): void {
  const path = location.pathname.replace(/\/+$/, '');
  const playing = path === '/play' || path.startsWith('/play/');
  const watching = path.startsWith('/watch/');
  const subtitle = document.querySelector('.subtitle');
  if (subtitle) subtitle.textContent = playing ? 'play' : watching ? 'watch' : 'replay viewer';
  document.title = playing ? 'Prompt Fighter — Play' : watching ? 'Prompt Fighter — Watch' : 'Prompt Fighter — Replay viewer';
  if (path === '/play') {
    lobbyPage();
    return;
  }
  const pm = /^\/play\/matches\/([^/]+)$/.exec(path);
  if (pm?.[1]) {
    playPage(decodeURIComponent(pm[1]));
    return;
  }
  const wr = /^\/watch\/matches\/([^/]+)\/rounds\/(\d+)$/.exec(path);
  if (wr?.[1] && wr[2]) {
    void roundPage(decodeURIComponent(wr[1]), Number(wr[2]), true);
    return;
  }
  const wm = /^\/watch\/matches\/([^/]+)$/.exec(path);
  if (wm?.[1]) {
    watchPage(decodeURIComponent(wm[1]));
    return;
  }
  let m = /^\/view\/matches\/([^/]+)\/rounds\/(\d+)$/.exec(path);
  if (m?.[1] && m[2]) {
    void roundPage(decodeURIComponent(m[1]), Number(m[2]));
    return;
  }
  m = /^\/view\/matches\/([^/]+)$/.exec(path);
  if (m?.[1]) {
    matchPage(decodeURIComponent(m[1]));
    return;
  }
  homePage();
}

route();
