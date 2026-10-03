import { describe, expect, it } from 'vitest';
import { addChips, splitChips } from './chips';

describe('splitChips', () => {
  it('splits on commas, semicolons and newlines and tidies whitespace', () => {
    expect(splitChips(' maplestory,  cold   brew;\nrainy window ,, ')).toEqual(['maplestory', 'cold brew', 'rainy window']);
  });
});

describe('addChips', () => {
  it('skips case-insensitive repeats', () => {
    expect(addChips(['Maplestory'], 'maplestory, Cold Brew')).toEqual(['Maplestory', 'Cold Brew']);
  });

  it('stops at the limit and trims long chips', () => {
    expect(addChips(['a', 'b'], 'c, d', 3)).toEqual(['a', 'b', 'c']);
    expect(addChips([], 'x'.repeat(100))[0]).toHaveLength(80);
  });
});
