import { useMemo } from 'react';
import type { Data, Layout } from 'plotly.js';
import Plotly from 'plotly.js/lib/core';
import bar from 'plotly.js/lib/bar';
import box from 'plotly.js/lib/box';
import heatmap from 'plotly.js/lib/heatmap';
import scatter from 'plotly.js/lib/scatter';
import createPlotlyComponent from 'react-plotly.js/factory';

Plotly.register([box, bar, scatter, heatmap]);

const Plot = createPlotlyComponent(Plotly);

const chartPalette = ['#f2e9e1', '#286983', '#56949f', '#797593', '#9893a5', '#907aa9', '#b4637a', '#d7827e', '#ea9d34'] as const;
const beeswarmPalette = chartPalette.filter((color) => color !== '#f2e9e1');
const basePlotConfig = { displayModeBar: false, responsive: true } as const;
const baseFont = { family: 'Inter, system-ui, sans-serif', color: '#000000' };
const domainFill = chartPalette[5];
const modalityFill = chartPalette[7];
const degreeFill = chartPalette[4];
const yoeFill = chartPalette[2];
const hoverLabelFontColor = chartPalette[4];
const hoverLabelBgColor = chartPalette[0];
const heatmapColorscale = (() => {
  const steps = [
    chartPalette[0],
    chartPalette[1],
    chartPalette[2],
    chartPalette[3],
    chartPalette[4],
    chartPalette[5],
    chartPalette[6],
    chartPalette[7],
    chartPalette[8],
  ];
  const segment = 1 / (steps.length - 1);
  return steps.map((color, index) => [Number((index * segment).toFixed(4)), color]) as Array<[number, string]>;
})();

const baseBoxLayout: Partial<Layout> = {
  margin: { l: 60, r: 20, t: 40, b: 80 },
  paper_bgcolor: 'rgba(0,0,0,0)',
  plot_bgcolor: 'rgba(0,0,0,0)',
  showlegend: false,
  hovermode: 'closest',
  font: baseFont,
  yaxis: {
    title: { text: 'Min years of experience' },
    zeroline: false,
    gridcolor: '#e2e8f0',
  },
  xaxis: {
    automargin: true,
    tickangle: -35,
  },
  height: 360,
};

const baseHorizontalBarLayout: Partial<Layout> = {
  margin: { l: 160, r: 20, t: 60, b: 40 },
  paper_bgcolor: 'rgba(0,0,0,0)',
  plot_bgcolor: 'rgba(0,0,0,0)',
  font: baseFont,
  xaxis: { gridcolor: '#e2e8f0' },
  yaxis: { automargin: true, autorange: 'reversed' as const },
};

const baseVerticalBarLayout: Partial<Layout> = {
  margin: { l: 50, r: 20, t: 60, b: 80 },
  paper_bgcolor: 'rgba(0,0,0,0)',
  plot_bgcolor: 'rgba(0,0,0,0)',
  font: baseFont,
  xaxis: { automargin: true, tickangle: -35 },
  yaxis: { gridcolor: '#e2e8f0', rangemode: 'tozero' },
};

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

export function MinYoeBoxPlot({
  stats,
  title,
  color,
  emptyMessage,
  hoverLabelColor = hoverLabelFontColor,
  hoverLabelBg = hoverLabelBgColor,
}: MinYoeBoxPlotProps) {
  const plotData = useMemo<Data[]>(
    () =>
      stats.map((entry) => ({
        type: 'box' as const,
        y: entry.values,
        name: entry.label,
        boxpoints: 'suspectedoutliers' as const,
        marker: { color },
        hovertemplate: `<b>${entry.label}</b><br>Min YOE: %{y}<extra></extra>`,
        hoverlabel: { font: { color: hoverLabelColor }, bgcolor: hoverLabelBg },
      })),
    [stats, color, hoverLabelColor, hoverLabelBg],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      ...baseBoxLayout,
    }),
    [],
  );

  const chartHeight = (layout.height as number | undefined) ?? 360;

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-4 shadow-sm">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-slate-900">{title}</h3>
      </div>
      {plotData.length > 0 ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${chartHeight}px` }} />
      ) : (
        <p className="text-sm text-slate-500">{emptyMessage}</p>
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
    [domains],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      ...baseHorizontalBarLayout,
      height:
        height ??
        Math.max(
          720,
          Math.min(900, 48 * domains.length + 260),
        ),
    }),
    [domains.length, height],
  );

  return (
    <div
      className="flex h-full flex-col rounded-2xl border border-slate-100 bg-white p-4 shadow-sm"
      style={height ? { minHeight: height } : undefined}
    >
      <div className="mb-3 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-900">Domain popularity</h3>
        </div>
        <span className="text-xs font-semibold uppercase text-black">{totalJobs} jobs</span>
      </div>
      {domains.length > 0 ? (
        <div className="flex-1" style={{ minHeight: 0 }}>
          <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: '100%' }} />
        </div>
      ) : (
        <p className="text-sm text-slate-500">No domains available yet.</p>
      )}
    </div>
  );
}

type ModalityPopularityChartProps = {
  modalities: DomainCount[];
};

export function ModalityPopularityChart({ modalities }: ModalityPopularityChartProps) {
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
    [modalities],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      ...baseVerticalBarLayout,
      height: 320,
    }),
    [],
  );

  const chartHeight = (layout.height as number | undefined) ?? 320;

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-4 shadow-sm">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-slate-900">Modality mix</h3>
      </div>
      {modalities.length > 0 ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${chartHeight}px` }} />
      ) : (
        <p className="text-sm text-slate-500">No modality data available.</p>
      )}
    </div>
  );
}

type DegreeRequirementsChartProps = {
  degrees: DomainCount[];
};

export function DegreeRequirementsChart({ degrees }: DegreeRequirementsChartProps) {
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
    [degrees],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      ...baseVerticalBarLayout,
      height: 320,
    }),
    [],
  );

  const chartHeight = (layout.height as number | undefined) ?? 320;

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-4 shadow-sm">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-slate-900">Degree requirements</h3>
      </div>
      {degrees.length > 0 ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${chartHeight}px` }} />
      ) : (
        <p className="text-sm text-slate-500">No degree data available.</p>
      )}
    </div>
  );
}

type YoeDistributionChartProps = {
  buckets: DomainCount[];
  height?: number;
};

export function YoeDistributionChart({ buckets, height }: YoeDistributionChartProps) {
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
    [buckets],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      ...baseVerticalBarLayout,
      height: height ?? 320,
      xaxis: { ...(baseVerticalBarLayout.xaxis ?? {}), title: { text: 'Min years of experience' } },
      yaxis: { ...(baseVerticalBarLayout.yaxis ?? {}), title: { text: 'Postings' } },
    }),
    [height],
  );

  const chartHeight = (layout.height as number | undefined) ?? 320;

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-4 shadow-sm">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-slate-900">YOE distribution</h3>
      </div>
      {buckets.some((entry) => entry.count > 0) ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${chartHeight}px` }} />
      ) : (
        <p className="text-sm text-slate-500">No experience data yet.</p>
      )}
    </div>
  );
}

export function LanguageBeeswarmPlot({ samples }: BeeswarmPlotProps) {
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
  }, [samples]);

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
          line: { color: chartPalette[0], width: 0.5 },
        },
        hovertemplate: '<b>%{text}</b><br>Min YOE: %{y}<extra></extra>',
        hoverlabel: { font: { color: hoverLabelFontColor }, bgcolor: hoverLabelBgColor },
      },
    ],
    [positions, samples],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      ...baseBoxLayout,
      xaxis: {
        tickmode: 'array',
        tickvals: labels.map((_, idx) => idx),
        ticktext: labels,
        tickangle: -35,
        range: [-0.7, labels.length - 0.3],
        showgrid: false,
        zeroline: false,
      },
    }),
    [labels],
  );

  const chartHeight = (layout.height as number | undefined) ?? 360;

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-4 shadow-sm">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-slate-900">YOE beeswarm by language</h3>
      </div>
      {samples.length > 0 ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${chartHeight}px` }} />
      ) : (
        <p className="text-sm text-slate-500">No languages with min YOE yet.</p>
      )}
    </div>
  );
}


type DomainYoeHeatmapProps = {
  data: DomainHeatmapData;
};

export function DomainYoeHeatmap({ data }: DomainYoeHeatmapProps) {
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
    [data],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      margin: { l: 160, r: 20, t: 40, b: 60 },
      paper_bgcolor: 'rgba(0,0,0,0)',
      plot_bgcolor: 'rgba(0,0,0,0)',
      font: baseFont,
      xaxis: { automargin: true },
      yaxis: { automargin: true },
      height: Math.max(320, 28 * data.domains.length + 120),
    }),
    [data.domains.length],
  );

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-4 shadow-sm">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-slate-900">YOE heatmap by domain</h3>
      </div>
      {data.domains.length > 0 ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${layout.height as number}px` }} />
      ) : (
        <p className="text-sm text-slate-500">Not enough YOE data to populate the heatmap.</p>
      )}
    </div>
  );
}

type DomainPopularityTrendChartProps = {
  series: DomainTrendSeries[];
  dates: string[];
};

export function DomainPopularityTrendChart({ series, dates }: DomainPopularityTrendChartProps) {
  const colorMap = useMemo(() => {
    const map = new Map<string, string>();
    const length = chartPalette.length;
    series.forEach((entry, index) => {
      if (!map.has(entry.domain)) {
        map.set(entry.domain, chartPalette[index % length]);
      }
    });
    return map;
  }, [series]);

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
    [series, colorMap],
  );

  const layout = useMemo<Partial<Layout>>(
    () => ({
      margin: { l: 60, r: 20, t: 40, b: 60 },
      paper_bgcolor: 'rgba(0,0,0,0)',
      plot_bgcolor: 'rgba(0,0,0,0)',
      font: baseFont,
      xaxis: {
        type: 'category',
        tickvals: dates,
        ticktext: dates.map((date) => date.slice(5)),
        tickangle: -30,
      },
      yaxis: { title: { text: 'Postings per day' }, rangemode: 'tozero' },
      legend: { orientation: 'h', yanchor: 'bottom', y: 1.02, x: 0, traceorder: 'normal' },
      height: 360,
    }),
    [dates],
  );

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-4 shadow-sm">
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-slate-900">Domain popularity over time</h3>
      </div>
      {series.length > 0 ? (
        <Plot data={plotData} layout={layout} config={basePlotConfig} style={{ width: '100%', height: `${layout.height as number}px` }} />
      ) : (
        <p className="text-sm text-slate-500">Not enough daily data yet.</p>
      )}
    </div>
  );
}
