# Design

## Context

- **Web app:** `web/` is a Vite + strict TypeScript app with Phaser `3.90.0`, built into the committed `web/dist` and embedded in `pf` (package `web`).
  - `src/app.ts` routes `/view`, `/view/matches/{id}` and `/view/matches/{id}/rounds/{n}` by `location.pathname`.
  - `renderPlayer(log, {heading, matchLink})` builds the round player. `matchPage` polls `GET /matches/{id}` every 2 s and auto-opens new rounds through `match.ts` `newRoundToOpen`.
  - Every DOM insertion of text goes through `el()`, using `textContent` only. A Vitest scan rejects HTML sinks in `src/*.ts`.
- **Server:** `server/api.go` registers method-checked routes with `s.route(pattern, handler, methods...)`, plus a JSON-404 catch-all on `/`. Viewer responses get `viewerHeaders`, whose CSP is `default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data: blob:; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'`. `Routes()` feeds the `docs/API.md` endpoint-table test.
- **The match API used as-is:** `POST /matches` and `POST /matches/{id}/join` return `{match_id, seat, token, match}`. `POST /matches/{id}/scripts` needs a bearer token and returns `{match}`. `GET /matches/{id}` returns the `View`, and `POST /validate` returns `{valid, errors:[{line,col,message,text}]}`. Columns are **byte** offsets (`dsl.Pos`).
- **CodeMirror 6, checked on npm and in its source:** `@codemirror/view` 6.43.15, `state` 6.7.6, `language` 6.13.1, `lint` 6.9.7, `commands` 6.11.1 and `@lezer/highlight` 1.2.5. Its styles go through `style-mod` 4.1.4. On a `Document`, that creates a `<style>` element, which our `style-src 'self'` blocks without a nonce. On a `ShadowRoot` (no `.head`, has `adoptedStyleSheets`), it uses a constructed `CSSStyleSheet`, which CSP does not restrict.

## Goals / Non-Goals

**Goals:**
- The whole Phase 5 loop in the browser (lobby, editor, submit, wait, auto-play), using only the existing API and shipped in the same binary.
- A CSP that stays exactly as strict as in Phase 4.
- The play-page logic as pure, tested functions, with the DOM kept thin.

**Non-Goals:**
- Accounts, authentication, rate limiting and public-hosting hardening.
- Autocompletion and a language server.
- Server-side changes beyond routes.

## Decisions

### D1. Files
```
web/src/play.ts        lobby + play page controllers (DOM), polling, navigation
web/src/editor.ts      createEditor(host, opts): CodeMirror in a ShadowRoot, lint source, status/size callbacks
web/src/pflang.ts      StreamLanguage tokenizer for .pf + HighlightStyle
web/src/diagnostics.ts toDiagnostics(text, errors): byte line/col → CodeMirror ranges
web/src/playstate.ts   playState(view | null, seat | null) → discriminated union of the 7 states
web/src/seats.ts       SeatStore over a Storage-like object (localStorage or in-memory fallback)
web/src/samples.ts     sample bots via import.meta.glob('../../scripts/*.pf', {query: '?raw', eager: true})
```
`app.ts` routes `/play` → `lobbyPage()` and `/play/matches/{id}` → `playPage(id)`. The existing `el`, `show` and `fetchJSON` helpers move to `web/src/dom.ts`, so that `play.ts` and `app.ts` share them. `vite.config.ts` adds `server.fs.allow: ['..']` so that the dev server may read `../scripts`. Production builds inline the samples as strings.

### D2. Routes
In `server/api.go`:

| route | handler |
|---|---|
| `GET /play` | `handleViewerPage` (same `index.html`, same headers) |
| `GET /play/matches/{id}` | `handleViewerPage` |
| `GET /{$}` | the exact root: `viewerHeaders`, then `302 Location: /play` |

Go's ServeMux treats `/{$}` as more specific than the `/` catch-all, so `GET /nope` still gets the JSON 404 and non-GET on `/` gets a JSON 405. The three rows are added to the `docs/API.md` endpoint table, and the `Routes()` test is updated.

### D3. Editor under the strict CSP
`createEditor(host, {doc, onStatus, onChange, readOnly?})` does the following:
- **Mount:** it attaches `host.attachShadow({mode: 'open'})` and creates `new EditorView({parent: shadow, root: shadow, ...})`. CodeMirror and `style-mod` then use `adoptedStyleSheets`, so no `<style>` element or nonce is needed.
- **Extensions:**
  - `lineNumbers()`, `history()` and `keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab])`
  - `pfLanguage` with `syntaxHighlighting(pfHighlight)`
  - `EditorView.theme({...})` with dark colours matching `style.css`
  - `linter(source, {delay: 300})` and `lintGutter()`
  - an update listener for `onChange`
- **Lint source:** it `POST`s `/validate` with the current text, using `AbortController` to cancel any stale request. It maps the errors with `toDiagnostics`, then reports `onStatus({state: 'valid'|'invalid'|'checking'|'error', count, bytes, validatedText})`.
- **Gating:** the play page enables its action only when `status.state === 'valid' && status.validatedText === view.state.doc.toString()`, so a keystroke after the last validation disables submission until the next result.
- **Size:** `new TextEncoder().encode(text).length`, shown against 4096 and turned red over the limit (the server would reject it anyway).

Alternative considered: a per-response CSP nonce, with the server rewriting `index.html` and setting `EditorView.cspNonce`. Rejected because the shadow root keeps the policy static and the HTML byte-identical to `dist`. A nonce is kept as the fallback if a browser lacks `adoptedStyleSheets`. All current evergreen browsers support it.

### D4. Byte columns to editor positions
The validator's `col` is a 1-based byte offset in the line. `toDiagnostics` takes the line text and walks its UTF-16 code units, adding each code point's UTF-8 length until the byte offset is reached. The diagnostic spans from there to the end of the token, or to the line's end if no token follows. Errors on a line beyond the document clamp to its last position. The severity is always `error`, and `message` is the validator's `message`. Vitest covers ASCII, a multi-byte comment before the error column, a line past the end, and an error at the end of the file.

### D5. `.pf` highlighting
`StreamLanguage.define({token})` is a small hand-written tokenizer (`@codemirror/language`'s legacy-mode API):

| text | tag |
|---|---|
| `#…` | comment |
| `x\d+` and integers | number |
| `if elif else loop repeat hold and or not` | keyword |
| the 13 actions | variableName (`function` style) |
| `enemy.*`, `distance`, `health`, `in_range` | propertyName |
| `on_hit`, `on_block` | definition keyword |
| `{ } ( )` | bracket |
| comparison operators | operator |

The word lists mirror `sim/dsl/vocab.go`. A Vitest test tokenizes every sample bot, checks that each word is classified (never the fallback), and checks the lists against the player guide's tables, which the Go guide test already keeps in sync with the engine.

### D6. Play-page state as a pure function
`playState(view: MatchView | null, seat: StoredSeat | null)` returns a discriminated union. A seat whose token is rejected by the server is handled by the submission error path (`unauthorized` → offer "forget this seat").

| state | condition |
|---|---|
| `notFound` | `view === null` |
| `spectator` | no stored seat, and both seats are taken |
| `join` | no stored seat, and seat b is empty |
| `waitingOpponent` | stored seat a, and seat b is empty |
| `finished` | the match is finished |
| `yourMove` | awaiting scripts, and this seat has not submitted |
| `waitingTheirs` | awaiting scripts, and this seat has submitted while the opponent has not |

Each state carries what the page needs: the round number, seat names and wins, the invite URL and the round links. `play.ts` renders the state and re-renders on every poll. The editor instance is created once and survives re-renders, so the cursor and undo history are kept.

### D7. Polling, auto-play and the replay link
The play page polls `GET /matches/{id}` with chained `setTimeout`s: 2 s normally, 5 s after a network error, and stopping once the match is finished.
- **After every poll or submission response:** `newRoundToOpen(prev, cur)` (reused from `match.ts`) picks the round to open, and the page navigates to `/view/matches/{id}/rounds/{n}?from=play&autoplay=1`.
- **On the first load** `prev` is null, so existing rounds are listed but not auto-opened.
- **After a create or join**, `prev` is set to the state before the request, so a join that plays round 1 opens it immediately.
- **In `roundPage`:** `URLSearchParams` reads `autoplay` and `from`. With `from=play`, the back link is `/play/matches/{id}` ("back to the match"). With `autoplay=1` and no `#f=`, the player calls `pb.play()` after mounting.

### D8. Seat tokens and drafts
`SeatStore` wraps a `Storage`. Every access is in `try/catch`, and if `localStorage` throws or is missing, it falls back to an in-memory map and sets `persistent = false`, which shows a notice.
- **Keys:** `pf.seat.<matchId>` → `{seat, token, name}` and `pf.draft.<matchId>` → the script text. The lobby uses `pf.draft.lobby`.
- **Prefill:** the editor starts from the draft, if there is one. Otherwise it uses the last submitted script, taken from this seat's script in the most recent round's log (fetched once when needed), or a sample.

Tokens in `localStorage` are readable by any script on the origin. The CSP (same-origin scripts only), the no-HTML-sinks test and textContent-only rendering are the mitigation, and that's acceptable for the trusted, local deployment the Phase 2 scope chose. Recorded as a risk.

### D9. Tests and verification
- **Vitest:** `toDiagnostics`, the `pflang` tokenizer (sample bots, guide vocabulary), `playState` (every row of D6), `SeatStore` (round trip, forget, fallback when the storage throws), `samples` (six names; sources equal `scripts/*.pf`), and the no-HTML-sinks scan, extended to the new files.
- **Go:**
  - `server` tests: `/play` and `/play/matches/{id}` return 200 HTML with the viewer headers; `GET /` redirects 302 to `/play` with headers; `POST /` returns a JSON 405; `GET /nope` returns a JSON 404.
  - The `Routes()` test and the `API.md` table.
  - `go test ./web`, for dist consistency and the frontend tests.
- **Browser (implementation task):** headless Chrome via `puppeteer-core` (the chrome-devtools MCP profile was locked in Phase 4), with two separate browser contexts as two players:
  1. Alice opens `/`, is redirected to the lobby, loads the `rushdown` sample, types `kik` to see the inline error and the disabled Create button, fixes it, and creates the match.
  2. Bob opens the invite link in the other context and joins.
  3. Both get auto-play of round 1.
  4. Alice submits, and her page shows "waiting for their script".
  5. Bob submits, and both auto-play round 2.
  6. Reloading keeps the seat and draft.
  7. A third context sees the spectator state.

  Throughout, there must be no console errors and no CSP violations. Screenshots are taken of the lobby, the editor with an error, the waiting state and the auto-playing replay.

## Risks / Trade-offs

- **[CodeMirror styles vs CSP]** → Shadow-root mounting avoids inline `<style>`. The browser check asserts there are no violations, with the per-response nonce as the documented fallback.
- **[Seat tokens in `localStorage`]** → Covered by the strict CSP, the no-sink rule and local-only deployment. Phase 5's public-hosting follow-ups (accounts, auth) remain in `ISSUES.md`.
- **[Bundle growth: CodeMirror adds roughly 250–350 KB minified]** → It's served locally from the binary and cached. Lazy-loading the editor chunk with a dynamic `import()` is a follow-up if it matters.
- **[Validation requests on every pause in typing]** → A 300 ms debounce and abort-on-change keep it to one request per pause. `/validate` is cheap and local.
- **[`.pf` word lists duplicated in TypeScript]** → The tokenizer test checks them against the player guide, which the Go guide test keeps in sync with the engine vocabulary.

## Migration Plan

Additive. The new routes and pages don't change the API contract. Players who already used `curl` or MCP are unaffected. To roll back, remove the play files and the three routes, and rebuild `dist`.

## Open Questions

- Whether the lobby should list open matches waiting for an opponent. That needs a new API endpoint (`GET /matches?status=waiting_for_opponent`) and is out of scope here; recorded as a follow-up.
