import { createContext, useContext } from 'react';
import { FALLBACK_THEME, type PlotTheme } from './usePlotTheme';

export const PlotThemeContext = createContext<PlotTheme>(FALLBACK_THEME);

export function usePlotThemeContext() {
  return useContext(PlotThemeContext);
}
