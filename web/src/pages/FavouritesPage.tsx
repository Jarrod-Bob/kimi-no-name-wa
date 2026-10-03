import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, ApiError, type NameIdea } from '../api';
import { NameCard } from '../components/NameCard';

export function FavouritesPage() {
  const [favourites, setFavourites] = useState<NameIdea[] | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    let live = true;
    api
      .favourites()
      .then((r) => live && setFavourites(r.favourites))
      .catch((err) => live && setError(err instanceof ApiError ? err.message : 'Loading favourites failed.'));
    return () => {
      live = false;
    };
  }, []);

  // An unstarred card stays until the page is left, so a misclick can be
  // undone by starring it again.
  const update = (idea: NameIdea) =>
    setFavourites((fs) => fs?.map((f) => (f.id === idea.id ? { ...f, ...idea, description: f.description } : f)) ?? null);

  return (
    <div className="page">
      <h1 className="page-title">
        <span className="panel-kanji" aria-hidden="true">
          宝物
        </span>
        Favourites
      </h1>
      <p className="page-lede">Names you starred, newest first.</p>
      {error && (
        <div className="error-box" role="alert">
          {error}
        </div>
      )}
      {favourites === null && !error && <p className="muted">Loading…</p>}
      {favourites?.length === 0 && (
        <div className="empty small">
          <h2 className="empty-title">No favourites yet.</h2>
          <p>
            Star a name you like on the <Link to="/">Generate</Link> page and it waits for you here.
          </p>
        </div>
      )}
      {favourites && favourites.length > 0 && (
        <div className="card-grid">
          {favourites.map((idea, i) => (
            <NameCard key={idea.id} idea={idea} index={i} showContext onFavouriteChange={update} />
          ))}
        </div>
      )}
    </div>
  );
}
