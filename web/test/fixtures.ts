// Shared test fixtures: the engine's golden logs and the byte-exact `pf sim`
// text snapshots, loaded through Vite so no Node typings are needed.
import type { Log } from '../src/types';

const rawLogs = import.meta.glob<string>('../../sim/testdata/golden/*.json', { eager: true, query: '?raw', import: 'default' });
const rawText = import.meta.glob<string>('../../cmd/pf/testdata/text/*.txt', { eager: true, query: '?raw', import: 'default' });

function base(path: string, suffix: string): string {
  const file = path.slice(path.lastIndexOf('/') + 1);
  return file.endsWith(suffix) ? file.slice(0, -suffix.length) : file;
}

export const goldenLogs: Record<string, Log> = Object.fromEntries(
  Object.entries(rawLogs).map(([p, raw]) => [base(p, '.json'), JSON.parse(raw) as Log]),
);

/** pf sim text snapshots keyed by "<pair>" (no checks) and "<pair>.v" (with checks). */
export const textSnapshots: Record<string, string> = Object.fromEntries(
  Object.entries(rawText).map(([p, raw]) => [base(p, '.txt'), raw]),
);
