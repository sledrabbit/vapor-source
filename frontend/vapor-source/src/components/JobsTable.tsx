import { createColumnHelper, flexRender, getCoreRowModel, useReactTable, type ColumnDef } from '@tanstack/react-table';
import type { Job } from '../types/job';

const columnHelper = createColumnHelper<Job>();

const lineClampStyle = (lines: number) => ({
  display: '-webkit-box',
  WebkitLineClamp: lines,
  WebkitBoxOrient: 'vertical' as const,
  overflow: 'hidden',
});

type ClampedTextProps = {
  text?: string;
  lines?: number;
  className?: string;
  fallback?: string;
};

const ClampedText = ({
  text,
  lines = 2,
  className = 'text-slate-700',
  fallback = '—',
}: ClampedTextProps) => {
  const value = text && text.trim().length ? text : fallback;
  return (
    <p className={className} style={lineClampStyle(lines)} title={value}>
      {value}
    </p>
  );
};

const columns: ColumnDef<Job, any>[] = [
  columnHelper.accessor('postedDate', {
    header: 'Posted',
  }),
  columnHelper.accessor('title', {
    header: 'Role',
    cell: (info) => {
      const job = info.row.original;
      return (
        <div className="flex flex-col gap-1">
          <a
            href={job.url}
            target="_blank"
            rel="noreferrer"
            className="font-semibold text-slate-900 transition hover:text-sky-600"
            style={lineClampStyle(2)}
            title={info.getValue() ?? ''}
          >
            {info.getValue()}
          </a>
          <ClampedText className="text-sm text-slate-500" lines={1} text={job.company} />
        </div>
      );
    },
  }),
  // columnHelper.accessor('company', {
  //   header: 'Company',
  //   cell: (info) => info.getValue(),
  // }),
  columnHelper.accessor('location', {
    header: 'Location',
    cell: (info) => info.getValue() || '—',
  }),
  columnHelper.accessor('modality', {
    header: 'Modality',
    cell: (info) => (
      <span className="inline-flex items-center rounded-full bg-sky-50 px-2 py-0.5 text-xs font-semibold text-sky-700">
        {info.getValue() ?? 'Unknown'}
      </span>
    ),
  }),
  columnHelper.accessor('minYearsExperience', {
    header: 'Min YOE',
    cell: (info) => (info.getValue() ?? '—'),
  }),
  columnHelper.accessor('languages', {
    header: 'Languages',
    cell: (info) => {
      const langs = info.getValue() ?? [];
      return <ClampedText text={langs.length ? langs.join(', ') : ''} />;
    },
  }),
  columnHelper.accessor('domain', {
    header: 'Domain',
    cell: (info) =>
      info.getValue() ? (
        <span className="inline-flex items-center rounded-full bg-orange-50 px-2 py-0.5 text-xs font-medium text-orange-700">
          {info.getValue()}
        </span>
      ) : (
        '—'
      ),
  }),
  columnHelper.accessor('technologies', {
    header: 'Technologies',
    cell: (info) => {
      const tech = info.getValue() ?? [];
      return <ClampedText text={tech.length ? tech.join(', ') : ''} />;
    },
  }),
  columnHelper.accessor('minDegree', {
    header: 'Degree',
    cell: (info) => info.getValue() || 'Unspecified',
  }),
  columnHelper.accessor('url', {
    header: 'Listing',
    cell: (info) => (
      <a href={info.getValue()} target="_blank" rel="noreferrer" className="text-sky-600 hover:underline">
        View
      </a>
    ),
  }),
  columnHelper.accessor('parsedDescription', {
    header: 'Parsed Description',
    cell: (info) => {
      return <ClampedText text={info.getValue() ?? ''} lines={3} className="text-slate-600" />;
    },
  }),
];

type JobsTableProps = {
  jobs: Job[];
};

export function JobsTable({ jobs }: JobsTableProps) {
  const table = useReactTable({
    data: jobs,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <div className="mt-6 overflow-hidden rounded-2xl border border-slate-100">
      <div className="overflow-x-auto">
        <table className="min-w-full border-separate border-spacing-0 text-sm text-slate-700">
          <thead>
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className="border-b border-slate-200 bg-slate-50 px-3 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-500"
                  >
                    {flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {table.getRowModel().rows.map((row) => (
              <tr key={row.id} className={row.index % 2 === 0 ? 'bg-white' : 'bg-slate-50'}>
                {row.getVisibleCells().map((cell) => (
                  <td key={cell.id} className="px-3 py-3 align-top text-slate-700">
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
