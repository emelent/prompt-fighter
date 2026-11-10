// The bundled sample bots (scripts/*.pf), embedded at build time.
const raw = import.meta.glob<string>('../../scripts/*.pf', { eager: true, query: '?raw', import: 'default' });

export interface Sample {
  name: string;
  source: string;
}

export const SAMPLES: Sample[] = Object.entries(raw)
  .map(([path, source]) => ({ name: path.slice(path.lastIndexOf('/') + 1, -'.pf'.length), source }))
  .sort((a, b) => a.name.localeCompare(b.name));

export function sample(name: string): Sample | undefined {
  return SAMPLES.find((s) => s.name === name);
}
