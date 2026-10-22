import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';

/**
 * Device & privacy settings.
 *
 * Persisted locally because a social worker's choices (dark mode for evening
 * paperwork, field-privacy mask for public transport) must survive a reload
 * and never depend on a network round-trip.
 */

export type ThemeMode = 'light' | 'dark' | 'system';

interface SettingsState {
  theme: ThemeMode;
  /** Mask personal identifiers on screen — for use in public spaces. */
  fieldPrivacy: boolean;
  notifyAssignments: boolean;
  notifySchedule: boolean;
  notifyReviews: boolean;
}

interface SettingsContextValue extends SettingsState {
  setTheme: (t: ThemeMode) => void;
  toggleFieldPrivacy: () => void;
  setNotify: (key: 'notifyAssignments' | 'notifySchedule' | 'notifyReviews', value: boolean) => void;
  resolvedTheme: 'light' | 'dark';
}

const STORAGE_KEY = 'ramona.sw.settings.v1';

const defaults: SettingsState = {
  theme: 'light',
  fieldPrivacy: false,
  notifyAssignments: true,
  notifySchedule: true,
  notifyReviews: true,
};

const SettingsContext = createContext<SettingsContextValue | null>(null);

function load(): SettingsState {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? { ...defaults, ...(JSON.parse(raw) as Partial<SettingsState>) } : defaults;
  } catch {
    return defaults;
  }
}

export function SettingsProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<SettingsState>(load);
  const [systemDark, setSystemDark] = useState(
    () => typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches,
  );

  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const onChange = (e: MediaQueryListEvent) => setSystemDark(e.matches);
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, []);

  const resolvedTheme = state.theme === 'system' ? (systemDark ? 'dark' : 'light') : state.theme;

  useEffect(() => {
    document.documentElement.dataset.theme = resolvedTheme;
    document.documentElement.style.colorScheme = resolvedTheme;
  }, [resolvedTheme]);

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch {
      /* storage may be unavailable — settings then stay session-only */
    }
  }, [state]);

  const value = useMemo<SettingsContextValue>(
    () => ({
      ...state,
      resolvedTheme,
      setTheme: (theme) => setState((s) => ({ ...s, theme })),
      toggleFieldPrivacy: () => setState((s) => ({ ...s, fieldPrivacy: !s.fieldPrivacy })),
      setNotify: (key, v) => setState((s) => ({ ...s, [key]: v })),
    }),
    [state, resolvedTheme],
  );

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>;
}

export function useSettings(): SettingsContextValue {
  const ctx = useContext(SettingsContext);
  if (!ctx) throw new Error('useSettings must be used inside <SettingsProvider>');
  return ctx;
}

/** Convenience hook for masking confidential values in the UI. */
export function usePrivacy() {
  const { fieldPrivacy } = useSettings();
  return useCallback((value: string) => (fieldPrivacy ? '••••••' : value), [fieldPrivacy]);
}
