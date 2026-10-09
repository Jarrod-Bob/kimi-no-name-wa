import { useEffect, useLayoutEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react';
import { flushSync } from 'react-dom';
import { api, ApiError, type Generation, type NameIdea } from '../api';
import { ChipInput } from '../components/ChipInput';
import { NameCard } from '../components/NameCard';
import { OllamaBanner } from '../components/OllamaBanner';
import { RerollIcon, SparkIcon } from '../components/icons';
import { shownNames } from '../lib/names';
import { MAX_COUNT, TONES } from '../lib/tones';
import { emptyDraft, useAppState, type Draft } from '../session/AppState';

// The server refuses more; it only reads the newest few anyway.
const MAX_AVOID = 500;

const EXAMPLE: Draft = {
  ...emptyDraft,
  description: 'A tool that ingests documents much faster than reading them myself.',
  inspiration: ['maplestory', 'cold brew', 'rainy window'],
  tones: ['witty', 'punny', 'japanese'],
};

interface Pending {
  label: string;
  count: number;
}

/** What a generation asks for, beyond the shared inputs. */
interface Ask {
  inputs: Pick<Generation, 'description' | 'inspiration' | 'keywords' | 'tones'>;
  like?: NameIdea;
  label: string;
}

export function GeneratePage() {
  const { draft, setDraft, batches, addBatch, clearSession, health, refreshHealth } = useAppState();
  const [pending, setPending] = useState<Pending | null>(null);
  const [error, setError] = useState<ApiError | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const resultsRef = useRef<HTMLHeadingElement | null>(null);
  const [focusLatest, setFocusLatest] = useState(false);
  const latestDraft = useRef(draft);
  useLayoutEffect(() => {
    latestDraft.current = draft;
  }, [draft]);

  useEffect(() => () => abortRef.current?.abort(), []);
  useEffect(() => {
    if (focusLatest && resultsRef.current) {
      resultsRef.current.focus();
      resultsRef.current.scrollIntoView({ block: 'start', behavior: 'smooth' });
      setFocusLatest(false);
    }
  }, [focusLatest, batches]);

  const generate = async ({ inputs, like, label }: Ask) => {
    if (inputs.description.trim() === '') {
      setError(new ApiError('Describe what you’re naming first.', 400, 'invalid_request'));
      return;
    }
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setError(null);
    setPending({ label, count: draft.count });

    // Avoid every name already shown for this same project, so a re-roll
    // never repeats itself.
    const sameProject = batches.filter((b) => b.description.trim() === inputs.description.trim());
    try {
      const batch = await api.generate(
        {
          description: inputs.description,
          inspiration: inputs.inspiration,
          keywords: inputs.keywords,
          tones: inputs.tones,
          count: draft.count,
          avoid: shownNames(sameProject).slice(-MAX_AVOID),
          like: like ? { name: like.name, technique: like.technique, explanation: like.explanation } : undefined,
          client: 'web',
        },
        controller.signal,
      );
      addBatch(batch);
      setFocusLatest(true);
      // A working generation clears a model that failed to load last time.
      if (health?.status === 'degraded') void refreshHealth();
    } catch (err) {
      if (controller.signal.aborted) return;
      const apiErr = err instanceof ApiError ? err : new ApiError('Generating failed.', 0, '');
      setError(apiErr);
      if (['ollama_unreachable', 'model_missing', 'model_error'].includes(apiErr.code)) void refreshHealth();
    } finally {
      if (abortRef.current === controller) {
        abortRef.current = null;
        setPending(null);
      }
    }
  };

  const fromDraft = (d: Draft): Ask => ({
    inputs: { description: d.description, inspiration: d.inspiration, keywords: d.keywords, tones: d.tones },
    label: 'Finding names',
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    void generate(fromDraft(draft));
  };

  const onFormKeyDown = (e: KeyboardEvent<HTMLFormElement>) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      // A chip box commits its typed text on this same keypress; render that
      // before reading the draft so the text is in the request.
      flushSync(() => {});
      void generate(fromDraft(latestDraft.current));
    }
  };

  const moreLike = (batch: Generation) => (idea: NameIdea) =>
    void generate({ inputs: batch, like: idea, label: `More like “${idea.name}”` });

  const reroll = () => {
    const latest = batches[0];
    if (!latest) return;
    void generate({ inputs: latest, label: 'Re-rolling' });
  };

  const stop = () => {
    abortRef.current?.abort();
    abortRef.current = null;
    setPending(null);
  };

  const toggleTone = (id: string) =>
    setDraft((d) => {
      const on = d.tones.includes(id);
      // Keep at least one tone selected; the server would default anyway,
      // but an empty row of toggles reads as broken.
      if (on && d.tones.length === 1) return d;
      return { ...d, tones: on ? d.tones.filter((t) => t !== id) : [...d.tones, id] };
    });

  const shownCount = shownNames(batches.filter((b) => batches[0] && b.description === batches[0].description)).length;
  const model = health?.ollama.model;

  return (
    <div className="generate">
      <OllamaBanner />
      <form className="panel form-panel" onSubmit={onSubmit} onKeyDown={onFormKeyDown} aria-labelledby="form-title">
        <h1 id="form-title" className="panel-title">
          <span className="panel-kanji" aria-hidden="true">
            名付け
          </span>
          What are you naming?
        </h1>

        <div className="field">
          <label className="field-label" htmlFor="description">
            Describe it
          </label>
          <p className="field-hint" id="description-hint">
            What it does, who it’s for, how it feels. A sentence or two is plenty.
          </p>
          <textarea
            id="description"
            required
            rows={4}
            maxLength={2000}
            value={draft.description}
            aria-describedby="description-hint"
            placeholder="A tool that ingests documents faster than I can read them…"
            onChange={(e) => setDraft((d) => ({ ...d, description: e.target.value }))}
          />
        </div>

        <ChipInput
          label="Surroundings & inspiration"
          hint="Things you like or can see right now: a game, your desk, the weather, a song."
          placeholder="maplestory, cold brew, rainy window…"
          value={draft.inspiration}
          onChange={(inspiration) => setDraft((d) => ({ ...d, inspiration }))}
        />

        <ChipInput
          label="Keywords"
          hint="Words to work in where they fit."
          placeholder="read, fast, documents…"
          value={draft.keywords}
          onChange={(keywords) => setDraft((d) => ({ ...d, keywords }))}
        />

        <fieldset className="field tones">
          <legend className="field-label">Tones</legend>
          <p className="field-hint">Pick one or more. Each name says which it went for.</p>
          <div className="tone-grid">
            {TONES.map((tone) => {
              const on = draft.tones.includes(tone.id);
              return (
                <button
                  key={tone.id}
                  type="button"
                  className={`tone-toggle tone-${tone.id}${on ? ' on' : ''}`}
                  aria-pressed={on}
                  title={tone.gloss}
                  onClick={() => toggleTone(tone.id)}
                >
                  <span className="tone-glyph" aria-hidden="true">
                    {tone.glyph}
                  </span>
                  {tone.label}
                </button>
              );
            })}
          </div>
        </fieldset>

        <div className="field count-field">
          <label className="field-label" htmlFor="count">
            How many
          </label>
          <div className="count-row">
            <input
              id="count"
              type="range"
              min={1}
              max={MAX_COUNT}
              value={draft.count}
              onChange={(e) => setDraft((d) => ({ ...d, count: Number(e.target.value) }))}
            />
            <output htmlFor="count" className="count-value">
              {draft.count}
            </output>
          </div>
        </div>

        <div className="form-actions">
          <button type="submit" className="primary-btn" disabled={pending !== null}>
            <SparkIcon />
            {pending ? 'Naming…' : 'Name it'}
          </button>
          <span className="kbd-hint" aria-hidden="true">
            <kbd>Ctrl</kbd> + <kbd>Enter</kbd>
          </span>
        </div>
      </form>

      <section className="results" aria-labelledby="results-title">
        <h2 id="results-title" className="sr-only">
          Names
        </h2>

        {error && (
          <div className="error-box" role="alert">
            <strong>That didn’t work.</strong> {error.message}
          </div>
        )}

        {pending && (
          <div className="pending" aria-live="polite">
            <div className="pending-head">
              <span className="comet-spinner" aria-hidden="true" />
              <span>
                {pending.label}
                {model ? ` with ${model}` : ''}… the first run after Ollama starts loads the model and can take a minute.
              </span>
              <button type="button" className="text-btn" onClick={stop}>
                Stop
              </button>
            </div>
            <div className="card-grid">
              {Array.from({ length: Math.min(pending.count, 6) }, (_, i) => (
                <div key={i} className="name-card skeleton" aria-hidden="true" style={{ animationDelay: `${i * 90}ms` }} />
              ))}
            </div>
          </div>
        )}

        {batches.length === 0 && !pending && (
          <div className="empty">
            <svg className="empty-art" viewBox="0 0 220 140" aria-hidden="true" focusable="false">
              <path d="M10 120 C 70 100, 130 60, 200 16" className="empty-tail" />
              <path d="M30 128 C 90 104, 150 56, 200 16" className="empty-tail thin" />
              <circle cx="200" cy="16" r="5" className="empty-head" />
              <path d="M20 132 C 60 120, 90 138, 130 126 S 190 118, 210 124" className="empty-cord" />
            </svg>
            <h2 className="empty-title">Every project is waiting to hear its name.</h2>
            <p>Describe it, toss in a few things from around you, pick some tones, and the names come to you.</p>
            <button type="button" className="secondary-btn" onClick={() => setDraft(() => EXAMPLE)}>
              Try the example: a faster way to read documents
            </button>
          </div>
        )}

        {batches.map((batch, bi) => (
          <section key={batch.id} className="batch" aria-labelledby={`batch-${batch.id}`}>
            <header className="batch-head">
              <h2 id={`batch-${batch.id}`} className="batch-title" tabIndex={-1} ref={bi === 0 ? resultsRef : undefined}>
                <span className="batch-no">#{batches.length - bi}</span>
                {batch.like_name ? <>More like “{batch.like_name}”</> : <>{batch.names.length === 1 ? '1 name' : `${batch.names.length} names`}</>}
              </h2>
              <p className="batch-for" title={batch.description}>
                for: {batch.description}
              </p>
            </header>
            <div className="card-grid">
              {batch.names.map((idea, i) => (
                <NameCard key={idea.id} idea={idea} index={i} onMoreLikeThis={moreLike(batch)} />
              ))}
            </div>
            {bi === 0 && (
              <div className="reroll-row">
                <button type="button" className="secondary-btn" onClick={reroll} disabled={pending !== null}>
                  <RerollIcon />
                  Re-roll
                </button>
                <span className="reroll-hint">New names for the same brief, none of the {shownCount} already shown.</span>
                <button type="button" className="link-btn clear-btn" onClick={clearSession} disabled={pending !== null}>
                  Clear this session
                </button>
              </div>
            )}
          </section>
        ))}
      </section>
    </div>
  );
}
