/**
 * Splits typed text into chip values on commas, semicolons and newlines,
 * collapsing whitespace and dropping empties.
 */
export function splitChips(text: string): string[] {
  return text
    .split(/[,;\n]/)
    .map((part) => part.replace(/\s+/g, ' ').trim())
    .filter((part) => part !== '');
}

/** Adds the chips in `text` to `existing`, skipping case-insensitive repeats. */
export function addChips(existing: string[], text: string, limit = 20): string[] {
  const out = [...existing];
  const seen = new Set(existing.map((c) => c.toLowerCase()));
  for (const chip of splitChips(text)) {
    const key = chip.toLowerCase();
    if (seen.has(key) || out.length >= limit) continue;
    seen.add(key);
    out.push(chip.slice(0, 80));
  }
  return out;
}
