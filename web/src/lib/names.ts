import type { Generation, NameIdea } from '../api';

/**
 * A name's identity for duplicate checks. Mirrors namegen.Key in Go: case,
 * spacing and punctuation don't make "Accele-Read" a new name.
 */
export function nameKey(name: string): string {
  return Array.from(name.toLowerCase())
    .filter((ch) => /[\p{L}\p{N}]/u.test(ch))
    .join('');
}

/**
 * Every distinct name shown across the session's batches, oldest first: what
 * a re-roll asks the server to avoid.
 */
export function shownNames(batches: Generation[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const batch of [...batches].reverse()) {
    for (const n of batch.names) {
      const key = nameKey(n.name);
      if (key && !seen.has(key)) {
        seen.add(key);
        out.push(n.name);
      }
    }
  }
  return out;
}

/** Replaces the copy of `updated` wherever it appears in the batches. */
export function replaceName(batches: Generation[], updated: NameIdea): Generation[] {
  return batches.map((batch) =>
    batch.names.some((n) => n.id === updated.id)
      ? { ...batch, names: batch.names.map((n) => (n.id === updated.id ? { ...n, ...updated, description: n.description } : n)) }
      : batch,
  );
}
