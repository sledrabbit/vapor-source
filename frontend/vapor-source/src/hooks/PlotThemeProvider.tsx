import type { ReactNode } from 'react';
import { usePlotTheme } from './usePlotTheme';
import { PlotThemeContext } from './plotThemeContext';

export function PlotThemeProvider({ children }: { children: ReactNode }) {
  const theme = usePlotTheme();
  return <PlotThemeContext.Provider value={theme}>{children}</PlotThemeContext.Provider>;
}
