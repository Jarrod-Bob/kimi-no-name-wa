import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, ApiError, type Generation, type NameIdea } from '../api';
import { NameCard } from '../components/NameCard';
import { TrashIcon } from '../components/icons';
import { formatRelative } from '../lib/formatRelative';
import { toneMeta } from '../lib/tones';

export function HistoryPage() {
  const [generations, setGenerations] = useState<Generation[] | null>(null);
  const [next, setNext] = useState<number | null>(null);
  const [error, setError] = useState('');
  const [loadingMore, setLoadingMore] = useState(false);

  const load = useCallback(async (before: number | null) => {
    try {
      const page = await api.history(before);
      setGenerations((gs) => (before && gs ? [...gs, ...page.generations] : page.generations));
      setNext(page.next_before);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Loading the history failed.');
    }
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- fetching on mount is the point
    void load(null);
  }, [load]);

  const more = async () => {
    setLoadingMore(true);
    await load(next);
    setLoadingMore(false);
  };

  const remove = async (g: Generation) => {
    const starred = g.names.filter((n) => n.favourite).length;
    const note = starred > 0 ? `\n\nIts ${starred} starred name${starred === 1 ? '' : 's'} stay in Favourites.` : '';
    if (!window.confirm(`Delete this generation for “${g.description.slice(0, 60)}”?${note}`)) return;
    try {
      await api.deleteGeneration(g.id);
      setGenerations((gs) => gs?.filter((x) => x.id !== g.id) ?? null);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Deleting failed.');
    }
  };

  const update = (idea: NameIdea) =>
    setGenerations(
      (gs) =>
        gs?.map((g) =>
          g.id === idea.generation_id ? { ...g, names: g.names.map((n) => (n.id === idea.id ? { ...n, ...idea, description: n.description } : n)) } : g,
        ) ?? null,
    );

  return (
    <div className="page">
      <h1 className="page-title">
        <span className="panel-kanji" aria-hidden="true">
          記憶
        </span>
        History
      </h1>
      <p className="page-lede">Every batch of names, from this app and from other apps calling the API.</p>
      {error && (
        <div className="error-box" role="alert">
          {error}
        </div>
      )}
      {generations === null && !error && <p className="muted">Loading…</p>}
      {generations?.length === 0 && (
        <div className="empty small">
          <h2 className="empty-title">Nothing yet.</h2>
          <p>
            Names you <Link to="/">generate</Link> are kept here.
          </p>
        </div>
      )}
      <ol className="history-list">
        {generations?.map((g) => (
          <li key={g.id} className="history-item">
            <details>
              <summary>
                <span className="history-desc">{g.description}</span>
                <span className="history-meta">
                  <time dateTime={g.created_at} title={new Date(g.created_at).toLocaleString()}>
                    {formatRelative(g.created_at)}
                  </time>
                  <span>· {g.names.length === 1 ? '1 name' : `${g.names.length} names`}</span>
                  {g.like_name && <span>· more like “{g.like_name}”</span>}
                  {g.client !== 'web' && <span className="client-badge">via {g.client}</span>}
                  <span className="history-tones" aria-label={`Tones: ${g.tones.map((t) => toneMeta(t).label).join(', ')}`}>
                    {g.tones.map((t) => (
                      <span key={t} className={`mini-tone tone-${t}`} aria-hidden="true" title={toneMeta(t).label}>
                        {toneMeta(t).glyph}
                      </span>
                    ))}
                  </span>
                </span>
                <span className="history-preview" aria-hidden="true">
                  {g.names
                    .slice(0, 5)
                    .map((n) => n.name)
                    .join(' · ')}
                </span>
              </summary>
              <div className="history-body">
                {(g.inspiration.length > 0 || g.keywords.length > 0) && (
                  <p className="muted small">
                    {g.inspiration.length > 0 && <>Inspiration: {g.inspiration.join(', ')}. </>}
                    {g.keywords.length > 0 && <>Keywords: {g.keywords.join(', ')}. </>}
                  </p>
                )}
                <div className="card-grid">
                  {g.names.map((idea, i) => (
                    <NameCard key={idea.id} idea={idea} index={i} onFavouriteChange={update} />
                  ))}
                </div>
                <div className="history-actions">
                  <span className="muted small">Model: {g.model}</span>
                  <button type="button" className="text-btn danger" onClick={() => void remove(g)}>
                    <TrashIcon />
                    Delete this batch
                  </button>
                </div>
              </div>
            </details>
          </li>
        ))}
      </ol>
      {next && (
        <button type="button" className="secondary-btn load-more" onClick={() => void more()} disabled={loadingMore}>
          {loadingMore ? 'Loading…' : 'Show older'}
        </button>
      )}
    </div>
  );
}
