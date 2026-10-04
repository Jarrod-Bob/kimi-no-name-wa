import { useEffect, useState, type FormEvent } from 'react';
import { api, ApiError, type OllamaSettings } from '../api';
import { CopyIcon } from '../components/icons';
import { healthLabel, healthState } from '../lib/health';
import { useAppState } from '../session/AppState';

/** Models known to suit the app, offered as suggestions beside pulled ones. */
const SUGGESTED = [
  { model: 'gemma4:31b', note: 'Default. Best wordplay; ~20 GB, wants a 24 GB+ GPU.' },
  { model: 'gemma4:26b', note: 'Mixture-of-experts: faster, ~18 GB.' },
  { model: 'gemma4:12b', note: 'For ~12 GB GPUs; ~8 GB.' },
];

export function SettingsPage() {
  const { health, healthFailed, refreshHealth, copy, notify } = useAppState();
  const [form, setForm] = useState<OllamaSettings | null>(null);
  const [defaults, setDefaults] = useState<OllamaSettings | null>(null);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api
      .settings()
      .then((s) => {
        setForm({ ollama_url: s.ollama_url, model: s.model });
        setDefaults(s.defaults);
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Loading settings failed.'));
  }, []);

  const save = async (e: FormEvent) => {
    e.preventDefault();
    if (!form) return;
    setSaving(true);
    setError('');
    try {
      const saved = await api.saveSettings(form);
      setForm({ ollama_url: saved.ollama_url, model: saved.model });
      notify('Settings saved');
      await refreshHealth();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Saving failed.');
    } finally {
      setSaving(false);
    }
  };

  const state = healthState(health, healthFailed);
  const model = form?.model || health?.ollama.model || 'gemma4:31b';
  const pull = `ollama pull ${model}`;
  const origin = window.location.origin;
  const curl = `curl -s ${origin}/api/v1/names -H "Content-Type: application/json" -d '{"description":"a bank for little ideas","tones":["witty","japanese"],"count":5,"client":"my-app"}'`;

  return (
    <div className="page settings">
      <h1 className="page-title">
        <span className="panel-kanji" aria-hidden="true">
          設定
        </span>
        Settings
      </h1>

      <section className="panel" aria-labelledby="model-title">
        <h2 id="model-title" className="section-title">
          Model
        </h2>
        <p className={`status-line ${state}`}>
          <span className="status-dot" aria-hidden="true" />
          {healthLabel(state, health?.ollama.model ?? '')}
          {health?.ollama.version && <span className="muted"> · Ollama {health.ollama.version}</span>}
        </p>
        {health?.ollama.error && <p className="muted">{health.ollama.error}</p>}

        {form && (
          <form onSubmit={save} className="settings-form">
            <div className="field">
              <label className="field-label" htmlFor="ollama-url">
                Ollama URL
              </label>
              <p className="field-hint" id="ollama-url-hint">
                Where Ollama listens. Out of the box that’s {defaults?.ollama_url}.
              </p>
              <input
                id="ollama-url"
                type="url"
                required
                value={form.ollama_url}
                aria-describedby="ollama-url-hint"
                onChange={(e) => setForm({ ...form, ollama_url: e.target.value })}
              />
            </div>
            <div className="field">
              <label className="field-label" htmlFor="model">
                Model
              </label>
              <p className="field-hint" id="model-hint">
                Any model you’ve pulled into Ollama. Suggestions include the ones already pulled.
              </p>
              <input
                id="model"
                required
                list="model-options"
                value={form.model}
                aria-describedby="model-hint"
                spellCheck={false}
                onChange={(e) => setForm({ ...form, model: e.target.value })}
              />
              <datalist id="model-options">
                {[...new Set([...(health?.ollama.models ?? []), ...SUGGESTED.map((s) => s.model)])].map((m) => (
                  <option key={m} value={m} />
                ))}
              </datalist>
              <ul className="model-suggestions">
                {SUGGESTED.map((s) => (
                  <li key={s.model}>
                    <button type="button" className="link-btn" onClick={() => setForm({ ...form, model: s.model })}>
                      {s.model}
                    </button>{' '}
                    <span className="muted small">{s.note}</span>
                  </li>
                ))}
              </ul>
            </div>
            {error && (
              <div className="error-box" role="alert">
                {error}
              </div>
            )}
            <div className="form-actions">
              <button type="submit" className="primary-btn" disabled={saving}>
                {saving ? 'Saving…' : 'Save'}
              </button>
              {defaults && (
                <button type="button" className="secondary-btn" onClick={() => setForm({ ...defaults })}>
                  Use defaults
                </button>
              )}
            </div>
          </form>
        )}
        {!form && error && (
          <div className="error-box" role="alert">
            {error}
          </div>
        )}

        <p className="field-hint">Pull the model once, in a terminal:</p>
        <div className="cmd block">
          <code>{pull}</code>
          <button type="button" className="icon-btn small" aria-label={`Copy ${pull}`} onClick={() => void copy(pull)}>
            <CopyIcon />
          </button>
        </div>
      </section>

      <section className="panel" aria-labelledby="api-title">
        <h2 id="api-title" className="section-title">
          For other apps
        </h2>
        <p>
          Local apps can ask for names over HTTP: <code>POST {origin}/api/v1/names</code> takes the same inputs as this page
          and answers with the same list. <code>GET /api/v1/health</code> says whether Ollama and the model are ready. The
          README has the full reference.
        </p>
        <div className="cmd block">
          <code>{curl}</code>
          <button type="button" className="icon-btn small" aria-label="Copy the example request" onClick={() => void copy(curl)}>
            <CopyIcon />
          </button>
        </div>
      </section>
    </div>
  );
}
