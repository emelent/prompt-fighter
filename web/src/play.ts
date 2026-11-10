// The lobby (/play) and the play page (/play/matches/{id}).
import { alertBox, el, fetchJSON, postJSON, show, type APIErrorBody } from './dom';
import { createEditor, MAX_SCRIPT_BYTES, type EditorStatus, type ScriptEditor } from './editor';
import { newRoundToOpen, POLL_BACKOFF_MS, POLL_MS } from './match';
import { otherSeat, playState, seatView, type PlayState, type StoredSeat } from './playstate';
import { SAMPLES, sample } from './samples';
import { browserSeatStore, LOBBY_DRAFT, type SeatStore } from './seats';
import type { Log, MatchView, Seat } from './types';
import { loadSimpleSettings, matchIdFrom, saveSimpleSettings } from './spectate';
import menuMusicUrl from './audio/menu.mp3';

interface SeatResponse {
  match_id: string;
  seat: Seat;
  token: string;
  match: MatchView;
}

const MAX_NAME_BYTES = 40;

function nameOK(name: string): boolean {
  const n = new TextEncoder().encode(name.trim()).length;
  return n >= 1 && n <= MAX_NAME_BYTES;
}

function replayURL(id: string, n: number, autoplay: boolean): string {
  return `/view/matches/${encodeURIComponent(id)}/rounds/${n}?from=play${autoplay ? '&autoplay=1' : ''}`;
}

function apiErrorBox(prefix: string, e: APIErrorBody, ...extra: (Node | string)[]): HTMLElement {
  const lines = e.errors.map((x) => el('li', {}, x.text));
  return alertBox(`${prefix}: ${e.message}`, lines.length ? el('ul', {}, ...lines) : null, ...extra);
}

/** Editor plus its status line, size counter and sample picker. */
function editorPanel(store: SeatStore, draftKey: string, initial: string, onStatus: (s: EditorStatus) => void): { node: HTMLElement; editor: ScriptEditor } {
  const host = el('div', { class: 'editor-host' });
  const statusLine = el('span', { class: 'editor-status', 'aria-live': 'polite' }, 'checking…');
  const size = el('span', { class: 'editor-size' });
  const picker = el('select', { 'aria-label': 'Start from a sample bot' }, el('option', { value: '' }, 'Start from a sample bot…'), ...SAMPLES.map((s) => el('option', { value: s.name }, s.name)));
  const editor = createEditor(host, {
    doc: initial,
    label: 'Fight script',
    onChange: (text) => store.setDraft(draftKey, text),
    onStatus: (s) => {
      statusLine.className = `editor-status ${s.state}`;
      statusLine.textContent =
        s.state === 'checking' ? 'checking…' : s.state === 'valid' ? 'valid ✓' : s.state === 'invalid' ? `${s.errors} error${s.errors === 1 ? '' : 's'}` : (s.message ?? 'cannot validate');
      size.textContent = `${s.bytes} / ${MAX_SCRIPT_BYTES} bytes`;
      size.classList.toggle('over', s.bytes > MAX_SCRIPT_BYTES);
      onStatus(s);
    },
  });
  picker.addEventListener('change', () => {
    const s = sample(picker.value);
    picker.value = '';
    if (!s) return;
    if (editor.text().trim() !== '' && !window.confirm(`Replace the editor content with the ${s.name} sample?`)) return;
    editor.setText(s.source);
    editor.focus();
  });
  const node = el('div', { class: 'editor-panel' }, el('div', { class: 'editor-bar' }, picker, statusLine, size), host);
  return { node, editor };
}

function persistenceNotice(store: SeatStore): HTMLElement | null {
  return store.persistent ? null : el('p', { class: 'notice' }, 'This browser does not allow local storage: your seat and drafts will be lost if you reload the page.');
}

// --- watch link and menu music ---------------------------------------------------

/** The shareable spectator link for a match, with a copy button. */
function watchLinkRow(id: string): HTMLElement {
  const url = `${location.origin}/watch/matches/${encodeURIComponent(id)}`;
  const input = el('input', { type: 'text', readonly: 'readonly', value: url, 'aria-label': 'Watch link', class: 'invite' });
  const copy = el('button', { type: 'button' }, 'Copy watch link');
  copy.addEventListener('click', async () => {
    try {
      await navigator.clipboard.writeText(url);
      copy.textContent = 'Copied ✓';
    } catch {
      input.select();
    }
  });
  return el('p', { class: 'watch-link hint' }, 'Spectators: ', input, copy);
}

let menuAudio: HTMLAudioElement | null = null;

/**
 * Loops the menu music while the lobby is open, from the first click or key
 * press (browsers block audio until then), with a sound toggle that shares
 * the replay player's sound setting.
 */
function menuMusic(): HTMLElement {
  const settings = loadSimpleSettings();
  const audio = new Audio(menuMusicUrl);
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
  tryPlay(); // works when the browser already allows audio
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

/** Stops the menu music (when the lobby is left within the page). */
export function stopMenuMusic(): void {
  menuAudio?.pause();
  menuAudio = null;
}

// --- lobby ---------------------------------------------------------------------

export function lobbyPage(): void {
  const store = browserSeatStore();
  const name = el('input', { type: 'text', 'aria-label': 'Fighter name', placeholder: 'your fighter name', maxlength: '40', autocomplete: 'nickname' });
  const create = el('button', { type: 'button', class: 'primary' }, 'Create match');
  const createErr = el('div', { 'aria-live': 'polite' });
  let editorValid = false;
  const refresh = () => {
    create.disabled = !(editorValid && nameOK(name.value));
  };
  const { node: panel, editor } = editorPanel(store, LOBBY_DRAFT, store.draft(LOBBY_DRAFT) ?? '', () => {
    editorValid = editor.isValid();
    refresh();
  });
  name.addEventListener('input', refresh);
  create.addEventListener('click', async () => {
    if (!editor.isValid() || !nameOK(name.value)) return;
    create.disabled = true;
    const res = await postJSON<SeatResponse>('/matches', { name: name.value.trim(), script: editor.text() });
    if (!res.ok) {
      createErr.replaceChildren(apiErrorBox('Could not create the match', res.error));
      refresh();
      return;
    }
    store.setSeat(res.data.match_id, { seat: res.data.seat, token: res.data.token, name: name.value.trim() });
    store.setDraft(res.data.match_id, editor.text());
    location.assign(`/play/matches/${encodeURIComponent(res.data.match_id)}`);
  });

  const idInput = el('input', { type: 'text', 'aria-label': 'Match id', placeholder: '16-character match id', autocomplete: 'off', spellcheck: 'false' });
  const open = el('form', { class: 'open-match' }, idInput, el('button', { type: 'submit' }, 'Open match'));
  open.addEventListener('submit', (e) => {
    e.preventDefault();
    const id = idInput.value.trim();
    if (id) location.assign(`/play/matches/${encodeURIComponent(id)}`);
  });

  const watchInput = el('input', { type: 'text', 'aria-label': 'Match id or link to watch', placeholder: 'match id or link', autocomplete: 'off', spellcheck: 'false' });
  const watchErr = el('p', { class: 'notice', 'aria-live': 'polite' });
  const watch = el('form', { class: 'open-match' }, watchInput, el('button', { type: 'submit' }, 'Watch'));
  watch.addEventListener('submit', (e) => {
    e.preventDefault();
    const wid = matchIdFrom(watchInput.value);
    if (wid) location.assign(`/watch/matches/${wid}`);
    else watchErr.textContent = 'That is not a match id or a match link (a match id is 16 characters, 0-9 and a-f).';
  });

  show(
    el(
      'div',
      { class: 'play' },
      el('div', { class: 'lobby-head' }, el('h1', {}, 'Play Prompt Fighter'), menuMusic()),
      el('p', { class: 'hint' }, 'Write your fighter’s script, create a match, and send the invite link to your opponent. Best of 3: you may revise your script before every round.'),
      persistenceNotice(store),
      el('section', { class: 'card' }, el('h2', {}, 'Create a match'), el('label', { class: 'field' }, 'Name ', name), panel, el('div', { class: 'actions' }, create), createErr),
      el('section', { class: 'card' }, el('h2', {}, 'Join a match'), el('p', { class: 'hint' }, 'Open an invite link, or paste its match id:'), open),
      el('section', { class: 'card' }, el('h2', {}, 'Watch a match'), el('p', { class: 'hint' }, 'Paste a match id or any link to the match to watch it as a spectator:'), watch, watchErr),
      el('p', { class: 'hint' }, 'Rules and language: see the player guide. Replays of any match: ', el('a', { href: '/view' }, 'replay viewer'), '.'),
    ),
  );
  editor.view.dispatch({}); // trigger the first validation pass
}

// --- play page -----------------------------------------------------------------

export function playPage(id: string): void {
  stopMenuMusic();
  const store = browserSeatStore();
  const enc = encodeURIComponent(id);
  let view: MatchView | null = null;
  let prev: MatchView | null = null;
  let loaded = false;
  let busy = false;
  let lastKey = '';

  // Persistent pieces that survive re-renders.
  let editorValid = false;
  const actionBtn = el('button', { type: 'button', class: 'primary' });
  const nameInput = el('input', { type: 'text', 'aria-label': 'Fighter name', placeholder: 'your fighter name', maxlength: '40' });
  const feedback = el('div', { 'aria-live': 'polite' });
  const refresh = () => {
    const s = currentState();
    const needName = s.kind === 'join';
    actionBtn.disabled = busy || !editorValid || (needName && !nameOK(nameInput.value));
  };
  const { node: editorNode, editor } = editorPanel(store, id, store.draft(id) ?? '', () => {
    editorValid = editor.isValid();
    refresh();
  });
  nameInput.addEventListener('input', refresh);

  const header = el('div', { class: 'play-head' });
  const body = el('div', { class: 'play-body' });
  const roundsList = el('div', { class: 'rounds-box' });
  show(el('div', { class: 'play' }, header, persistenceNotice(store), body, roundsList));

  function currentState(): PlayState {
    return playState(view, store.seat(id));
  }

  function maybeAutoplay(before: MatchView | null, after: MatchView): boolean {
    const n = newRoundToOpen(before, after);
    if (n === null) return false;
    location.assign(replayURL(id, n, true));
    return true;
  }

  async function prefillFromLastRound(seat: StoredSeat, v: MatchView): Promise<void> {
    if (store.draft(id) !== null || v.rounds.length === 0 || editor.text() !== '') return;
    const last = v.rounds[v.rounds.length - 1]!;
    const res = await fetchJSON<Log>(`/matches/${enc}/rounds/${last.n}`);
    if (!res.ok || editor.text() !== '') return;
    const mine = last.p1 === seat.seat ? res.data.players[0] : res.data.players[1];
    editor.setText(mine.script);
  }

  function forgetButton(): HTMLElement {
    const b = el('button', { type: 'button', class: 'link' }, 'forget this seat');
    b.addEventListener('click', () => {
      if (!window.confirm('Forget your seat in this match on this browser? You will not be able to submit scripts for it again.')) return;
      store.forgetSeat(id);
      lastKey = '';
      render();
    });
    return b;
  }

  async function act(kind: 'join' | 'submit'): Promise<void> {
    const s = currentState();
    if (!editor.isValid()) return;
    busy = true;
    refresh();
    feedback.replaceChildren();
    const before = view;
    if (kind === 'join') {
      const nm = nameInput.value.trim();
      const res = await postJSON<SeatResponse>(`/matches/${enc}/join`, { name: nm, script: editor.text() });
      busy = false;
      if (!res.ok) {
        feedback.replaceChildren(apiErrorBox('Could not join', res.error));
        refresh();
        return;
      }
      store.setSeat(id, { seat: res.data.seat, token: res.data.token, name: nm });
      view = res.data.match;
      if (maybeAutoplay(before, view)) return;
    } else if (s.kind === 'yourMove' || s.kind === 'waitingTheirs' || s.kind === 'waitingOpponent') {
      const res = await postJSON<{ match: MatchView }>(`/matches/${enc}/scripts`, { script: editor.text() }, s.seat.token);
      busy = false;
      if (!res.ok) {
        const extra = res.error.code === 'unauthorized' ? [el('p', {}, 'This browser’s seat token is not valid for this match. ', forgetButton())] : [];
        feedback.replaceChildren(apiErrorBox('Could not submit', res.error, ...extra));
        refresh();
        return;
      }
      view = res.data.match;
      if (maybeAutoplay(before, view)) return;
      feedback.replaceChildren(el('p', { class: 'ok' }, 'Script submitted.'));
    }
    prev = view;
    lastKey = '';
    render();
  }
  actionBtn.addEventListener('click', () => {
    const s = currentState();
    void act(s.kind === 'join' ? 'join' : 'submit');
  });

  function seatLine(v: MatchView, seat: Seat, mine: StoredSeat | null): HTMLElement {
    const sv = seatView(v, seat);
    const you = mine?.seat === seat ? ' (you)' : '';
    if (!sv) return el('li', {}, `seat ${seat}: not joined yet`);
    const sub = v.status === 'finished' ? '' : sv.submitted ? ' · next script submitted' : ' · next script not submitted';
    return el('li', {}, `seat ${seat} — `, el('strong', {}, sv.name), `${you}: ${sv.wins} win${sv.wins === 1 ? '' : 's'}${sub}`);
  }

  function render(): void {
    const s = currentState();
    const key = JSON.stringify({ k: s.kind, v: view, seat: store.seat(id) });
    if (key === lastKey) return;
    lastKey = key;
    if (s.kind === 'notFound') {
      header.replaceChildren(el('h1', {}, `Match ${id}`));
      body.replaceChildren(alertBox(`No match with id ${id}. `, el('a', { href: '/play' }, 'Back to the lobby')));
      roundsList.replaceChildren();
      return;
    }
    const v = s.view;
    const mine = s.kind === 'join' || s.kind === 'spectator' ? null : s.seat;
    const head: HTMLElement[] = [el('h1', {}, `Match ${id}`), el('ul', { class: 'seats' }, seatLine(v, 'a', mine), seatLine(v, 'b', mine))];
    if (mine) head.push(el('p', { class: 'hint' }, `You are seat ${mine.seat} (${mine.name}) on this browser · `, forgetButton()), watchLinkRow(id));
    header.replaceChildren(...head);
    roundsList.replaceChildren(
      v.rounds.length
        ? el('section', { class: 'card' }, el('h2', {}, 'Rounds'), el('ol', { class: 'rounds' }, ...v.rounds.map((r) => {
            const winner = r.winner ? `seat ${r.winner} (${seatView(v, r.winner)?.name ?? ''}) won` : 'draw';
            return el('li', {}, el('a', { href: replayURL(id, r.n, false) }, `Round ${r.n}`), ` — seat ${r.p1} was p1; ${winner} by ${r.reason} at f${r.end_frame}`);
          })))
        : el('span'),
    );
    const invite = `${location.origin}/play/matches/${enc}`;
    switch (s.kind) {
      case 'join':
        actionBtn.textContent = 'Join match';
        body.replaceChildren(
          el('section', { class: 'card' }, el('h2', {}, 'Join this match'), el('p', { class: 'hint' }, `Seat b is free. Pick a name and write your round-1 script; round 1 starts as soon as you join.`), el('label', { class: 'field' }, 'Name ', nameInput), editorNode, el('div', { class: 'actions' }, actionBtn), feedback),
        );
        break;
      case 'spectator':
        body.replaceChildren(el('section', { class: 'card' }, el('h2', {}, 'This match is full'), el('p', {}, 'Both seats are taken. You can still ', el('a', { href: `/watch/matches/${enc}` }, 'watch the match'), ' as a spectator.')));
        break;
      case 'waitingOpponent': {
        const inviteInput = el('input', { type: 'text', readonly: 'readonly', value: invite, 'aria-label': 'Invite link', class: 'invite' });
        const copy = el('button', { type: 'button' }, 'Copy invite link');
        copy.addEventListener('click', async () => {
          try {
            await navigator.clipboard.writeText(invite);
            copy.textContent = 'Copied ✓';
          } catch {
            inviteInput.select();
          }
        });
        actionBtn.textContent = 'Replace round-1 script';
        body.replaceChildren(
          el('section', { class: 'card waiting' }, el('h2', {}, 'Waiting for an opponent to join'), el('p', {}, 'Send them this link. Round 1 starts as soon as they join.'), el('div', { class: 'invite-row' }, inviteInput, copy)),
          el('section', { class: 'card' }, el('h2', {}, 'Your round-1 script'), el('p', { class: 'hint' }, 'You can still change it until your opponent joins.'), editorNode, el('div', { class: 'actions' }, actionBtn), feedback),
        );
        break;
      }
      case 'yourMove':
        actionBtn.textContent = `Submit for round ${s.round}`;
        body.replaceChildren(
          el('section', { class: 'card' }, el('h2', {}, `Your move: round ${s.round}`), el('p', { class: 'hint' }, 'Revise your script (or keep it) and submit. Watch the last replay first if you want to study it.'), editorNode, el('div', { class: 'actions' }, actionBtn), feedback),
        );
        break;
      case 'waitingTheirs': {
        const opp = seatView(v, otherSeat(s.seat.seat));
        actionBtn.textContent = `Replace my round ${s.round} script`;
        body.replaceChildren(
          el('section', { class: 'card waiting' }, el('h2', {}, `Waiting for ${opp?.name ?? 'your opponent'} to submit round ${s.round}…`), el('p', {}, 'The replay opens automatically as soon as the round is played.')),
          el('section', { class: 'card' }, el('h2', {}, 'Your submitted script'), el('p', { class: 'hint' }, 'You may still replace it until the round runs.'), editorNode, el('div', { class: 'actions' }, actionBtn), feedback),
        );
        break;
      }
      case 'finished': {
        const outcome = v.winner ? `${seatView(v, v.winner)?.name ?? `seat ${v.winner}`} (seat ${v.winner}) won the match` : 'The match was a draw';
        body.replaceChildren(el('section', { class: 'card' }, el('h2', {}, 'Match finished'), el('p', { class: 'outcome' }, outcome), el('p', {}, el('a', { href: '/play' }, 'Play another match'))));
        break;
      }
    }
    if (mine && (s.kind === 'yourMove' || s.kind === 'waitingTheirs' || s.kind === 'waitingOpponent')) void prefillFromLastRound(mine, v);
    refresh();
  }

  async function poll(): Promise<void> {
    const res = await fetchJSON<MatchView>(`/matches/${enc}`);
    if (!res.ok) {
      if (res.status === 404) {
        view = null;
        render();
        return; // stop polling
      }
      feedback.replaceChildren(el('p', { class: 'notice' }, `Could not reach the server (${res.message}); retrying…`));
      setTimeout(() => void poll(), POLL_BACKOFF_MS);
      return;
    }
    view = res.data;
    if (loaded && maybeAutoplay(prev, view)) return;
    loaded = true;
    prev = view;
    render();
    if (view.status !== 'finished') setTimeout(() => void poll(), POLL_MS);
  }
  void poll();
}
