// The script editor: CodeMirror 6 mounted in a ShadowRoot so that its styles
// use constructed stylesheets (adoptedStyleSheets) and the page's strict CSP
// (style-src 'self', no inline <style>) needs no exception.
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
import { syntaxHighlighting } from '@codemirror/language';
import { linter, lintGutter, type Diagnostic } from '@codemirror/lint';
import { EditorState } from '@codemirror/state';
import { EditorView, keymap, lineNumbers } from '@codemirror/view';
import { toDiagnostics, type ValidationError } from './diagnostics';
import { pfHighlight, pfLanguage } from './pflang';

export const MAX_SCRIPT_BYTES = 4096;
export const VALIDATE_DELAY_MS = 300;

export interface EditorStatus {
  state: 'checking' | 'valid' | 'invalid' | 'error';
  errors: number;
  bytes: number;
  /** The exact text the state refers to (valid/invalid/error). */
  validatedText: string | null;
  message?: string;
}

export interface EditorOptions {
  doc: string;
  onChange?: (text: string) => void;
  onStatus?: (s: EditorStatus) => void;
  label?: string;
}

export interface ScriptEditor {
  view: EditorView;
  text(): string;
  setText(text: string): void;
  /** True only when the current text has been validated and has no errors. */
  isValid(): boolean;
  focus(): void;
}

const theme = EditorView.theme(
  {
    '&': { backgroundColor: '#0d1115', color: '#e6e9ec', fontSize: '13px', border: '1px solid #2f3a45', borderRadius: '6px' },
    '&.cm-focused': { outline: '2px solid #f2b134' },
    '.cm-content': { fontFamily: "ui-monospace, 'SF Mono', Menlo, monospace", caretColor: '#f2b134', minHeight: '260px' },
    '.cm-gutters': { backgroundColor: '#12171c', color: '#6f7d8b', border: 'none' },
    '.cm-activeLine': { backgroundColor: '#161d24' },
    '.cm-activeLineGutter': { backgroundColor: '#161d24' },
    '.cm-cursor': { borderLeftColor: '#f2b134' },
    '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': { backgroundColor: '#2c3a4a' },
    '.cm-scroller': { maxHeight: '420px', overflow: 'auto' },
    '.cm-tooltip': { backgroundColor: '#1a2027', color: '#e6e9ec', border: '1px solid #3a4652' },
    '.cm-diagnostic-error': { borderLeft: '4px solid #ff5c5c' },
  },
  { dark: true },
);

export function utf8Bytes(text: string): number {
  return new TextEncoder().encode(text).length;
}

export function createEditor(host: HTMLElement, opts: EditorOptions): ScriptEditor {
  const shadow = host.shadowRoot ?? host.attachShadow({ mode: 'open' });
  let status: EditorStatus = { state: 'checking', errors: 0, bytes: utf8Bytes(opts.doc), validatedText: null };
  let inflight: AbortController | null = null;
  const report = (s: EditorStatus) => {
    status = s;
    opts.onStatus?.(s);
  };

  const source = async (view: EditorView): Promise<Diagnostic[]> => {
    const text = view.state.doc.toString();
    inflight?.abort();
    const ctl = new AbortController();
    inflight = ctl;
    report({ state: 'checking', errors: 0, bytes: utf8Bytes(text), validatedText: null });
    try {
      const r = await fetch('/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify({ script: text }),
        signal: ctl.signal,
      });
      if (!r.ok) {
        report({ state: 'error', errors: 0, bytes: utf8Bytes(text), validatedText: text, message: `validation failed (HTTP ${r.status})` });
        return [];
      }
      const body = (await r.json()) as { valid: boolean; errors: ValidationError[] };
      if (view.state.doc.toString() !== text) return []; // stale; a newer check follows
      report({ state: body.valid ? 'valid' : 'invalid', errors: body.errors.length, bytes: utf8Bytes(text), validatedText: text });
      return toDiagnostics(text, body.errors);
    } catch (e) {
      if (ctl.signal.aborted) return [];
      report({ state: 'error', errors: 0, bytes: utf8Bytes(text), validatedText: text, message: `cannot reach the server to validate (${e instanceof Error ? e.message : String(e)})` });
      return [];
    }
  };

  const view = new EditorView({
    root: shadow,
    parent: shadow,
    state: EditorState.create({
      doc: opts.doc,
      extensions: [
        lineNumbers(),
        history(),
        keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
        pfLanguage,
        syntaxHighlighting(pfHighlight),
        theme,
        lintGutter(),
        linter(source, { delay: VALIDATE_DELAY_MS }),
        EditorView.contentAttributes.of({ 'aria-label': opts.label ?? 'Fight script', spellcheck: 'false', autocapitalize: 'off' }),
        EditorView.updateListener.of((u) => {
          if (u.docChanged) {
            const text = u.state.doc.toString();
            report({ state: 'checking', errors: 0, bytes: utf8Bytes(text), validatedText: null });
            opts.onChange?.(text);
          }
        }),
      ],
    }),
  });

  return {
    view,
    text: () => view.state.doc.toString(),
    setText(text: string) {
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
    },
    isValid: () => status.state === 'valid' && status.validatedText === view.state.doc.toString() && status.bytes <= MAX_SCRIPT_BYTES,
    focus: () => view.focus(),
  };
}
