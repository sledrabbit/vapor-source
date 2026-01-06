import { useCallback, useEffect, useState } from 'react';

type ThemeMode = 'light' | 'dark';

const STORAGE_KEY = 'vapor-theme-mode';
const THEME_LABEL: Record<ThemeMode, string> = {
  light: 'Light',
  dark: 'Dark',
};

const getSystemMode = (): ThemeMode => {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return 'light';
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
};

export function ThemeToggle() {
  const [systemMode, setSystemMode] = useState<ThemeMode>(() => getSystemMode());
  const [locked, setLocked] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false;
    const stored = window.localStorage.getItem(STORAGE_KEY);
    return stored === 'light' || stored === 'dark';
  });
  const [mode, setMode] = useState<ThemeMode>(() => {
    if (typeof window === 'undefined') return 'light';
    const stored = window.localStorage.getItem(STORAGE_KEY) as ThemeMode | null;
    if (stored === 'light' || stored === 'dark') {
      return stored;
    }
    return getSystemMode();
  });

  useEffect(() => {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;
    if (locked) {
      root.dataset.theme = mode;
    } else {
      root.removeAttribute('data-theme');
    }
  }, [mode, locked]);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    if (locked) {
      window.localStorage.setItem(STORAGE_KEY, mode);
    } else {
      window.localStorage.removeItem(STORAGE_KEY);
    }
  }, [mode, locked]);

  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return;
    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = (event: MediaQueryListEvent) => {
      const next = event.matches ? 'dark' : 'light';
      setSystemMode(next);
      setLocked(false);
      setMode(next);
    };
    if (typeof media.addEventListener === 'function') {
      media.addEventListener('change', handleChange);
    } else {
      media.onchange = handleChange;
    }
    return () => {
      if (typeof media.removeEventListener === 'function') {
        media.removeEventListener('change', handleChange);
      } else {
        media.onchange = null;
      }
    };
  }, []);

  const handleToggle = useCallback(() => {
    setMode((current) => {
      const base = locked ? current : systemMode;
      const next = base === 'light' ? 'dark' : 'light';
      setLocked(next !== systemMode);
      return next;
    });
  }, [locked, systemMode]);

  const isDark = mode === 'dark';

  return (
    <button
      type="button"
      onClick={handleToggle}
      className="inline-flex items-center gap-2 rounded-md bg-transparent px-3 py-1 text-xs font-semibold text-[var(--text-secondary)] transition hover:bg-[var(--surface-muted)]"
      aria-label={`Switch to ${isDark ? 'light' : 'dark'} mode`}
      title={`Switch to ${isDark ? 'light' : 'dark'} mode`}
    >
      <span className="sr-only">Toggle theme</span>
      <span
        className={`relative inline-flex h-5 w-10 items-center rounded-full transition ${
          isDark ? 'bg-slate-900' : 'bg-slate-300'
        }`}
      >
        <span
          className={`inline-block h-4 w-4 transform rounded-full bg-white transition ${
            isDark ? 'translate-x-5' : 'translate-x-1'
          }`}
        />
      </span>
      <span>{THEME_LABEL[mode]}</span>
    </button>
  );
}
