import { describe, expect, it } from 'vitest';
import { goldenLogs } from './fixtures';

describe('fixtures', () => {
  it('loads the seven golden logs from sim/testdata/golden', () => {
    expect(Object.keys(goldenLogs).sort()).toEqual([
      'ducker_vs_jumper', 'ducker_vs_lazy', 'jumper_vs_lazy', 'lazy_vs_thrower', 'rushdown_vs_turtle', 'thrower_vs_rushdown', 'turtle_vs_jumper',
    ]);
    for (const log of Object.values(goldenLogs)) {
      expect(log.format_version).toBe('1');
      expect(log.frames.length).toBeGreaterThan(0);
    }
  });
});
