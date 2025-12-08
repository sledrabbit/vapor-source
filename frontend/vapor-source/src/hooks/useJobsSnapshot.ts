import { useEffect, useMemo, useState } from 'react';
import type { Job } from '../types/job';

const SNAPSHOT_BASE_URL = '/snapshots/';
const SNAPSHOT_MANIFEST_URL = `${SNAPSHOT_BASE_URL}snapshot-manifest.json`;

type SnapshotManifestEntry = {
  date: string;
  key: string;
  jobCount: number;
  updatedAt: string;
};

async function fetchManifestEntries(): Promise<SnapshotManifestEntry[]> {
  const res = await fetch(SNAPSHOT_MANIFEST_URL);
  if (!res.ok) throw new Error('Manifest load failed');
  const manifest: unknown = await res.json();

  if (!Array.isArray(manifest)) {
    throw new Error('Invalid manifest payload');
  }

  return (manifest as SnapshotManifestEntry[]).sort((a, b) => b.date.localeCompare(a.date));
}

async function fetchJobs(url: string): Promise<Job[]> {
  const res = await fetch(url);
  if (!res.ok) throw new Error('Snapshot load failed');
  const text = await res.text();
  return text
    .trim()
    .split('\n')
    .map((line) => JSON.parse(line) as Job);
}

export function useJobsSnapshot(targetCount = 10) {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const controller = new AbortController();

    async function load() {
      try {
        setLoading(true);
        const manifestEntries = await fetchManifestEntries();
        if (manifestEntries.length === 0) {
          throw new Error('No snapshot entries found');
        }

        const aggregated: Job[] = [];

        for (const entry of manifestEntries) {
          const url = `${SNAPSHOT_BASE_URL}${entry.key}`;
          const snapshotJobs = await fetchJobs(url);
          aggregated.push(...snapshotJobs);
          if (aggregated.length >= targetCount) {
            break;
          }
        }

        aggregated.sort((a, b) => {
          const dateDiff = b.postedDate.localeCompare(a.postedDate);
          if (dateDiff !== 0) return dateDiff;
          return (b.postedTime ?? '').localeCompare(a.postedTime ?? '');
        });

        if (!controller.signal.aborted) setJobs(aggregated);
      } catch (err) {
        if (!controller.signal.aborted)
          setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }

    load();
    return () => controller.abort();
  }, []);

  const latest = useMemo(() => jobs.slice(0, targetCount), [jobs, targetCount]);

  return { jobs, latest, loading, error };
}
