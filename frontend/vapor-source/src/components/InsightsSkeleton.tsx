const baseCard = 'rounded-2xl border border-[var(--skeleton-card-border)] bg-[var(--skeleton-card-bg)] p-4 shadow-sm';
const shimmer = 'relative overflow-hidden';
const shimmerInner =
  'absolute inset-0 -translate-x-full animate-[shimmer_1.6s_infinite] bg-gradient-to-r from-transparent via-[var(--skeleton-shimmer)] to-transparent';

export function InsightsSkeleton() {
  return (
    <div className="space-y-4">
      <SkeletonCard title="Domain popularity over time" height={360} />
      <div className="grid gap-4 lg:grid-cols-2">
        <SkeletonCard title="Min YOE by language" height={360} />
        <SkeletonCard title="Min YOE by domain" height={360} />
      </div>
      <SkeletonCard title="YOE beeswarm by language" height={360} />
      <div className="grid gap-4 lg:grid-cols-2 items-stretch">
        <SkeletonCard title="Domain popularity" height={360} />
        <div className="grid gap-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <SkeletonCard title="Modality mix" height={320} />
            <SkeletonCard title="Degree requirements" height={320} />
          </div>
          <SkeletonCard title="YOE distribution" height={320} />
        </div>
      </div>
      <SkeletonCard title="YOE heatmap by domain" height={360} />
    </div>
  );
}

function SkeletonCard({ title, height }: { title: string; height: number }) {
  return (
    <div className={`${baseCard} ${shimmer}`} style={{ minHeight: `${height}px` }} aria-label={`${title} loading`}>
      <SkeletonHeader title={title} />
      <div className="mt-4 h-full rounded-xl bg-[var(--skeleton-chart-bg)]" />
      <div className={shimmerInner} aria-hidden />
    </div>
  );
}

function SkeletonHeader({ title }: { title: string }) {
  return (
    <div>
      <div className="h-4 w-32 rounded bg-[var(--skeleton-line-bg)]" />
      <div className="mt-2 h-5 w-64 rounded bg-[var(--skeleton-line-bg)]" aria-hidden />
      <span className="sr-only">{title} loading</span>
    </div>
  );
}
