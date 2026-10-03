import { Link } from 'react-router-dom';
import { useAppState } from '../session/AppState';
import { healthState } from '../lib/health';
import { CopyIcon } from './icons';

/**
 * Explains what's missing when names can't be generated yet: Ollama not
 * installed or running, or the model not pulled. Hidden once ready.
 */
export function OllamaBanner() {
  const { health, healthFailed, refreshHealth, copy } = useAppState();
  const state = healthState(health, healthFailed);
  if (state === 'ready' || state === 'checking') return null;

  const model = health?.ollama.model ?? 'gemma4:31b';
  const pull = `ollama pull ${model}`;

  return (
    <section className="banner" role="status" aria-labelledby="banner-title">
      <div className="banner-moon" aria-hidden="true">
        {state === 'offline' ? '眠' : '待'}
      </div>
      <div className="banner-body">
        <h2 id="banner-title" className="banner-title">
          {state === 'offline' ? 'Ollama is asleep' : 'One more step: pull the model'}
        </h2>
        <p>
          {healthFailed
            ? 'Can’t reach kimi-no-name-wa’s own server. Is it still running?'
            : state === 'offline'
              ? `Names come from a model running on this PC, and nothing answered at ${health?.ollama.url}.`
              : `Ollama is running, but ${model} hasn’t been downloaded yet.`}
        </p>
        {!healthFailed && (
          <ol className="banner-steps">
            {state === 'offline' && (
              <li>
                Install Ollama from{' '}
                <a href="https://ollama.com/download" target="_blank" rel="noreferrer">
                  ollama.com/download
                </a>{' '}
                and start it.
              </li>
            )}
            <li>
              In a terminal, run{' '}
              <span className="cmd">
                <code>{pull}</code>
                <button type="button" className="icon-btn small" aria-label={`Copy ${pull}`} onClick={() => void copy(pull)}>
                  <CopyIcon />
                </button>
              </span>
            </li>
            <li>
              Come back here. This checks again by itself, or{' '}
              <button type="button" className="link-btn" onClick={() => void refreshHealth()}>
                check now
              </button>
              . A different URL or model? Change it in <Link to="/settings">Settings</Link>.
            </li>
          </ol>
        )}
      </div>
    </section>
  );
}
