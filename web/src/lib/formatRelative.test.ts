import { describe, expect, it } from 'vitest';
import { formatRelative } from './formatRelative';

const now = new Date('2026-10-03T12:00:00Z');

describe('formatRelative', () => {
  it.each([
    ['2026-10-03T11:59:40Z', 'just now'],
    ['2026-10-03T11:55:00Z', '5 min ago'],
    ['2026-10-03T09:00:00Z', '3 h ago'],
    ['2026-10-02T11:00:00Z', 'yesterday'],
    ['2026-09-30T12:00:00Z', '3 days ago'],
  ])('%s → %s', (iso, want) => {
    expect(formatRelative(iso, now)).toBe(want);
  });

  it('falls back to a date after a week', () => {
    expect(formatRelative('2026-09-01T12:00:00Z', now)).not.toMatch(/ago/);
  });
});
