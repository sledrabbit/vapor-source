import { useCallback, useEffect, useRef, useState } from 'react';

type ThemeMode = 'light' | 'dark';

const STORAGE_KEY = 'vapor-theme-mode';

const getSystemMode = (): ThemeMode => {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return 'light';
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
};

export function ThemeToggle() {
  const [systemMode, setSystemMode] = useState<ThemeMode>(() => getSystemMode());
  const [isSwitching, setIsSwitching] = useState(false);
  const transitionTimeoutRef = useRef<number | null>(null);
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

  useEffect(() => {
    return () => {
      if (transitionTimeoutRef.current !== null) {
        window.clearTimeout(transitionTimeoutRef.current);
      }
    };
  }, []);

  const handleToggle = useCallback(() => {
    setIsSwitching(true);
    if (transitionTimeoutRef.current !== null) {
      window.clearTimeout(transitionTimeoutRef.current);
    }
    transitionTimeoutRef.current = window.setTimeout(() => {
      setIsSwitching(false);
    }, 200);
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
      className={`inline-flex items-center justify-center rounded-full bg-transparent p-3 text-[var(--text-secondary)] ${isSwitching ? 'transition-none' : 'transition'
        } hover:bg-[var(--surface-muted)]`}
      aria-label={`Switch to ${isDark ? 'light' : 'dark'} mode`}
      title={`Switch to ${isDark ? 'light' : 'dark'} mode`}
    >
      <span className="sr-only">Toggle theme</span>
      <span className="h-7 w-7 text-[var(--text-secondary)]" aria-hidden="true">
        {isDark ? (
          // sun svg
          <svg viewBox="0 0 24 24" className="h-7 w-7" fill="none" stroke="currentColor" strokeWidth="1.8">
            <circle cx="12" cy="12" r="5" />
            <path
              d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        ) : (
          // moon svg
          <svg viewBox="0 0 24 24" className="h-7 w-7" fill="none" stroke="currentColor" strokeWidth="1.8">
            <g transform="translate(12 12) scale(0.9) translate(-12 -12)">
              <path
                d="M12 3a6 6 0 1 0 9 9 9 9 0 1 1-9-9Z"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </g>
          </svg>
        )}
      </span>
    </button>
  );
}
