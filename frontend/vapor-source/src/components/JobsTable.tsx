import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type PaginationState,
  type SortingState,
} from '@tanstack/react-table';
import { useEffect, useState } from 'react';
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
  pageSize?: number;
};

export function JobsTable({ jobs, pageSize = 10 }: JobsTableProps) {
  const [sorting, setSorting] = useState<SortingState>([{ id: 'postedDate', desc: true }]);
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize });

  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageSize }));
  }, [pageSize]);

  const table = useReactTable({
    data: jobs,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    onSortingChange: setSorting,
    onPaginationChange: setPagination,
    state: { sorting, pagination },
    autoResetPageIndex: false,
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
                    {header.isPlaceholder ? null : (
                      <button
                        type="button"
                        className="flex items-center gap-1 text-left"
                        onClick={header.column.getToggleSortingHandler()}
                      >
                        {flexRender(header.column.columnDef.header, header.getContext())}
                        {{
                          asc: '↑',
                          desc: '↓',
                        }[header.column.getIsSorted() as 'asc' | 'desc'] ?? null}
                      </button>
                    )}
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
      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 px-4 py-3 text-sm text-slate-600">
        <span>
          Page {table.getState().pagination.pageIndex + 1} of {Math.max(table.getPageCount(), 1)}
        </span>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
            className="rounded-md border border-slate-200 px-3 py-1 text-sm font-medium text-slate-600 transition enabled:hover:border-slate-300 enabled:hover:text-slate-800 disabled:opacity-50"
          >
            Previous
          </button>
          <button
            type="button"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
            className="rounded-md border border-slate-200 px-3 py-1 text-sm font-medium text-slate-600 transition enabled:hover:border-slate-300 enabled:hover:text-slate-800 disabled:opacity-50"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  );
}
