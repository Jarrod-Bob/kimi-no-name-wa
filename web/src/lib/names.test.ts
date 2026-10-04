import { describe, expect, it } from 'vitest';
import type { Generation, NameIdea } from '../api';
import { nameKey, replaceName, shownNames } from './names';

function idea(id: number, name: string, favourite = false): NameIdea {
  return { id, generation_id: 1, name, technique: 'pun', explanation: 'x', tone: 'witty', favourite, favourited_at: null };
}

function batch(id: number, ...names: NameIdea[]): Generation {
  return {
    id,
    created_at: '2026-10-03T00:00:00Z',
    client: 'web',
    model: 'gemma4:31b',
    description: 'd',
    inspiration: [],
    keywords: [],
    tones: ['witty'],
    like_name: '',
    names,
  };
}

describe('nameKey', () => {
  it('ignores case, spacing and punctuation like namegen.Key', () => {
    expect(nameKey('Accele-Read')).toBe('acceleread');
    expect(nameKey('Kimi no Name_wa!')).toBe('kiminonamewa');
    expect(nameKey('君の名は')).toBe('君の名は');
  });
});

describe('shownNames', () => {
  it('lists distinct names oldest batch first', () => {
    const batches = [batch(2, idea(3, 'Skimurai'), idea(4, 'accele-read')), batch(1, idea(1, 'acceleread'), idea(2, 'nuggets'))];
    expect(shownNames(batches)).toEqual(['acceleread', 'nuggets', 'Skimurai']);
  });

  it('is empty for no batches', () => {
    expect(shownNames([])).toEqual([]);
  });
});

describe('replaceName', () => {
  it('updates every copy of a name and leaves others alone', () => {
    const batches = [batch(1, idea(1, 'a'), idea(2, 'b'))];
    const next = replaceName(batches, idea(2, 'b', true));
    expect(next[0].names[1].favourite).toBe(true);
    expect(next[0].names[0]).toBe(batches[0].names[0]);
  });
});
