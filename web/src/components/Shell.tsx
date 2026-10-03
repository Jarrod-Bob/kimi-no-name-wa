import { NavLink, Outlet } from 'react-router-dom';
import { useAppState } from '../session/AppState';
import { healthLabel, healthState } from '../lib/health';

const NAV = [
  { to: '/', label: 'Generate', kanji: '名付け', end: true },
  { to: '/favourites', label: 'Favourites', kanji: '宝物', end: false },
  { to: '/history', label: 'History', kanji: '記憶', end: false },
  { to: '/settings', label: 'Settings', kanji: '設定', end: false },
];

export function Shell() {
  const { health, healthFailed, toast } = useAppState();
  const state = healthState(health, healthFailed);
  const model = health?.ollama.model ?? '';

  return (
    <div className="app">
      <a className="skip-link" href="#main">
        Skip to content
      </a>
      <header className="sky">
        <div className="stars" aria-hidden="true" />
        <svg className="comet" viewBox="0 0 600 160" preserveAspectRatio="none" aria-hidden="true" focusable="false">
          <defs>
            <linearGradient id="tail" x1="0" y1="1" x2="1" y2="0">
              <stop offset="0" stopColor="#bfefff" stopOpacity="0" />
              <stop offset="1" stopColor="#ffffff" stopOpacity="0.95" />
            </linearGradient>
          </defs>
          <path d="M40 150 C 220 110, 380 70, 560 18" stroke="url(#tail)" strokeWidth="2.5" fill="none" />
          <path d="M120 150 C 280 112, 420 64, 560 18" stroke="url(#tail)" strokeWidth="1.2" fill="none" opacity="0.7" />
          <circle cx="560" cy="18" r="4" fill="#fff" />
        </svg>
        <div className="sky-inner">
          <NavLink to="/" className="brand" aria-label="kimi-no-name-wa, home">
            <span className="brand-kanji" aria-hidden="true">
              君の名は
            </span>
            <span className="brand-word">
              kimi<span className="dash">-</span>no<span className="dash">-</span>
              <span className="brand-name">name</span>
              <span className="dash">-</span>wa
            </span>
            <span className="brand-tag">names for the things you make</span>
          </NavLink>
          <NavLink to="/settings" className={`status-pill ${state}`} title={health?.ollama.error ?? health?.ollama.url ?? ''}>
            <span className="status-dot" aria-hidden="true" />
            <span className="sr-only">Model status: </span>
            {healthLabel(state, model)}
          </NavLink>
        </div>
        <nav className="tabs" aria-label="Main">
          {NAV.map((item) => (
            <NavLink key={item.to} to={item.to} end={item.end} className="tab">
              <span className="tab-kanji" aria-hidden="true">
                {item.kanji}
              </span>
              <span className="tab-label">{item.label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="cord" aria-hidden="true" />
      </header>

      <main id="main" className="main" tabIndex={-1}>
        <Outlet />
      </main>

      <footer className="footer">
        <span>
          Runs on your own machine with Ollama. Nothing leaves it except the <em>Taken?</em> checks.
        </span>
        <span className="build">build {__KIMI_BUILD__}</span>
      </footer>

      <div className="toast-region" role="status" aria-live="polite">
        {toast && <div className="toast">{toast}</div>}
      </div>
    </div>
  );
}
