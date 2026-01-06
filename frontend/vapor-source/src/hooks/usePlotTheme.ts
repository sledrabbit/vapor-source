import { useEffect, useState } from 'react';

export type PlotTheme = {
  chartPalette: string[];
  baseFontColor: string;
  gridColor: string;
  hoverLabelFontColor: string;
  hoverLabelBgColor: string;
};

const CHART_COLOR_VARS = [
  '--chart-color-1',
  '--chart-color-2',
  '--chart-color-3',
  '--chart-color-4',
  '--chart-color-5',
  '--chart-color-6',
  '--chart-color-7',
  '--chart-color-8',
  '--chart-color-9',
] as const;

export const FALLBACK_THEME: PlotTheme = {
  chartPalette: ['#f2e9e1', '#286983', '#56949f', '#797593', '#9893a5', '#907aa9', '#b4637a', '#d7827e', '#ea9d34'],
  baseFontColor: '#0f172a',
  gridColor: '#e2e8f0',
  hoverLabelFontColor: '#9893a5',
  hoverLabelBgColor: '#f2e9e1',
};

function readPlotTheme(): PlotTheme {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return FALLBACK_THEME;
  }
  const root = document.documentElement;
  const styles = getComputedStyle(root);
  const chartPalette = CHART_COLOR_VARS.map((varName, index) => {
    const value = styles.getPropertyValue(varName).trim();
    return value || FALLBACK_THEME.chartPalette[index];
  });

  const readVar = (varName: string, fallback: string) => {
    const value = styles.getPropertyValue(varName).trim();
    return value || fallback;
  };

  return {
    chartPalette,
    baseFontColor: readVar('--text-primary', FALLBACK_THEME.baseFontColor),
    gridColor: readVar('--chart-grid-color', FALLBACK_THEME.gridColor),
    hoverLabelFontColor: readVar('--chart-hover-label-font', FALLBACK_THEME.hoverLabelFontColor),
    hoverLabelBgColor: readVar('--chart-hover-label-bg', FALLBACK_THEME.hoverLabelBgColor),
  };
}

export function usePlotTheme() {
  const [theme, setTheme] = useState<PlotTheme>(() => readPlotTheme());

  useEffect(() => {
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return;
    }
    const updateTheme = () => setTheme(readPlotTheme());
    updateTheme();

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleMediaChange = () => updateTheme();
    if (typeof mediaQuery.addEventListener === 'function') {
      mediaQuery.addEventListener('change', handleMediaChange);
    } else {
      mediaQuery.addListener(handleMediaChange);
    }

    const observer = new MutationObserver((mutations) => {
      if (mutations.some((mutation) => mutation.attributeName === 'data-theme')) {
        updateTheme();
      }
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });

    return () => {
      if (typeof mediaQuery.removeEventListener === 'function') {
        mediaQuery.removeEventListener('change', handleMediaChange);
      } else {
        mediaQuery.removeListener(handleMediaChange);
      }
      observer.disconnect();
    };
  }, []);

  return theme;
}
