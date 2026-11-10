// Seat tokens and drafts kept in the browser, keyed by match id. Falls back to
// memory (persistent = false) when localStorage is unavailable or throws.
import type { StoredSeat } from './playstate';

export interface StorageLike {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

const SEAT_PREFIX = 'pf.seat.';
const DRAFT_PREFIX = 'pf.draft.';
export const LOBBY_DRAFT = 'lobby';

class MemoryStorage implements StorageLike {
  private readonly m = new Map<string, string>();
  getItem(k: string): string | null {
    return this.m.get(k) ?? null;
  }
  setItem(k: string, v: string): void {
    this.m.set(k, v);
  }
  removeItem(k: string): void {
    this.m.delete(k);
  }
}

function probe(s: StorageLike | null | undefined): s is StorageLike {
  if (!s) return false;
  try {
    const k = 'pf.probe';
    s.setItem(k, '1');
    s.removeItem(k);
    return true;
  } catch {
    return false;
  }
}

export class SeatStore {
  readonly persistent: boolean;
  private readonly s: StorageLike;
  private readonly mem = new MemoryStorage();

  constructor(storage: StorageLike | null | undefined) {
    this.persistent = probe(storage);
    this.s = this.persistent ? (storage as StorageLike) : this.mem;
  }

  private get(k: string): string | null {
    try {
      return this.s.getItem(k);
    } catch {
      return this.mem.getItem(k);
    }
  }

  private put(k: string, v: string): void {
    try {
      this.s.setItem(k, v);
    } catch {
      this.mem.setItem(k, v);
    }
  }

  private del(k: string): void {
    try {
      this.s.removeItem(k);
    } catch {
      /* ignore */
    }
    this.mem.removeItem(k);
  }

  seat(matchId: string): StoredSeat | null {
    const raw = this.get(SEAT_PREFIX + matchId);
    if (!raw) return null;
    try {
      const v = JSON.parse(raw) as Partial<StoredSeat>;
      if ((v.seat === 'a' || v.seat === 'b') && typeof v.token === 'string' && typeof v.name === 'string') {
        return { seat: v.seat, token: v.token, name: v.name };
      }
    } catch {
      /* corrupt entry */
    }
    return null;
  }

  setSeat(matchId: string, seat: StoredSeat): void {
    this.put(SEAT_PREFIX + matchId, JSON.stringify(seat));
  }

  forgetSeat(matchId: string): void {
    this.del(SEAT_PREFIX + matchId);
  }

  draft(key: string): string | null {
    return this.get(DRAFT_PREFIX + key);
  }

  setDraft(key: string, text: string): void {
    this.put(DRAFT_PREFIX + key, text);
  }
}

/** The store over window.localStorage, or memory when it is blocked. */
export function browserSeatStore(): SeatStore {
  let ls: StorageLike | null = null;
  try {
    ls = window.localStorage;
  } catch {
    ls = null;
  }
  return new SeatStore(ls);
}
