import { useState, type CSSProperties } from 'react';
import { api, ApiError, type CheckResult, type NameIdea, type SourceResult } from '../api';
import { useAppState } from '../session/AppState';
import { techniqueLabel, toneMeta } from '../lib/tones';
import { CopyIcon, ExternalIcon, SearchIcon, SparkIcon, StarIcon } from './icons';

interface Props {
  idea: NameIdea;
  /** Shown when the card asks for "more like this"; omitted hides the action. */
  onMoreLikeThis?: (idea: NameIdea) => void;
  /** Called after the star changes, with the updated name. */
  onFavouriteChange?: (idea: NameIdea) => void;
  /** Shows what the name was generated for (favourites list). */
  showContext?: boolean;
  /** Stagger index for the entrance animation. */
  index?: number;
}

type CheckState = { kind: 'idle' } | { kind: 'loading' } | { kind: 'done'; result: CheckResult } | { kind: 'error'; message: string };

export function NameCard({ idea, onMoreLikeThis, onFavouriteChange, showContext, index = 0 }: Props) {
  const { copy, toggleFavourite, notify } = useAppState();
  const [busy, setBusy] = useState(false);
  const [check, setCheck] = useState<CheckState>({ kind: 'idle' });
  const tone = toneMeta(idea.tone);

  const star = async () => {
    setBusy(true);
    try {
      const updated = await toggleFavourite(idea);
      onFavouriteChange?.(updated);
      notify(updated.favourite ? `Starred “${idea.name}”` : `Unstarred “${idea.name}”`);
    } catch (err) {
      notify(err instanceof ApiError ? err.message : 'Starring failed.');
    } finally {
      setBusy(false);
    }
  };

  const runCheck = async () => {
    setCheck({ kind: 'loading' });
    try {
      setCheck({ kind: 'done', result: await api.check(idea.name) });
    } catch (err) {
      setCheck({ kind: 'error', message: err instanceof ApiError ? err.message : 'The check failed.' });
    }
  };

  return (
    <article
      className={`name-card${idea.favourite ? ' is-favourite' : ''}`}
      style={{ '--i': index } as CSSProperties}
      aria-label={idea.name}
    >
      {idea.favourite && (
        <span className="hanko" aria-hidden="true" title="Favourite">
          結
        </span>
      )}
      <div className="name-meta">
        <span className={`tone-tag tone-${tone.id}`} title={tone.gloss}>
          <span className="tone-glyph" aria-hidden="true">
            {tone.glyph}
          </span>
          {tone.label}
        </span>
        <span className="technique-tag">{techniqueLabel(idea.technique)}</span>
      </div>

      <h3 className="name-word">{idea.name}</h3>
      <p className="name-why">{idea.explanation}</p>
      {showContext && idea.description && <p className="name-context">for: {idea.description}</p>}

      <div className="name-actions">
        <button
          type="button"
          className={`icon-btn star${idea.favourite ? ' on' : ''}`}
          aria-pressed={idea.favourite}
          aria-label={idea.favourite ? `Unstar ${idea.name}` : `Star ${idea.name}`}
          title={idea.favourite ? 'Unstar' : 'Star to keep'}
          disabled={busy}
          onClick={star}
        >
          <StarIcon filled={idea.favourite} />
        </button>
        <button type="button" className="icon-btn" aria-label={`Copy ${idea.name}`} title="Copy" onClick={() => void copy(idea.name)}>
          <CopyIcon />
        </button>
        {onMoreLikeThis && (
          <button type="button" className="text-btn" onClick={() => onMoreLikeThis(idea)}>
            <SparkIcon />
            More like this
          </button>
        )}
        <button
          type="button"
          className="text-btn"
          onClick={runCheck}
          disabled={check.kind === 'loading'}
          aria-label={`Check whether ${idea.name} is taken on GitHub and npm`}
        >
          <SearchIcon />
          {check.kind === 'loading' ? 'Checking…' : 'Taken?'}
        </button>
      </div>

      <div aria-live="polite">
        {check.kind === 'done' && (
          <dl className="check-result">
            <CheckRow label="GitHub" result={check.result.github} />
            <CheckRow label="npm" result={check.result.npm} />
          </dl>
        )}
        {check.kind === 'error' && <p className="check-error">{check.message}</p>}
      </div>
    </article>
  );
}

function CheckRow({ label, result }: { label: string; result: SourceResult }) {
  const word = result.status === 'taken' ? 'taken' : result.status === 'free' ? 'free' : 'unknown';
  return (
    <div className="check-row">
      <dt>{label}</dt>
      <dd>
        <span className={`check-pill ${result.status}`}>{word}</span>
        <a href={result.url} target="_blank" rel="noreferrer" className="check-link" title={result.detail}>
          {result.detail}
          <ExternalIcon />
          <span className="sr-only"> (opens in a new tab)</span>
        </a>
      </dd>
    </div>
  );
}
