import { useMemo } from 'react';
import type { Job } from '../types/job';
import {
  DegreeRequirementsChart,
  DomainPopularityChart,
  DomainPopularityTrendChart,
  DomainYoeHeatmap,
  LanguageBeeswarmPlot,
  MinYoeBoxPlot,
  ModalityPopularityChart,
  YoeDistributionChart,
  type BoxStat,
  type DomainCount,
  type DomainHeatmapData,
  type DomainTrendSeries,
  type LanguageSample,
} from './InsightsCharts';

type JobsInsightsProps = {
  jobs: Job[];
};

const insightsPalette = ['#f2e9e1', '#286983', '#797593', '#9893a5', '#907aa9', '#b4637a', '#d7827e', '#56949f', '#ea9d34'] as const;
const primaryLanguageColor = insightsPalette[7];
const primaryDomainColor = insightsPalette[4];

const LANGUAGE_ALIAS_MAP: Record<string, string | null> = {
  html: null,
  css: null,
  javascript: 'JavaScript',
  js: 'JavaScript',
  'java script': 'JavaScript',
  typescript: 'TypeScript',
  ts: 'TypeScript',
};

const PRIORITY_LANGUAGES = ['Swift', 'Kotlin', 'Dart'];

function toTitleCase(value: string) {
  return value
    .toLowerCase()
    .replace(/(^|[\s-])([a-z])/g, (_, boundary: string, char: string) => `${boundary}${char.toUpperCase()}`);
}

function normalizeLanguage(raw?: string | null) {
  if (!raw) return undefined;
  const trimmed = raw.trim();
  if (!trimmed) return undefined;
  const lower = trimmed.toLowerCase();
  if (lower in LANGUAGE_ALIAS_MAP) {
    const alias = LANGUAGE_ALIAS_MAP[lower];
    return alias ?? undefined;
  }
  if (trimmed === trimmed.toLowerCase() || trimmed === trimmed.toUpperCase()) {
    return toTitleCase(trimmed);
  }
  return trimmed;
}

function computeMedian(values: number[]) {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((a, b) => a - b);
  const mid = Math.floor(sorted.length / 2);
  if (sorted.length % 2 === 0) {
    return (sorted[mid - 1] + sorted[mid]) / 2;
  }
  return sorted[mid];
}

function normalizeDomain(raw?: string | null) {
  if (!raw) return undefined;
  const trimmed = raw.trim();
  if (!trimmed) return undefined;
  if (trimmed.toLowerCase() === 'other') return undefined;
  return trimmed;
}

function normalizeModality(raw?: string | null) {
  if (!raw) return undefined;
  const trimmed = raw.trim();
  if (!trimmed) return undefined;
  return toTitleCase(trimmed);
}

export function JobsInsights({ jobs }: JobsInsightsProps) {
  const {
    languageBoxData,
    domainBoxData,
    domainPopularity,
    modalityPopularity,
    degreeCounts,
    yoeDistribution,
    domainSummaryHeight,
    languageSamples,
    domainHeatmap,
    domainTrendSeries,
    domainTrendDates,
  } = useMemo(() => {
    const languageYoe = new Map<string, number[]>();
    const domainYoe = new Map<string, number[]>();
    const domainCounts = new Map<string, number>();
    const domainBucketCounts = new Map<string, Map<number, number>>();
    const domainCountsByDate = new Map<string, Map<string, number>>();
    const modalityCounts = new Map<string, number>();
    const degreeCounts = new Map<string, number>();
    const overallYoeBuckets = new Map<number, number>();
    const languageSamples: LanguageSample[] = [];
    const MAX_BUCKET = 20;
    let observedMaxYoe = 0;

    const ensureArray = (map: Map<string, number[]>, key: string) => {
      if (!map.has(key)) {
        map.set(key, []);
      }
      return map.get(key)!;
    };

    const ensureBucketMap = (key: string) => {
      if (!domainBucketCounts.has(key)) {
        domainBucketCounts.set(key, new Map());
      }
      return domainBucketCounts.get(key)!;
    };

    const ensureDateMap = (key: string) => {
      if (!domainCountsByDate.has(key)) {
        domainCountsByDate.set(key, new Map());
      }
      return domainCountsByDate.get(key)!;
    };

    for (const job of jobs) {
      const domainKey = normalizeDomain(job.domain);
      const minYoe =
        typeof job.minYearsExperience === 'number' ? job.minYearsExperience : undefined;
      const modalityKey = normalizeModality(job.modality);
      const degreeKey = normalizeModality(job.minDegree);

      if (domainKey) {
        domainCounts.set(domainKey, (domainCounts.get(domainKey) ?? 0) + 1);
      }

      if (modalityKey) {
        modalityCounts.set(modalityKey, (modalityCounts.get(modalityKey) ?? 0) + 1);
      }

      if (degreeKey) {
        degreeCounts.set(degreeKey, (degreeCounts.get(degreeKey) ?? 0) + 1);
      }

      if (minYoe != null && domainKey) {
        observedMaxYoe = Math.max(observedMaxYoe, minYoe);
        ensureArray(domainYoe, domainKey).push(minYoe);

        const bucket = Math.min(Math.max(0, Math.floor(minYoe)), MAX_BUCKET);
        const bucketMap = ensureBucketMap(domainKey);
        bucketMap.set(bucket, (bucketMap.get(bucket) ?? 0) + 1);
        overallYoeBuckets.set(bucket, (overallYoeBuckets.get(bucket) ?? 0) + 1);

        for (const rawLang of job.languages ?? []) {
          const lang = normalizeLanguage(rawLang);
          if (!lang) continue;
          ensureArray(languageYoe, lang).push(minYoe);
          languageSamples.push({ label: lang, value: minYoe });
        }
      }

      const postedDateRaw = job.postedDate?.trim();
      if (postedDateRaw && domainKey) {
        const normalizedDate = postedDateRaw.split('T')[0] ?? postedDateRaw;
        const dateMap = ensureDateMap(normalizedDate);
        dateMap.set(domainKey, (dateMap.get(domainKey) ?? 0) + 1);
      }
    }

    const toBoxStats = (map: Map<string, number[]>) =>
      Array.from(map.entries())
        .map(
          ([label, values]): BoxStat => ({
            label,
            values,
            count: values.length,
            median: computeMedian(values),
          }),
        )
        .filter((entry) => entry.count > 0);

    const languageStats: BoxStat[] = toBoxStats(languageYoe).sort(
      (a, b) =>
        b.count - a.count || b.median - a.median || a.label.localeCompare(b.label),
    );
    const domainStats: BoxStat[] = toBoxStats(domainYoe).sort(
      (a, b) =>
        b.count - a.count || b.median - a.median || a.label.localeCompare(b.label),
    );

    const popularity: DomainCount[] = Array.from(domainCounts.entries())
      .map(([label, count]) => ({ label, count }))
      .sort((a, b) => b.count - a.count || a.label.localeCompare(b.label));

    const modalityPopularity: DomainCount[] = Array.from(modalityCounts.entries())
      .map(([label, count]) => ({ label, count }))
      .sort((a, b) => b.count - a.count || a.label.localeCompare(b.label));
    const degreePopularity: DomainCount[] = Array.from(degreeCounts.entries())
      .map(([label, count]) => ({ label, count }))
      .sort((a, b) => b.count - a.count || a.label.localeCompare(b.label));

    const selectTopLanguages = () => {
      const limit = 12;
      const top = languageStats.slice(0, limit);
      const seen = new Set(top.map((entry) => entry.label));
      for (const lang of PRIORITY_LANGUAGES) {
        if (seen.has(lang)) continue;
        const match = languageStats.find((entry) => entry.label === lang);
        if (match) {
          top.push(match);
          seen.add(lang);
        }
      }
      return top;
    };

    const baseLanguageStats = languageStats.slice(0, 12);
    const topLanguageStats = selectTopLanguages();
    const topLanguageLabels = topLanguageStats.map((entry) => entry.label);
    const filteredSamples = languageSamples.filter((sample) =>
      topLanguageLabels.includes(sample.label),
    );

    const domainHeatmapDomains = domainStats.slice(0, 10).map((entry) => entry.label);
    const maxBucketValue = Math.max(
      0,
      Math.min(MAX_BUCKET, Math.ceil(observedMaxYoe)),
    );
    const bucketValues = Array.from({ length: maxBucketValue + 1 }, (_, idx) => idx);
    const bucketLabels = bucketValues.map((bucket) =>
      bucket === MAX_BUCKET && observedMaxYoe > MAX_BUCKET ? `${bucket}+` : `${bucket}`,
    );
    const yoeDistribution = bucketValues.map((bucket, idx) => ({
      label: bucketLabels[idx],
      count: overallYoeBuckets.get(bucket) ?? 0,
    }));
    const heatmapZ = domainHeatmapDomains.map((domain) => {
      const map = domainBucketCounts.get(domain) ?? new Map();
      return bucketValues.map((bucket) => map.get(bucket) ?? 0);
    });
    const domainHeatmap: DomainHeatmapData = {
      domains: domainHeatmapDomains,
      buckets: bucketLabels,
      z: heatmapZ,
    };

    const sortedTrendDates = Array.from(domainCountsByDate.keys()).sort((a, b) =>
      a.localeCompare(b),
    );
    const topTrendDomains = domainStats.map((entry) => entry.label);
    const domainTrendSeries: DomainTrendSeries[] = topTrendDomains
      .map((domain) => {
        if (!domainCounts.has(domain)) return undefined;
        return {
          domain,
          x: sortedTrendDates,
          y: sortedTrendDates.map(
            (date) => domainCountsByDate.get(date)?.get(domain) ?? 0,
          ),
        };
      })
      .filter((entry): entry is DomainTrendSeries => Boolean(entry));

    const SUMMARY_CARD_HEIGHT = 320;
    const SUMMARY_GAP = 16;

    return {
      languageBoxData: baseLanguageStats,
      domainBoxData: domainStats,
      domainPopularity: popularity,
      modalityPopularity,
      degreeCounts: degreePopularity,
      yoeDistribution,
      domainSummaryHeight: SUMMARY_CARD_HEIGHT * 2 + SUMMARY_GAP,
      languageSamples: filteredSamples,
      domainHeatmap,
      domainTrendSeries,
      domainTrendDates: sortedTrendDates,
    };
  }, [jobs]);

  return (
    <div className="space-y-4">
      <DomainPopularityTrendChart series={domainTrendSeries} dates={domainTrendDates} />
      <div className="grid gap-4 lg:grid-cols-2">
        <MinYoeBoxPlot
          stats={languageBoxData}
          title="Min YOE By Language"
          color={primaryLanguageColor}
          emptyMessage="Not enough min YOE data tied to languages yet."
        />
        <MinYoeBoxPlot
          stats={domainBoxData}
          title="Min YOE By Domain"
          color={primaryDomainColor}
          emptyMessage="No posted domains with min YOE yet."
        />
      </div>
      <LanguageBeeswarmPlot samples={languageSamples} />
      <div className="grid gap-4 lg:grid-cols-2 items-stretch">
        <div className="h-full">
          <DomainPopularityChart
            domains={domainPopularity}
            totalJobs={jobs.length}
            height={domainSummaryHeight}
          />
        </div>
        <div className="grid h-full gap-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <ModalityPopularityChart modalities={modalityPopularity} />
            <DegreeRequirementsChart degrees={degreeCounts} />
          </div>
          <YoeDistributionChart buckets={yoeDistribution} />

        </div>
      </div>
      <DomainYoeHeatmap data={domainHeatmap} />
    </div>
  );
}
