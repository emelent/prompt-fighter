// Shared DOM helpers. All player-supplied text is inserted with textContent only.

export type Child = Node | string | null | undefined | false;

/** Creates an element; string children become text nodes, never markup. */
export function el<K extends keyof HTMLElementTagNameMap>(tag: K, attrs: Record<string, string> = {}, ...children: Child[]): HTMLElementTagNameMap[K] {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === 'class') e.className = v;
    else e.setAttribute(k, v);
  }
  for (const c of children) {
    if (c === null || c === undefined || c === false) continue;
    e.append(typeof c === 'string' ? document.createTextNode(c) : c);
  }
  return e;
}

export function show(...nodes: Child[]): void {
  const app = document.getElementById('app') as HTMLElement;
  app.replaceChildren();
  for (const n of nodes) if (n) app.append(typeof n === 'string' ? document.createTextNode(n) : n);
}

export function alertBox(text: string, ...extra: Child[]): HTMLElement {
  return el('div', { class: 'alert', role: 'alert' }, text, ...extra);
}

export async function fetchJSON<T>(url: string): Promise<{ ok: true; data: T } | { ok: false; status: number; message: string }> {
  try {
    const r = await fetch(url, { headers: { Accept: 'application/json' } });
    if (!r.ok) {
      let message = `HTTP ${r.status}`;
      try {
        const body = (await r.json()) as { error?: { message?: string } };
        if (body.error?.message) message = body.error.message;
      } catch {
        /* not JSON */
      }
      return { ok: false, status: r.status, message };
    }
    return { ok: true, data: (await r.json()) as T };
  } catch (e) {
    return { ok: false, status: 0, message: e instanceof Error ? e.message : String(e) };
  }
}

export interface APIErrorBody {
  code: string;
  message: string;
  errors: { player?: string; line: number; col: number; message: string; text: string }[];
}

/** POSTs JSON; on failure returns the API's error envelope (or a network error). */
export async function postJSON<T>(url: string, body: unknown, token?: string): Promise<{ ok: true; data: T } | { ok: false; status: number; error: APIErrorBody }> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json', Accept: 'application/json' };
  if (token) headers.Authorization = `Bearer ${token}`;
  try {
    const r = await fetch(url, { method: 'POST', headers, body: JSON.stringify(body) });
    const data = (await r.json().catch(() => null)) as unknown;
    if (r.ok) return { ok: true, data: data as T };
    const env = data as { error?: { code?: string; message?: string }; errors?: APIErrorBody['errors'] } | null;
    return {
      ok: false,
      status: r.status,
      error: { code: env?.error?.code ?? `http_${r.status}`, message: env?.error?.message ?? `HTTP ${r.status}`, errors: env?.errors ?? [] },
    };
  } catch (e) {
    return { ok: false, status: 0, error: { code: 'network', message: e instanceof Error ? e.message : String(e), errors: [] } };
  }
}
