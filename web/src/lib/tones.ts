/**
 * The tones the generator understands. Mirrors namegen.Tones in Go (the
 * source of truth for IDs); the kanji are this UI's decoration only.
 */
export interface ToneMeta {
  id: string;
  label: string;
  /** A kanji that carries the tone, shown beside the label. */
  glyph: string;
  /** The glyph's reading and meaning, for its tooltip. */
  gloss: string;
}

export const TONES: ToneMeta[] = [
  { id: 'witty', label: 'Witty', glyph: '機', gloss: 'ki · wit, quickness' },
  { id: 'punny', label: 'Punny', glyph: '洒', gloss: 'share · from dajare, a pun' },
  { id: 'sleek', label: 'Cool / sleek', glyph: '粋', gloss: 'iki · effortless style' },
  { id: 'comical', label: 'Comical', glyph: '笑', gloss: 'warai · laughter' },
  { id: 'melancholic', label: 'Melancholic', glyph: '哀', gloss: 'aware · from mono no aware' },
  { id: 'joking', label: 'Joking', glyph: '冗', gloss: 'jō · from jōdan, a joke' },
  { id: 'mythic', label: 'Mythic / epic', glyph: '神', gloss: 'kami · gods, myth' },
  { id: 'japanese', label: 'Japanese-flavoured', glyph: '和', gloss: 'wa · Japanese style' },
];

/** Mirrors namegen.DefaultTones. */
export const DEFAULT_TONES = ['witty', 'punny', 'sleek'];

/** Mirrors namegen.MaxCount and DefaultCount. */
export const MAX_COUNT = 20;
export const DEFAULT_COUNT = 8;

export function toneMeta(id: string): ToneMeta {
  return TONES.find((t) => t.id === id) ?? { id, label: id, glyph: '名', gloss: '' };
}

/** "cultural-reference" → "cultural reference". */
export function techniqueLabel(technique: string): string {
  return technique.replace(/-/g, ' ');
}
