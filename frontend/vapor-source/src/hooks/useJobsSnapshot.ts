import { useEffect, useMemo, useState } from 'react';
import type { Job } from '../types/job';

const SNAPSHOT_BASE_URL = '/snapshots/';
const SNAPSHOT_MANIFEST_URL = `${SNAPSHOT_BASE_URL}snapshot-manifest.json`;
const MS_PER_DAY = 24 * 60 * 60 * 1000;

type SnapshotManifestEntry = {
  date: string;
  key: string;
  jobCount: number;
  updatedAt: string;
};

async function fetchManifestEntries(signal?: AbortSignal): Promise<SnapshotManifestEntry[]> {
  const res = await fetch(SNAPSHOT_MANIFEST_URL, { signal });
  if (!res.ok) throw new Error('Manifest load failed');
  const manifest: unknown = await res.json();

  if (!Array.isArray(manifest)) {
    throw new Error('Invalid manifest payload');
  }

  return (manifest as SnapshotManifestEntry[]).sort((a, b) => b.date.localeCompare(a.date));
}

async function fetchJobs(url: string, signal?: AbortSignal): Promise<Job[]> {
  const res = await fetch(url, { signal });
  if (!res.ok) throw new Error('Snapshot load failed');
  const text = await res.text();
  return text
    .trim()
    .split('\n')
    .map((line) => JSON.parse(line) as Job);
}

function parseSnapshotDate(dateStr: string) {
  return new Date(`${dateStr}T00:00:00Z`);
}

function getWindowDates(days: number) {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const cutoff = new Date(today);
  cutoff.setDate(today.getDate() - days);
  return { today, cutoff };
}

export function useJobsSnapshot(targetCount = 10, backgroundDays = 30) {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [error, setError] = useState<string>();
  const [loadingLatest, setLoadingLatest] = useState(true);
  const [loadingAll, setLoadingAll] = useState(true);
  const [availableDays, setAvailableDays] = useState(0);
  const [fetchedDays, setFetchedDays] = useState(0);
  const [coveredDays, setCoveredDays] = useState(0);

  useEffect(() => {
    const controller = new AbortController();

    async function load() {
      try {
        setLoadingLatest(true);
        setLoadingAll(true);
        setError(undefined);
        setJobs([]);
        setAvailableDays(0);
        setFetchedDays(0);
        setCoveredDays(0);

        const manifestEntries = await fetchManifestEntries(controller.signal);
        if (manifestEntries.length === 0) {
          throw new Error('No snapshot entries found');
        }

        setAvailableDays(manifestEntries.length);
        const { today, cutoff } = getWindowDates(backgroundDays);

        const entriesInWindow: SnapshotManifestEntry[] = [];
        for (const entry of manifestEntries) {
          const entryDate = parseSnapshotDate(entry.date);
          if (entryDate < cutoff && entriesInWindow.length > 0) break;
          entriesInWindow.push(entry);
        }

        const entriesToFetch = entriesInWindow.length > 0 ? entriesInWindow : manifestEntries.slice(0, backgroundDays);

        const aggregated: Job[] = [];
        let maxDaysCovered = 0;
        let successfulDays = 0;
        let lastError: Error | undefined;

        const tableEntries: SnapshotManifestEntry[] = [];
        let accumulatedJobs = 0;
        for (const entry of entriesToFetch) {
          if (accumulatedJobs >= targetCount) break;
          tableEntries.push(entry);
          accumulatedJobs += entry.jobCount ?? 0;
        }
        const remainingEntries = entriesToFetch.slice(tableEntries.length);

        const fetchEntries = (entries: SnapshotManifestEntry[]) =>
          Promise.allSettled(
            entries.map(async (entry) => {
              const normalizedKey = entry.key.replace(/^\/?snapshots\//, '');
              const url = `${SNAPSHOT_BASE_URL}${normalizedKey}`;
              const snapshotJobs = await fetchJobs(url, controller.signal);
              return { entry, snapshotJobs };
            }),
          );

        const processResults = (
          results: PromiseSettledResult<{ entry: SnapshotManifestEntry; snapshotJobs: Job[] }>[],
        ) => {
          const fulfilled = results.filter(
            (result): result is PromiseFulfilledResult<{ entry: SnapshotManifestEntry; snapshotJobs: Job[] }> =>
              result.status === 'fulfilled',
          );
          const rejected = results.filter(
            (result): result is PromiseRejectedResult => result.status === 'rejected',
          );

          if (rejected.length > 0) {
            const reason = rejected[rejected.length - 1].reason;
            lastError = reason instanceof Error ? reason : new Error(String(reason));
          }

          for (const result of fulfilled) {
            const { entry, snapshotJobs } = result.value;
            aggregated.push(...snapshotJobs.filter((job) => job.IsSoftwareEngineerRelated));
            successfulDays += 1;
            const entryDate = parseSnapshotDate(entry.date);
            const daysCovered = Math.max(0, Math.round((today.getTime() - entryDate.getTime()) / MS_PER_DAY));
            maxDaysCovered = Math.max(maxDaysCovered, daysCovered);
          }
        };

        if (tableEntries.length > 0) {
          const tableResults = await fetchEntries(tableEntries);
          processResults(tableResults);

          aggregated.sort((a, b) => {
            const dateDiff = b.postedDate.localeCompare(a.postedDate);
            if (dateDiff !== 0) return dateDiff;
            return (b.postedTime ?? '').localeCompare(a.postedTime ?? '');
          });

          if (!controller.signal.aborted) {
            setJobs([...aggregated]);
            setFetchedDays(successfulDays);
            setCoveredDays(maxDaysCovered);
            setLoadingLatest(false);
            if (aggregated.length === 0 && lastError) {
              setError(lastError.message);
            }
          }
        } else {
          setLoadingLatest(false);
        }

        if (remainingEntries.length > 0) {
          const backgroundResults = await fetchEntries(remainingEntries);
          processResults(backgroundResults);
        }

        aggregated.sort((a, b) => {
          const dateDiff = b.postedDate.localeCompare(a.postedDate);
          if (dateDiff !== 0) return dateDiff;
          return (b.postedTime ?? '').localeCompare(a.postedTime ?? '');
        });

        if (!controller.signal.aborted) {
          setJobs([...aggregated]);
          setFetchedDays(successfulDays);
          setCoveredDays(maxDaysCovered);
          if (aggregated.length === 0 && lastError) {
            setError(lastError.message);
          }
        }
      } catch (err) {
        if (!controller.signal.aborted) {
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        if (!controller.signal.aborted) {
          setLoadingLatest(false);
          setLoadingAll(false);
        }
      }
    }

    load();
    return () => controller.abort();
  }, [backgroundDays, targetCount]);

  const latest = useMemo(() => jobs.slice(0, targetCount), [jobs, targetCount]);

  return {
    jobs,
    latest,
    loading: loadingLatest,
    loadingLatest,
    loadingAll,
    error,
    availableDays,
    fetchedDays,
    backgroundDays,
    coveredDays,
  };
}
