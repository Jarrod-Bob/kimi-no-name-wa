/**
 * Types and calls for the Go API under /api/v1. The types mirror the Go JSON
 * shapes named in each comment; keep them in step.
 */

/** Mirrors namegen.Name plus history.SavedName's fields. */
export interface NameIdea {
  id: number;
  generation_id: number | null;
  name: string;
  technique: string;
  explanation: string;
  tone: string;
  favourite: boolean;
  favourited_at: string | null;
  /** What the name was generated for; only the favourites list fills it in. */
  description?: string;
}

/** Mirrors namegen.Seed: a name to get "more like". */
export interface Seed {
  name: string;
  technique?: string;
  explanation?: string;
}

/** Mirrors the body of POST /api/v1/names (namegen.Request + client). */
export interface GenerateRequest {
  description: string;
  inspiration?: string[];
  keywords?: string[];
  tones?: string[];
  count?: number;
  avoid?: string[];
  like?: Seed;
  client?: string;
}

/** Mirrors history.Generation. */
export interface Generation {
  id: number;
  created_at: string;
  client: string;
  model: string;
  description: string;
  inspiration: string[];
  keywords: string[];
  tones: string[];
  like_name: string;
  names: NameIdea[];
}

/** Mirrors ollama.Health inside GET /api/v1/health. */
export interface OllamaHealth {
  url: string;
  reachable: boolean;
  version?: string;
  model: string;
  model_pulled: boolean;
  models: string[];
  error?: string;
}

export interface Health {
  status: 'ok' | 'degraded';
  ollama: OllamaHealth;
}

/** Mirrors ollama.Config. */
export interface OllamaSettings {
  ollama_url: string;
  model: string;
}

export interface SettingsResponse extends OllamaSettings {
  defaults: OllamaSettings;
}

/** Mirrors namecheck.SourceResult. */
export interface SourceResult {
  status: 'taken' | 'free' | 'unknown';
  url: string;
  detail: string;
}

/** Mirrors namecheck.Result. */
export interface CheckResult {
  name: string;
  slug: string;
  github: SourceResult;
  npm: SourceResult;
}

export interface HistoryPage {
  generations: Generation[];
  next_before: number | null;
}

/** The API's single error shape: { "error": { "message": "...", "code": "..." } } */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(message: string, status: number, code: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

/** Reads an API error envelope, falling back to a generic message. */
export function parseError(status: number, text: string): ApiError {
  try {
    const body = JSON.parse(text) as { error?: { message?: string; code?: string } };
    if (body.error?.message) {
      return new ApiError(body.error.message, status, body.error.code ?? '');
    }
  } catch {
    // not JSON: fall through
  }
  return new ApiError(`The server answered ${status}.`, status, '');
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, {
      ...init,
      headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
    });
  } catch {
    throw new ApiError("Can't reach kimi-no-name-wa. Is it still running?", 0, 'offline');
  }
  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  if (!response.ok) {
    throw parseError(response.status, text);
  }
  return JSON.parse(text) as T;
}

export const api = {
  health: () => request<Health>('/api/v1/health'),
  generate: (req: GenerateRequest, signal?: AbortSignal) =>
    request<Generation>('/api/v1/names', { method: 'POST', body: JSON.stringify(req), signal }),
  history: (before?: number | null) =>
    request<HistoryPage>(`/api/v1/generations?limit=20${before ? `&before=${before}` : ''}`),
  deleteGeneration: (id: number) => request<void>(`/api/v1/generations/${id}`, { method: 'DELETE' }),
  favourites: () => request<{ favourites: NameIdea[] }>('/api/v1/favourites'),
  setFavourite: (id: number, on: boolean) =>
    request<NameIdea>(`/api/v1/names/${id}/favourite`, { method: on ? 'PUT' : 'DELETE' }),
  check: (name: string) => request<CheckResult>(`/api/v1/check?name=${encodeURIComponent(name)}`),
  settings: () => request<SettingsResponse>('/api/v1/settings'),
  saveSettings: (s: OllamaSettings) =>
    request<SettingsResponse>('/api/v1/settings', { method: 'PUT', body: JSON.stringify(s) }),
};
