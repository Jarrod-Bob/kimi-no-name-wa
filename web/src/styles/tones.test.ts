// @vitest-environment jsdom
/// <reference types="node" />
import { readFileSync } from 'node:fs';
import { afterEach, beforeAll, describe, expect, it } from 'vitest';
import { TONES } from '../lib/tones';

// The .tone-* modifiers set --tone with the same specificity as each
// component's own default, so they only win when they come later in
// index.css. jsdom resolves that cascade for custom properties.
describe('tone colours', () => {
  beforeAll(() => {
    const style = document.createElement('style');
    // Vitest stubs CSS imports (even ?raw), so read the file itself; tests run from web/.
    style.textContent = readFileSync('src/styles/index.css', 'utf8');
    document.head.appendChild(style);
  });

  afterEach(() => {
    document.body.innerHTML = '';
  });

  function toneOf(className: string): string {
    const el = document.createElement('span');
    el.className = className;
    document.body.appendChild(el);
    return getComputedStyle(el).getPropertyValue('--tone').trim();
  }

  it.each(TONES.map((t) => t.id))('colours the %s tone tag, toggle and history dot', (id) => {
    expect(toneOf(`tone-tag tone-${id}`)).toBe(`var(--t-${id})`);
    expect(toneOf(`tone-toggle tone-${id}`)).toBe(`var(--t-${id})`);
    expect(toneOf(`mini-tone tone-${id}`)).toBe(`var(--t-${id})`);
  });

  it('falls back to ink without a tone class', () => {
    expect(toneOf('tone-tag')).toBe('var(--ink-2)');
    expect(toneOf('mini-tone')).toBe('var(--ink-2)');
  });
});
