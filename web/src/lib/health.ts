import type { Health } from '../api';

export type HealthState = 'checking' | 'ready' | 'no-model' | 'load-failed' | 'offline';

/** Boils a health report (or its absence) down to what the header shows. */
export function healthState(health: Health | null, failed: boolean): HealthState {
  if (failed) return 'offline';
  if (!health) return 'checking';
  if (!health.ollama.reachable) return 'offline';
  if (!health.ollama.model_pulled) return 'no-model';
  // Reachable and pulled, but the last generation couldn't run the model.
  if (health.status !== 'ok') return 'load-failed';
  return 'ready';
}

export function healthLabel(state: HealthState, model: string): string {
  switch (state) {
    case 'checking':
      return 'Checking Ollama…';
    case 'ready':
      return `${model} ready`;
    case 'no-model':
      return `${model} not pulled`;
    case 'load-failed':
      return `${model} won’t load`;
    case 'offline':
      return 'Ollama offline';
  }
}
