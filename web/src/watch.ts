// The spectator waiting screen at /watch/matches/{id}: round, score, who has
// submitted and who we are waiting for. It polls the match and counts down
// into each newly played round. Spectators only read: no POSTs are sent.
import { alertBox, el, fetchJSON, show } from './dom';
import { POLL_BACKOFF_MS, POLL_MS } from './match';
import { COUNTDOWN_SECONDS, Countdown, lastPlayed, nextToWatch, parseAfter, waitingState, type WaitingState } from './spectate';
import type { MatchView, RoundView } from './types';

function roundLinks(id: string, rounds: RoundView[], view: MatchView): HTMLElement {
  if (rounds.length === 0) return el('p', { class: 'hint' }, 'No rounds played yet.');
  const name = (seat: 'a' | 'b') => (seat === 'a' ? view.seats.a?.name : view.seats.b?.name) ?? `seat ${seat}`;
  return el(
    'ol',
    { class: 'watch-rounds' },
    ...rounds.map((r) =>
      el('li', {}, el('a', { href: `/watch/matches/${encodeURIComponent(id)}/rounds/${r.n}` }, `Round ${r.n}`), ` — ${r.winner ? `${name(r.winner)} won` : 'draw'} by ${r.reason}`),
    ),
  );
}

/** A read-only field with a Copy button; textarea for longer text. */
function copyField(label: string, value: string, long = false): HTMLElement {
  const field = long
    ? el('textarea', { readonly: 'readonly', rows: '2', 'aria-label': label, class: 'copy-text' }, value)
    : el('input', { type: 'text', readonly: 'readonly', value, 'aria-label': label, class: 'copy-text' });
  const copy = el('button', { type: 'button', class: 'copy-btn' }, 'Copy');
  copy.addEventListener('click', async () => {
    try {
      await navigator.clipboard.writeText(value);
      copy.textContent = 'Copied ✓';
    } catch {
      field.select();
    }
  });
  return el('div', { class: 'copy-row' }, el('span', { class: 'copy-label' }, label), field, copy);
}

function setupView(st: Extract<WaitingState, { kind: 'setup' }>): HTMLElement[] {
  const { setup } = st;
  return [
    el(
      'section',
      { class: 'card waiting' },
      el('h2', {}, `Waiting for players (${setup.joined.length}/2 joined)`),
      setup.joined.length ? el('ul', { class: 'watch-seats' }, ...setup.joined.map((n, i) => el('li', { class: 'submitted' }, `Player ${i + 1}: ${n} ✓`))) : el('p', { class: 'hint' }, 'Nobody has joined yet.'),
      el('p', {}, 'Send this game link to both players. The first to join is player 1, the second player 2. Round 1 starts as soon as both have joined.'),
      copyField('Game link', setup.gameLink),
    ),
    el(
      'section',
      { class: 'card' },
      el('h2', {}, 'Playing with an LLM'),
      el('p', { class: 'hint' }, 'Players can join in the browser from the game link, or play with Claude Code through the pf plugin (it needs the pf binary on their PATH, from the GitHub releases). Install the plugin once:'),
      copyField('Install the plugin', setup.pluginInstall, true),
      el('p', { class: 'hint' }, 'Then join this match. This only takes a seat; nothing is submitted yet:'),
      copyField('Join command', setup.joinCommand),
      el(
        'p',
        { class: 'hint' },
        'Each round: ask the LLM to study the last replay and change fighter.pf, then run ',
        el('code', {}, '/pf:submit'),
        ' to send it. The round runs once both players have submitted.',
      ),
    ),
  ];
}

function scoreLine(s: Extract<WaitingState, { kind: 'waiting' | 'finished' }>['score']): HTMLElement {
  return el(
    'p',
    { class: 'watch-score', 'aria-label': 'Score' },
    el('span', { class: 'p1' }, `${s.a.name} ${s.a.wins}`),
    ' – ',
    el('span', { class: 'p2' }, s.b ? `${s.b.wins} ${s.b.name}` : '0 (no opponent yet)'),
  );
}

export function watchPage(id: string): void {
  const enc = encodeURIComponent(id);
  const after = parseAfter(location.search);
  let openedWith: number | null = null; // rounds already played when the page opened
  const countdown = new Countdown();
  let counting = 0; // the round being counted down to, or 0
  const status = el('div', { class: 'watch-status', 'aria-live': 'polite' });
  const body = el('div', { class: 'watch-body' }, el('p', {}, `Loading match ${id}…`));
  show(el('div', { class: 'watch' }, el('h1', {}, `Watching match ${id}`), status, body));

  const render = (view: MatchView) => {
    const s = waitingState(view, location.origin);
    if (s.kind === 'not_found') return;
    if (s.kind === 'setup') {
      openedWith ??= 0; // players are still joining: round 1 is new to this page
      status.replaceChildren();
      body.replaceChildren(...setupView(s));
      return;
    }
    openedWith ??= lastPlayed(view);
    const next = nextToWatch(view, after, openedWith);
    if (next !== null && counting !== next) {
      counting = next;
      const href = `/watch/matches/${enc}/rounds/${next}?autoplay=1`;
      const note = el('p', { class: 'countdown big' });
      const now = el('button', { type: 'button', class: 'end-btn' }, 'Watch now');
      now.addEventListener('click', () => location.assign(href));
      status.replaceChildren(el('section', { class: 'card countdown-card' }, note, now));
      countdown.start(COUNTDOWN_SECONDS, (left) => (note.textContent = `Round ${next} starting in ${left}`), () => location.assign(href));
    }
    if (s.kind === 'finished') {
      if (next === null) {
        status.replaceChildren(
          el('section', { class: 'card' }, el('h2', {}, 'Match over'), el('p', { class: 'watch-result' }, s.winner ? `${s.winner} wins the match` : 'The match is a draw')),
        );
      }
      body.replaceChildren(scoreLine(s.score), el('h2', {}, 'Rounds'), roundLinks(id, s.rounds, view));
      return;
    }
    if (next === null) {
      status.replaceChildren(el('section', { class: 'card waiting' }, el('h2', {}, `Round ${s.round}`), el('p', {}, s.waitingFor.length ? `Waiting for ${s.waitingFor.join(' and ')}…` : 'Both scripts are in; the round is being played…')));
    }
    body.replaceChildren(
      scoreLine(s.score),
      el(
        'ul',
        { class: 'watch-seats' },
        ...s.seats.map((x) => el('li', { class: x.submitted ? 'submitted' : 'pending' }, `${x.name}: `, x.submitted ? `✓ submitted for round ${s.round}` : `still writing the round ${s.round} script`)),
      ),
      el('h2', {}, 'Rounds'),
      roundLinks(id, s.rounds, view),
    );
  };

  const tick = async () => {
    const r = await fetchJSON<MatchView>(`/matches/${enc}`);
    if (!r.ok) {
      if (r.status === 404) {
        show(alertBox(`No match with id ${id}.`), el('p', {}, el('a', { href: '/play' }, 'Back to the lobby')));
        return;
      }
      status.replaceChildren(el('p', { class: 'notice' }, `Could not reach the server (${r.message}); retrying…`));
      setTimeout(() => void tick(), POLL_BACKOFF_MS);
      return;
    }
    render(r.data);
    if (r.data.status !== 'finished') setTimeout(() => void tick(), POLL_MS);
  };
  void tick();
}
