import { useEffect, useMemo, useState } from 'react';
import type { Data, Layout } from 'plotly.js';
import Plotly from 'plotly.js/lib/core';
import bar from 'plotly.js/lib/bar';
import box from 'plotly.js/lib/box';
import heatmap from 'plotly.js/lib/heatmap';
import scatter from 'plotly.js/lib/scatter';
import createPlotlyComponent from 'react-plotly.js/factory';
import { usePlotThemeContext } from '../hooks/theme';

Plotly.register([box, bar, scatter, heatmap]);

const Plot = createPlotlyComponent(Plotly);

const basePlotConfig = { displayModeBar: false, responsive: true } as const;
function buildHeatmapColorscale(palette: string[]) {
  const steps = palette.slice(0, 9);
  const segment = steps.length > 1 ? 1 / (steps.length - 1) : 1;
  return steps.map((color, index) => [Number((index * segment).toFixed(4)), color]) as Array<[number, string]>;
}

function createBaseBoxLayout(font: Layout['font'], gridColor: string): Partial<Layout> {
  return {
    margin: { l: 60, r: 20, t: 40, b: 80 },
    paper_bgcolor: 'rgba(0,0,0,0)',
    plot_bgcolor: 'rgba(0,0,0,0)',
    showlegend: false,
    hovermode: 'closest',
    font,
    yaxis: {
      title: { text: 'Min years of experience' },
      zeroline: false,
      gridcolor: gridColor,
      range: [0, null],
    },
    xaxis: {
      automargin: true,
      tickangle: -35,
    },
    height: 360,
  };
}

function createBaseHorizontalBarLayout(font: Layout['font'], gridColor: string): Partial<Layout> {
  return {
    margin: { l: 160, r: 20, t: 60, b: 40 },
    paper_bgcolor: 'rgba(0,0,0,0)',
    plot_bgcolor: 'rgba(0,0,0,0)',
    font,
    xaxis: { gridcolor: gridColor },
    yaxis: { automargin: true, autorange: 'reversed' as const },
  };
}

function createBaseVerticalBarLayout(font: Layout['font'], gridColor: string): Partial<Layout> {
  return {
    margin: { l: 50, r: 20, t: 60, b: 80 },
    paper_bgcolor: 'rgba(0,0,0,0)',
    plot_bgcolor: 'rgba(0,0,0,0)',
    font,
    xaxis: { automargin: true, tickangle: -35 },
    yaxis: { gridcolor: gridColor, rangemode: 'tozero' },
  };
}

function useChartVisuals() {
  const theme = usePlotThemeContext();
  const chartPalette = theme.chartPalette;
  const beeswarmPalette = useMemo(() => chartPalette.filter((_, idx) => idx !== 0), [chartPalette]);
  const baseFont = useMemo(
    () => ({ family: 'Inter, system-ui, sans-serif', color: theme.baseFontColor }),
    [theme.baseFontColor],
  );
  const baseBoxLayout = useMemo(() => createBaseBoxLayout(baseFont, theme.gridColor), [baseFont, theme.gridColor]);
  const baseHorizontalBarLayout = useMemo(
    () => createBaseHorizontalBarLayout(baseFont, theme.gridColor),
    [baseFont, theme.gridColor],
  );
  const baseVerticalBarLayout = useMemo(
    () => createBaseVerticalBarLayout(baseFont, theme.gridColor),
    [baseFont, theme.gridColor],
  );
  const heatmapColorscale = useMemo(() => buildHeatmapColorscale(chartPalette), [chartPalette]);
  const domainFill = chartPalette[5] ?? chartPalette[0];
  const modalityFill = chartPalette[7] ?? chartPalette[2];
  const degreeFill = chartPalette[4] ?? chartPalette[1];
  const yoeFill = chartPalette[2] ?? chartPalette[0];

  return {
    chartPalette,
    beeswarmPalette,
    baseFont,
    baseBoxLayout,
    baseHorizontalBarLayout,
    baseVerticalBarLayout,
    heatmapColorscale,
    domainFill,
    modalityFill,
    degreeFill,
    yoeFill,
    hoverLabelFontColor: theme.hoverLabelFontColor,
    hoverLabelBgColor: theme.hoverLabelBgColor,
    gridColor: theme.gridColor,
  };
}

const MOBILE_MAX_WIDTH = 640;

function useCompactScreen(maxWidth = MOBILE_MAX_WIDTH) {
  const [isCompact, setIsCompact] = useState(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return false;
    }
    return window.matchMedia(`(max-width: ${maxWidth}px)`).matches;
  });

  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return;
    }
    const query = window.matchMedia(`(max-width: ${maxWidth}px)`);
    const handleChange = (event: MediaQueryListEvent) => setIsCompact(event.matches);
    if (typeof query.addEventListener === 'function') {
      query.addEventListener('change', handleChange);
      return () => query.removeEventListener('change', handleChange);
    }
    query.addListener(handleChange);
    return () => query.removeListener(handleChange);
  }, [maxWidth]);

  return isCompact;
}

export type BoxStat = {
  label: string;
  values: number[];
  count: number;
  median: number;
};

export type DomainCount = {
  label: string;
  count: number;
};

export type LanguageSample = {
  label: string;
  value: number;
};

export type DomainHeatmapData = {
  domains: string[];
  buckets: string[];
  z: number[][];
};

export type DomainTrendSeries = {
  domain: string;
  x: string[];
  y: number[];
};

type BeeswarmPlotProps = {
  samples: LanguageSample[];
};

type MinYoeBoxPlotProps = {
  stats: BoxStat[];
  title: string;
  color: string;
  emptyMessage: string;
  hoverLabelColor?: string;
  hoverLabelBg?: string;
};

export function MinYoeBoxPlot({ stats, title, color, emptyMessage, hoverLabelColor, hoverLabelBg }: MinYoeBoxPlotProps) {
  const { baseBoxLayout, hoverLabelFontColor, hoverLabelBgColor } = useChartVisuals();
  const isCompact = useCompactScreen();
  const effectiveHoverLabelColor = hoverLabelColor ?? hoverLabelFontColor;
  const effectiveHoverLabelBg = hoverLabelBg ?? hoverLabelBgColor;
  const plotData = useMemo<Data[]>(
    () =>
      stats.map((entry) => ({
        type: 'box' as const,
        y: entry.values,
        name: entry.label,
        boxpoints: 'suspectedoutliers' as const,
        marker: { color },
        hovertemplate: `<b>${entry.label}</b><br>Min YOE: %{y}<extra></extra>`,
        hoverlabel: { font: { color: effectiveHoverLabelColor }, bgcolor: effectiveHoverLabelBg },
      })),
    [stats, color, effectiveHoverLabelBg, effectiveHoverLabelColor],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      ...baseBoxLayout,
      margin: isCompact ? { l: 40, r: 12, t: 32, b: 60 } : baseBoxLayout.margin,
      height: isCompact ? 280 : baseBoxLayout.height,
      yaxis: baseBoxLayout.yaxis,
    }),
    [baseBoxLayout, isCompact],
  );

  const chartHeight = (layout.height as number | undefined) ?? 360;

  return (
    <div className="surface-panel rounded-2xl p-4">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-[var(--text-primary)]">{title}</h3>
      </div>
      {plotData.length > 0 ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${chartHeight}px` }} />
      ) : (
        <p className="text-sm text-[var(--text-muted)]">{emptyMessage}</p>
      )}
    </div>
  );
}

type DomainPopularityChartProps = {
  domains: DomainCount[];
  totalJobs: number;
  height?: number;
};

export function DomainPopularityChart({
  domains,
  totalJobs,
  height,
}: DomainPopularityChartProps) {
  const isCompact = useCompactScreen();
  const { baseHorizontalBarLayout, domainFill, hoverLabelFontColor, hoverLabelBgColor } = useChartVisuals();
  const plotData = useMemo<Data[]>(
    () => [
      {
        type: 'bar' as const,
        orientation: 'h' as const,
        x: domains.map((entry) => entry.count),
        y: domains.map((entry) => entry.label),
        marker: { color: domainFill },
        hovertemplate: '<b>%{y}</b><br>Postings: %{x}<extra></extra>',
        hoverlabel: { font: { color: hoverLabelFontColor }, bgcolor: hoverLabelBgColor },
      },
    ],
    [domainFill, domains, hoverLabelBgColor, hoverLabelFontColor],
  );

  const layout = useMemo<Partial<Layout>>(
    () => {
      const autoHeight = Math.max(720, Math.min(900, 48 * domains.length + 260));
      const targetHeight = height ?? autoHeight;
      const finalHeight = isCompact ? Math.max(360, Math.min(600, targetHeight)) : targetHeight;
      return {
        ...baseHorizontalBarLayout,
        margin: isCompact ? { l: 90, r: 12, t: 40, b: 32 } : baseHorizontalBarLayout.margin,
        xaxis: {
          ...(baseHorizontalBarLayout.xaxis ?? {}),
          title: { text: 'Postings' },
        },
        height: finalHeight,
      };
    },
    [baseHorizontalBarLayout, domains.length, height, isCompact],
  );

  const chartHeight = (layout.height as number | undefined) ?? 360;

  return (
    <div className="surface-panel flex h-full flex-col rounded-2xl p-4" style={{ minHeight: chartHeight }}>
      <div className="mb-3 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-[var(--text-primary)]">Domain Popularity</h3>
        </div>
        <span className="text-xs font-semibold uppercase text-[var(--text-secondary)]">{totalJobs} jobs</span>
      </div>
      {domains.length > 0 ? (
        <div className="flex-1" style={{ minHeight: 0 }}>
          <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: '100%' }} />
        </div>
      ) : (
        <p className="text-sm text-[var(--text-muted)]">No domains available yet.</p>
      )}
    </div>
  );
}

type ModalityPopularityChartProps = {
  modalities: DomainCount[];
};

export function ModalityPopularityChart({ modalities }: ModalityPopularityChartProps) {
  const isCompact = useCompactScreen();
  const { baseVerticalBarLayout, modalityFill, hoverLabelFontColor, hoverLabelBgColor } = useChartVisuals();
  const plotData = useMemo<Data[]>(
    () => [
      {
        type: 'bar' as const,
        x: modalities.map((entry) => entry.label),
        y: modalities.map((entry) => entry.count),
        marker: { color: modalityFill },
        hovertemplate: '<b>%{x}</b><br>Postings: %{y}<extra></extra>',
        hoverlabel: { font: { color: hoverLabelFontColor }, bgcolor: hoverLabelBgColor },
      },
    ],
    [hoverLabelBgColor, hoverLabelFontColor, modalities, modalityFill],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      ...baseVerticalBarLayout,
      margin: isCompact ? { l: 40, r: 12, t: 40, b: 60 } : baseVerticalBarLayout.margin,
      xaxis: {
        ...(baseVerticalBarLayout.xaxis ?? {}),
        tickangle: isCompact ? -15 : baseVerticalBarLayout.xaxis?.tickangle,
      },
      height: isCompact ? 260 : 320,
    }),
    [baseVerticalBarLayout, isCompact],
  );

  const chartHeight = (layout.height as number | undefined) ?? 320;

  return (
    <div className="surface-panel rounded-2xl p-4">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-[var(--text-primary)]">Modality Mix</h3>
      </div>
      {modalities.length > 0 ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${chartHeight}px` }} />
      ) : (
        <p className="text-sm text-[var(--text-muted)]">No modality data available.</p>
      )}
    </div>
  );
}

type DegreeRequirementsChartProps = {
  degrees: DomainCount[];
};

export function DegreeRequirementsChart({ degrees }: DegreeRequirementsChartProps) {
  const isCompact = useCompactScreen();
  const { baseVerticalBarLayout, degreeFill, hoverLabelFontColor, hoverLabelBgColor } = useChartVisuals();
  const plotData = useMemo<Data[]>(
    () => [
      {
        type: 'bar' as const,
        x: degrees.map((entry) => entry.label),
        y: degrees.map((entry) => entry.count),
        marker: { color: degreeFill },
        hovertemplate: '<b>%{x}</b><br>Postings: %{y}<extra></extra>',
        hoverlabel: { font: { color: hoverLabelFontColor }, bgcolor: hoverLabelBgColor },
      },
    ],
    [degreeFill, degrees, hoverLabelBgColor, hoverLabelFontColor],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      ...baseVerticalBarLayout,
      margin: isCompact ? { l: 40, r: 12, t: 40, b: 60 } : baseVerticalBarLayout.margin,
      xaxis: {
        ...(baseVerticalBarLayout.xaxis ?? {}),
        tickangle: isCompact ? -15 : baseVerticalBarLayout.xaxis?.tickangle,
      },
      height: isCompact ? 260 : 320,
    }),
    [baseVerticalBarLayout, isCompact],
  );

  const chartHeight = (layout.height as number | undefined) ?? 320;

  return (
    <div className="surface-panel rounded-2xl p-4">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-[var(--text-primary)]">Degree Requirements</h3>
      </div>
      {degrees.length > 0 ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${chartHeight}px` }} />
      ) : (
        <p className="text-sm text-[var(--text-muted)]">No degree data available.</p>
      )}
    </div>
  );
}

type YoeDistributionChartProps = {
  buckets: DomainCount[];
  height?: number;
};

export function YoeDistributionChart({ buckets, height }: YoeDistributionChartProps) {
  const isCompact = useCompactScreen();
  const { baseVerticalBarLayout, yoeFill, hoverLabelFontColor, hoverLabelBgColor } = useChartVisuals();
  const plotData = useMemo<Data[]>(
    () => [
      {
        type: 'bar' as const,
        x: buckets.map((entry) => entry.label),
        y: buckets.map((entry) => entry.count),
        marker: { color: yoeFill },
        hovertemplate: 'Min YOE %{x}<br>Postings: %{y}<extra></extra>',
        hoverlabel: { font: { color: hoverLabelFontColor }, bgcolor: hoverLabelBgColor },
      },
    ],
    [buckets, hoverLabelBgColor, hoverLabelFontColor, yoeFill],
  );

  const layout = useMemo<Partial<Layout>>(
    () => {
      const desiredHeight = height ?? 320;
      const finalHeight = isCompact ? Math.max(280, Math.min(360, desiredHeight)) : desiredHeight;
      return {
        ...baseVerticalBarLayout,
        margin: isCompact ? { l: 50, r: 12, t: 40, b: 70 } : baseVerticalBarLayout.margin,
        height: finalHeight,
        xaxis: {
          ...(baseVerticalBarLayout.xaxis ?? {}),
          title: { text: 'Min years of experience' },
          tickangle: isCompact ? -10 : baseVerticalBarLayout.xaxis?.tickangle,
        },
        yaxis: {
          ...(baseVerticalBarLayout.yaxis ?? {}),
          title: { text: 'Postings' },
        },
      };
    },
    [baseVerticalBarLayout, height, isCompact],
  );

  const chartHeight = (layout.height as number | undefined) ?? 320;

  return (
    <div className="surface-panel rounded-2xl p-4">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-[var(--text-primary)]">YOE Distribution</h3>
      </div>
      {buckets.some((entry) => entry.count > 0) ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${chartHeight}px` }} />
      ) : (
        <p className="text-sm text-[var(--text-muted)]">No experience data yet.</p>
      )}
    </div>
  );
}

export function LanguageBeeswarmPlot({ samples }: BeeswarmPlotProps) {
  const isCompact = useCompactScreen();
  const { beeswarmPalette, baseBoxLayout, hoverLabelFontColor, hoverLabelBgColor } = useChartVisuals();
  const { positions, labels, colorByLabel } = useMemo(() => {
    const uniqueLabels = Array.from(new Set(samples.map((sample) => sample.label)));
    const labelIndex = new Map(uniqueLabels.map((label, idx) => [label, idx]));
    const colorMap = new Map<string, string>();
    uniqueLabels.forEach((label, idx) => {
      colorMap.set(label, beeswarmPalette[idx % beeswarmPalette.length]);
    });

    const totalCounts = new Map<string, Map<number, number>>();
    for (const sample of samples) {
      const label = sample.label;
      const value = Number(sample.value);
      const labelMap = totalCounts.get(label) ?? new Map<number, number>();
      labelMap.set(value, (labelMap.get(value) ?? 0) + 1);
      totalCounts.set(label, labelMap);
    }

    const assignedCounts = new Map<string, Map<number, number>>();

    const positions = samples.map((sample) => {
      const label = sample.label;
      const value = Number(sample.value);
      const base = labelIndex.get(label) ?? 0;
      const total = totalCounts.get(label)?.get(value) ?? 1;
      const labelAssigned = assignedCounts.get(label) ?? new Map<number, number>();
      const assigned = labelAssigned.get(value) ?? 0;
      labelAssigned.set(value, assigned + 1);
      assignedCounts.set(label, labelAssigned);

      const spacing = 0.18;
      const offset = (assigned - (total - 1) / 2) * spacing;
      return base + offset;
    });

    return { positions, labels: uniqueLabels, colorByLabel: colorMap };
  }, [beeswarmPalette, samples]);

  const plotData = useMemo<Data[]>(
    () => [
      {
        type: 'scatter' as const,
        mode: 'markers' as const,
        x: positions,
        y: samples.map((sample) => sample.value),
        text: samples.map((sample) => sample.label),
        marker: {
          size: 7,
          opacity: 0.9,
          color: samples.map((sample) => colorByLabel.get(sample.label) ?? beeswarmPalette[0]),
        },
        hovertemplate: '<b>%{text}</b><br>Min YOE: %{y}<extra></extra>',
        hoverlabel: { font: { color: hoverLabelFontColor }, bgcolor: hoverLabelBgColor },
      },
    ],
    [beeswarmPalette, colorByLabel, hoverLabelBgColor, hoverLabelFontColor, positions, samples],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      ...baseBoxLayout,
      margin: isCompact ? { l: 40, r: 12, t: 32, b: 70 } : baseBoxLayout.margin,
      height: isCompact ? 320 : baseBoxLayout.height,
      xaxis: {
        tickmode: 'array',
        tickvals: labels.map((_, idx) => idx),
        ticktext: labels,
        tickangle: isCompact ? -45 : -35,
        showticklabels: true,
        range: [-0.7, labels.length - 0.3],
        showgrid: false,
        zeroline: false,
      },
    }),
    [baseBoxLayout, isCompact, labels],
  );

  const chartHeight = (layout.height as number | undefined) ?? 360;

  return (
    <div className="surface-panel rounded-2xl p-4">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-[var(--text-primary)]">YOE Beeswarm By Language</h3>
      </div>
      {samples.length > 0 ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${chartHeight}px` }} />
      ) : (
        <p className="text-sm text-[var(--text-muted)]">No languages with min YOE yet.</p>
      )}
    </div>
  );
}


type DomainYoeHeatmapProps = {
  data: DomainHeatmapData;
};

export function DomainYoeHeatmap({ data }: DomainYoeHeatmapProps) {
  const isCompact = useCompactScreen();
  const { heatmapColorscale, baseFont, hoverLabelFontColor, hoverLabelBgColor } = useChartVisuals();
  const plotData = useMemo<Data[]>(
    () => [
      {
        type: 'heatmap' as const,
        x: data.buckets,
        y: data.domains,
        z: data.z,
        colorscale: heatmapColorscale,
        colorbar: { title: { text: 'Postings', side: 'right' } },
        hovertemplate: '<b>%{y}</b><br>YOE %{x}: %{z}<extra></extra>',
        hoverlabel: { font: { color: hoverLabelFontColor }, bgcolor: hoverLabelBgColor },
      },
    ],
    [data, heatmapColorscale, hoverLabelBgColor, hoverLabelFontColor],
  );

  const layout = useMemo<Partial<Layout>>(
    () => {
      const autoHeight = Math.max(320, 28 * data.domains.length + 120);
      const finalHeight = isCompact ? Math.min(Math.max(320, autoHeight), 520) : autoHeight;
      return {
        margin: isCompact ? { l: 56, r: 16, t: 32, b: 48 } : { l: 140, r: 20, t: 40, b: 60 },
        paper_bgcolor: 'rgba(0,0,0,0)',
        plot_bgcolor: 'rgba(0,0,0,0)',
        font: baseFont,
        xaxis: {
          automargin: true,
          title: { text: 'Min years of experience' },
          titlefont: { size: isCompact ? 12 : 14 },
        },
        yaxis: {
          automargin: !isCompact,
          tickfont: { size: isCompact ? 10 : 12 },
        },
        height: finalHeight,
      };
    },
    [baseFont, data.domains.length, isCompact],
  );

  return (
    <div className="surface-panel rounded-2xl p-4">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-[var(--text-primary)]">YOE Heatmap By Domain</h3>
      </div>
      {data.domains.length > 0 ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${layout.height as number}px` }} />
      ) : (
        <p className="text-sm text-[var(--text-muted)]">Not enough YOE data to populate the heatmap.</p>
      )}
    </div>
  );
}

type DomainPopularityTrendChartProps = {
  series: DomainTrendSeries[];
  dates: string[];
};

export function DomainPopularityTrendChart({ series, dates }: DomainPopularityTrendChartProps) {
  const isCompact = useCompactScreen();
  const { chartPalette, baseFont, hoverLabelFontColor, hoverLabelBgColor, gridColor } = useChartVisuals();
  const MAX_TICKS = 10;
  const tickStep = Math.ceil(dates.length / MAX_TICKS);
  const tickDates = dates.filter((_, idx) => idx % tickStep === 0);

  const colorMap = useMemo(() => {
    const map = new Map<string, string>();
    const length = chartPalette.length;
    series.forEach((entry, index) => {
      if (!map.has(entry.domain)) {
        map.set(entry.domain, chartPalette[index % length]);
      }
    });
    return map;
  }, [chartPalette, series]);

  const legendEntries = useMemo(() => {
    const seen = new Set<string>();
    const entries: { domain: string; color: string }[] = [];
    series.forEach((entry) => {
      if (!seen.has(entry.domain)) {
        seen.add(entry.domain);
        entries.push({ domain: entry.domain, color: colorMap.get(entry.domain) ?? chartPalette[0] });
      }
    });
    return entries;
  }, [chartPalette, colorMap, series]);

  const plotData = useMemo<Data[]>(
    () =>
      series.map((entry) => {
        const baseColor = colorMap.get(entry.domain) ?? chartPalette[0];
        return {
          type: 'scatter' as const,
          mode: 'lines' as const,
          x: entry.x,
          y: entry.y,
          name: entry.domain,
          stackgroup: 'domain_trend',
          line: { color: baseColor, width: 1.5, shape: 'spline' },
          fill: 'tonexty' as const,
          fillcolor: baseColor,
          opacity: 0.65,
          hovertemplate: `<b>${entry.domain}</b><br>%{x}: %{y} postings<extra></extra>`,
          hoverlabel: { font: { color: hoverLabelFontColor }, bgcolor: hoverLabelBgColor },
        };
      }),
    [chartPalette, colorMap, hoverLabelBgColor, hoverLabelFontColor, series],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      margin: isCompact ? { l: 45, r: 12, t: 32, b: 60 } : { l: 60, r: 20, t: 40, b: 70 },
      paper_bgcolor: 'rgba(0,0,0,0)',
      plot_bgcolor: 'rgba(0,0,0,0)',
      font: baseFont,
      xaxis: {
        type: 'category',
        tickvals: isCompact ? [] : tickDates,
        ticktext: isCompact ? undefined : tickDates.map((d) => d.slice(5)),
        tickangle: isCompact ? -15 : -30,
        showticklabels: !isCompact,
        ticks: isCompact ? '' : undefined,
        gridcolor: gridColor,
        showgrid: true,
      },
      yaxis: { title: { text: 'Postings per day' }, rangemode: 'tozero', gridcolor: gridColor },
      showlegend: false,
      height: isCompact ? 300 : 360,
    }),
    [baseFont, gridColor, isCompact, tickDates],
  );

  return (
    <div className="surface-panel rounded-2xl p-4">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-[var(--text-primary)]">Domain Popularity Over Time</h3>
      </div>
      {series.length > 0 ? (
        <>
          <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${layout.height as number}px` }} />
          {legendEntries.length > 0 && (
            <div className="mt-3 overflow-x-auto pb-1">
              <div className="flex w-full min-w-full justify-center gap-4 text-xs font-semibold text-[var(--text-secondary)]">
                <div className="flex min-w-max items-center gap-4">
                  {legendEntries.map((entry) => (
                    <span key={entry.domain} className="inline-flex items-center gap-2 whitespace-nowrap">
                      <span className="h-2 w-2 rounded-full" style={{ backgroundColor: entry.color }} aria-hidden="true" />
                      {entry.domain}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          )}
        </>
      ) : (
        <p className="text-sm text-[var(--text-muted)]">Not enough daily data yet.</p>
      )}
    </div>
  );
}
