import { Suspense, lazy } from 'react';
import { JobsTable } from './components/JobsTable';
import { useJobsSnapshot } from './hooks/useJobsSnapshot';

const DEFAULT_PAGE_SIZE = 15;

const JobsInsights = lazy(() =>
  import('./components/JobsInsights').then((module) => ({ default: module.JobsInsights })),
);

function App() {
  const { jobs, loading, error } = useJobsSnapshot(DEFAULT_PAGE_SIZE);

  return (
    <main className="min-h-screen bg-gradient-to-b from-[rgba(255,255,252,0.25)] via-white to-white px-4 py-10 text-slate-900 sm:px-6 lg:px-8">
      <section className="mx-auto w-full max-w-6xl rounded-2xl border border-slate-200 bg-white p-8 shadow-[0_30px_80px_rgba(248,235,222,0.32)]">
        <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Vapor Source</p>
        <h1 className="mt-3 text-3xl font-semibold text-slate-900 sm:text-4xl">SWE Job Analytics</h1>
        <p className="mt-3 max-w-3xl text-base text-slate-600">
          Vapor Source ingests WorkSource postings and intelligently parses descriptions to determine
          the minimum years of experience, software domain, work modality, and technology used for the role.
        </p>
      </section>

      {!loading && !error && jobs.length > 0 && (
        <section className="mx-auto mt-6 w-full max-w-6xl rounded-2xl border border-slate-200 bg-white p-6 shadow-[0_30px_80px_rgba(248,235,222,0.32)] sm:p-8">
          <div className="flex flex-col gap-1">
            <h2 className="text-xl font-semibold text-slate-900">Snapshot insights</h2>
          </div>
          <div className="mt-5">
            <Suspense
              fallback={
                <p className="rounded-lg bg-slate-50 px-4 py-3 text-sm font-medium text-slate-600">
                  Loading charts…
                </p>
              }
            >
              <JobsInsights jobs={jobs} />
            </Suspense>
          </div>
        </section>
      )}
      <section className="mx-auto mt-6 w-full max-w-6xl rounded-2xl border border-slate-200 bg-white p-6 shadow-[0_30px_80px_rgba(248,235,222,0.32)] sm:p-8">
        <div className="flex flex-col gap-1">
          <h2 className="text-xl font-semibold text-slate-900">Job Listings</h2>
        </div>

        {loading && (
          <p className="mt-6 rounded-lg bg-slate-50 px-4 py-3 text-sm font-medium text-slate-600">Loading snapshot…</p>
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
