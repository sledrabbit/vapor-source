import { Suspense, lazy } from 'react';
import { JobsTable } from './components/JobsTable';
import { useJobsSnapshot } from './hooks/useJobsSnapshot';
import { InsightsSkeleton } from './components/InsightsSkeleton';
import { ThemeToggle } from './components/ThemeToggle';

const DEFAULT_PAGE_SIZE = 10;

const JobsInsights = lazy(() =>
  import('./components/JobsInsights').then((module) => ({ default: module.JobsInsights })),
);

function App() {
  const { jobs, loading, error } = useJobsSnapshot(DEFAULT_PAGE_SIZE);

  return (
    <main
      className="min-h-screen px-4 py-10 text-[var(--text-primary)] sm:px-6 lg:px-8"
      style={{ background: 'var(--background-app)' }}
    >
      <section className="surface-card mx-auto w-full max-w-6xl rounded-2xl p-8">
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <p className="text-xs font-semibold uppercase tracking-[0.3em] text-[var(--text-secondary)]">Vapor Source</p>
            <ThemeToggle />
          </div>
          <div>
            <h1 className="text-3xl font-semibold text-[var(--text-primary)] sm:text-4xl">SWE Job Analytics</h1>
            <p className="mt-3 max-w-3xl text-base text-[var(--text-secondary)]">
              Vapor Source ingests WorkSource postings and intelligently parses descriptions to determine
              the minimum years of experience, software domain, work modality, and technology used for the role.
            </p>
          </div>
        </div>
      </section>

      <section className="surface-card mx-auto mt-6 w-full max-w-6xl rounded-2xl p-6 sm:p-8">
        <div className="flex flex-col gap-1">
          <h2 className="text-xl font-semibold text-[var(--text-primary)]">Snapshot Insights</h2>
          <p className="text-sm text-[var(--text-muted)]">Aggregated analytics from the most recent 30-day snapshot window.</p>
        </div>
        <div className="mt-5">
          {loading && <InsightsSkeleton />}
          {!loading && error && (
            <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm font-medium text-rose-600">
              Failed to load insights: <span className="font-normal text-rose-500">{error}</span>
            </p>
          )}
          {!loading && !error && jobs.length === 0 && (
            <p className="rounded-lg bg-[var(--surface-muted)] px-4 py-3 text-sm font-medium text-[var(--text-secondary)]">
              No snapshot data yet.
            </p>
          )}
          {!loading && !error && jobs.length > 0 && (
            <Suspense fallback={<InsightsSkeleton />}>
              <JobsInsights jobs={jobs} />
            </Suspense>
          )}
        </div>
      </section>
      <section className="surface-card mx-auto mt-6 w-full max-w-6xl rounded-2xl p-6 sm:p-8">
        <div className="flex flex-col gap-1">
          <h2 className="text-xl font-semibold text-[var(--text-primary)]">Job Listings</h2>
          <p className="text-sm text-[var(--text-muted)]">Newest entries pulled across snapshot files, sorted by posted date.</p>
        </div>

        {loading && (
          <p className="mt-6 rounded-lg bg-[var(--surface-muted)] px-4 py-3 text-sm font-medium text-[var(--text-secondary)]">
            Loading snapshot…
          </p>
        )}
        {error && !loading && (
          <p className="mt-6 rounded-lg bg-rose-50 px-4 py-3 text-sm font-medium text-rose-600">
            Failed to load snapshot: <span className="font-normal text-rose-500">{error}</span>
          </p>
        )}

        {!loading && !error && <JobsTable jobs={jobs} pageSize={DEFAULT_PAGE_SIZE} />}
      </section>
    </main>
  );
}

export default App;
