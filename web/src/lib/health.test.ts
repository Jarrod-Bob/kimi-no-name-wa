import { describe, expect, it } from 'vitest';
import type { Health } from '../api';
import { healthLabel, healthState } from './health';

function health(reachable: boolean, pulled: boolean): Health {
  return {
    status: reachable && pulled ? 'ok' : 'degraded',
    ollama: { url: 'http://127.0.0.1:11434', reachable, model: 'gemma4:31b', model_pulled: pulled, models: [] },
  };
}

describe('healthState', () => {
  it('maps the report onto what the header shows', () => {
    expect(healthState(null, false)).toBe('checking');
    expect(healthState(null, true)).toBe('offline');
    expect(healthState(health(false, false), false)).toBe('offline');
    expect(healthState(health(true, false), false)).toBe('no-model');
    expect(healthState(health(true, true), false)).toBe('ready');
    expect(healthState({ ...health(true, true), status: 'degraded' }, false)).toBe('load-failed');
  });

  it('labels each state', () => {
    expect(healthLabel('ready', 'gemma4:31b')).toBe('gemma4:31b ready');
    expect(healthLabel('no-model', 'gemma4:31b')).toBe('gemma4:31b not pulled');
    expect(healthLabel('load-failed', 'gemma4:31b')).toBe('gemma4:31b won’t load');
    expect(healthLabel('offline', '')).toBe('Ollama offline');
  });
});
