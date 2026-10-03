import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { api, type Generation, type Health, type NameIdea } from '../api';
import { replaceName } from '../lib/names';
import { DEFAULT_COUNT, DEFAULT_TONES } from '../lib/tones';

/** The generate form's inputs, kept across tab switches and reloads. */
export interface Draft {
  description: string;
  inspiration: string[];
  keywords: string[];
  tones: string[];
  count: number;
}

// eslint-disable-next-line react-refresh/only-export-components
export const emptyDraft: Draft = {
  description: '',
  inspiration: [],
  keywords: [],
  tones: DEFAULT_TONES,
  count: DEFAULT_COUNT,
};

interface AppState {
  health: Health | null;
  healthFailed: boolean;
  refreshHealth: () => Promise<void>;

  draft: Draft;
  setDraft: (update: (d: Draft) => Draft) => void;
  /** This session's batches, newest first. */
  batches: Generation[];
  addBatch: (g: Generation) => void;
  clearSession: () => void;
  /** Stars or unstars a name, updating every copy the session shows. */
  toggleFavourite: (n: NameIdea) => Promise<NameIdea>;

  toast: string;
  notify: (message: string) => void;
  copy: (text: string) => Promise<void>;
}

const Ctx = createContext<AppState | null>(null);

const STORAGE_KEY = 'kimi.session.v1';

interface Stored {
  draft: Draft;
  batches: Generation[];
}

// sessionStorage can be missing or throw (private windows, blocked storage);
// the session then simply doesn't survive a reload.
function load(): Stored {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<Stored>;
      return {
        draft: { ...emptyDraft, ...parsed.draft },
        batches: Array.isArray(parsed.batches) ? parsed.batches : [],
      };
    }
  } catch {
    // ignore
  }
  return { draft: emptyDraft, batches: [] };
}

function save(s: Stored) {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(s));
  } catch {
    // ignore
  }
}

export function AppStateProvider({ children }: { children: ReactNode }) {
  const [initial] = useState(load);
  const [draft, setDraftState] = useState<Draft>(initial.draft);
  const [batches, setBatches] = useState<Generation[]>(initial.batches);
  const [health, setHealth] = useState<Health | null>(null);
  const [healthFailed, setHealthFailed] = useState(false);
  const [toast, setToast] = useState('');
  const toastTimer = useRef<number | undefined>(undefined);

  useEffect(() => save({ draft, batches }), [draft, batches]);

  const refreshHealth = useCallback(async () => {
    try {
      setHealth(await api.health());
      setHealthFailed(false);
    } catch {
      setHealthFailed(true);
    }
  }, []);

  // Check on load, when the tab comes back, and every 20 s while Ollama
  // isn't ready, so installing it or pulling the model shows up by itself.
  const ready = health?.status === 'ok' && !healthFailed;
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- fetching on mount is the point
    void refreshHealth();
    const onVisible = () => {
      if (document.visibilityState === 'visible') void refreshHealth();
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => document.removeEventListener('visibilitychange', onVisible);
  }, [refreshHealth]);
  useEffect(() => {
    if (ready) return;
    const timer = window.setInterval(() => void refreshHealth(), 20_000);
    return () => window.clearInterval(timer);
  }, [ready, refreshHealth]);

  const notify = useCallback((message: string) => {
    setToast(message);
    window.clearTimeout(toastTimer.current);
    toastTimer.current = window.setTimeout(() => setToast(''), 2600);
  }, []);

  const copy = useCallback(
    async (text: string) => {
      try {
        await navigator.clipboard.writeText(text);
        notify(`Copied “${text}”`);
      } catch {
        notify('Copying was blocked by the browser. Select the name and copy it instead.');
      }
    },
    [notify],
  );

  const toggleFavourite = useCallback(async (n: NameIdea) => {
    const updated = await api.setFavourite(n.id, !n.favourite);
    setBatches((bs) => replaceName(bs, updated));
    return updated;
  }, []);

  const value = useMemo<AppState>(
    () => ({
      health,
      healthFailed,
      refreshHealth,
      draft,
      setDraft: (update) => setDraftState(update),
      batches,
      addBatch: (g) => setBatches((bs) => [g, ...bs]),
      clearSession: () => setBatches([]),
      toggleFavourite,
      toast,
      notify,
      copy,
    }),
    [health, healthFailed, refreshHealth, draft, batches, toggleFavourite, toast, notify, copy],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAppState(): AppState {
  const state = useContext(Ctx);
  if (!state) throw new Error('useAppState outside AppStateProvider');
  return state;
}
