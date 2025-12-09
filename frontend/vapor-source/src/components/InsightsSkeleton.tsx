const baseCard = 'rounded-2xl border border-slate-100 bg-white p-4 shadow-sm';
const shimmer = 'relative overflow-hidden';
const shimmerInner = 'absolute inset-0 -translate-x-full animate-[shimmer_1.6s_infinite] bg-gradient-to-r from-transparent via-white/60 to-transparent';

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
      <div className="mt-4 h-full rounded-xl bg-slate-100" />
      <div className={shimmerInner} aria-hidden />
    </div>
  );
}

function SkeletonHeader({ title }: { title: string }) {
  return (
    <div>
      <div className="h-4 w-32 rounded bg-slate-200" />
      <div className="mt-2 h-5 w-64 rounded bg-slate-200" aria-hidden />
      <span className="sr-only">{title} loading</span>
    </div>
  );
}
