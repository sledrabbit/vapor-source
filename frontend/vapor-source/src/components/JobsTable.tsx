import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type ColumnFiltersState,
  type FilterFn,
  type PaginationState,
  type RowData,
  type SortingState,
} from '@tanstack/react-table';
import { useEffect, useMemo, useState } from 'react';
import type { Job } from '../types/job';

declare module '@tanstack/react-table' {
  interface ColumnMeta<TData extends RowData, TValue> {
    filterType?: 'text' | 'multi';
    options?: string[];
    title?: string;
  }
}

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

const defaultTextFilter: FilterFn<Job> = (row, columnId, filterValue) => {
  const filterText = String(filterValue ?? '').trim().toLowerCase();
  if (!filterText) return true;
  const value = row.getValue(columnId);
  if (value == null) return false;

  const matchesText = (input: unknown) => {
    if (input == null) return false;
    const text = String(input).toLowerCase();
    if (!text) return false;
    if (text.startsWith(filterText)) return true;
    const tokens = text.split(/[\s,;/|]+/);
    return tokens.some((token) => token.startsWith(filterText));
  };

  if (Array.isArray(value)) {
    return value.some((item) => {
      if (item == null) return false;
      const text = String(item).toLowerCase();
      return text === filterText;
    });
  }

  return matchesText(value);
};

const numericMaxFilter: FilterFn<Job> = (row, columnId, filterValue) => {
  const filterText = String(filterValue ?? '').trim();
  if (!filterText) return true;
  const maxNumber = Number(filterText);
  if (Number.isNaN(maxNumber)) return true;
  const value = row.getValue(columnId);
  if (value == null) return true;
  const numericValue = Number(value);
  if (Number.isNaN(numericValue)) return true;
  return numericValue <= maxNumber;
};

const multiSelectFilter: FilterFn<Job> = (row, columnId, filterValue) => {
  const selections = Array.isArray(filterValue)
    ? (filterValue as string[]).map((entry) => String(entry))
    : [];
  if (selections.length === 0) return true;
  const value = row.getValue(columnId);
  if (value == null) return false;
  if (Array.isArray(value)) {
    return value.some((item) => item && selections.includes(String(item)));
  }
  return selections.includes(String(value));
};

function formatFilterValue(value: unknown) {
  if (Array.isArray(value)) {
    return value.join(', ');
  }
  if (value && typeof value === 'object') {
    return Object.values(value as Record<string, string>).filter(Boolean).join(' – ');
  }
  return String(value ?? '');
}

const textFilterControls = [
  { id: 'location', label: 'Location', placeholder: 'Filter location' },
  { id: 'minYearsExperience', label: 'Min YOE', placeholder: 'Up to…', type: 'number' as const },
];

const modalFilterButtons = [
  { id: 'modality', label: 'Modality' },
  { id: 'languages', label: 'Languages' },
  { id: 'domain', label: 'Domain' },
  { id: 'technologies', label: 'Technologies' },
  { id: 'minDegree', label: 'Degree' },
];

type FilterModalProps = {
  title: string;
  options: string[];
  selected: string[];
  onClose: () => void;
  onApply: (values: string[]) => void;
};

function FilterModal({ title, options, selected, onApply, onClose }: FilterModalProps) {
  const [localSelections, setLocalSelections] = useState<string[]>(selected);

  useEffect(() => {
    setLocalSelections(selected);
  }, [selected]);

  const toggleSelection = (option: string) => {
    setLocalSelections((prev) => (prev.includes(option) ? prev.filter((value) => value !== option) : [...prev, option]));
  };

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 px-4">
      <div className="w-full max-w-md rounded-2xl bg-white p-5 shadow-2xl">
        <div className="flex items-center justify-between">
          <h3 className="text-base font-semibold text-slate-900">{title}</h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-full p-1 text-slate-500 transition hover:bg-slate-100 hover:text-slate-700"
            aria-label="Close"
          >
            ×
          </button>
        </div>
        <div className="mt-4 max-h-64 overflow-y-auto pr-1 text-sm text-slate-700">
          {options.length === 0 && <p className="text-slate-500">No available values.</p>}
          {options.map((option) => (
            <label key={option} className="mb-2 flex items-center gap-2">
              <input
                type="checkbox"
                checked={localSelections.includes(option)}
                onChange={() => toggleSelection(option)}
                className="h-4 w-4 rounded border-slate-300 text-sky-600 focus:ring-sky-500"
              />
              <span>{option}</span>
            </label>
          ))}
        </div>
        <div className="mt-4 flex justify-between gap-2">
          <button
            type="button"
            onClick={() => setLocalSelections([])}
            className="rounded-md border border-slate-200 px-3 py-2 text-sm font-medium text-slate-600 transition hover:border-slate-300 hover:text-slate-800"
          >
            Clear
          </button>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md border border-slate-200 px-3 py-2 text-sm font-medium text-slate-600 transition hover:border-slate-300 hover:text-slate-800"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => {
                onApply(localSelections);
                onClose();
              }}
              className="rounded-md bg-sky-600 px-3 py-2 text-sm font-semibold text-white transition hover:bg-sky-500"
            >
              Apply
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

type JobsTableProps = {
  jobs: Job[];
  pageSize?: number;
};

export function JobsTable({ jobs, pageSize = 10 }: JobsTableProps) {
  const [sorting, setSorting] = useState<SortingState>([{ id: 'postedDate', desc: true }]);
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize });
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [activeFilterColumnId, setActiveFilterColumnId] = useState<string>();

  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageSize }));
  }, [pageSize]);

  const filterOptions = useMemo(() => {
    const modality = new Set<string>();
    const domain = new Set<string>();
    const technologies = new Set<string>();
    const languages = new Set<string>();
    const degrees = new Set<string>();

    for (const job of jobs) {
      if (job.modality) modality.add(job.modality);
      if (job.domain) domain.add(job.domain);
      if (job.minDegree) degrees.add(job.minDegree);
      (job.technologies ?? []).forEach((tech) => tech && technologies.add(tech));
      (job.languages ?? []).forEach((lang) => lang && languages.add(lang));
    }

    const toSortedArray = (set: Set<string>) => Array.from(set).sort((a, b) => a.localeCompare(b));

    return {
      modality: toSortedArray(modality),
      domain: toSortedArray(domain),
      technologies: toSortedArray(technologies),
      languages: toSortedArray(languages),
      degrees: toSortedArray(degrees),
    };
  }, [jobs]);

  const columns = useMemo<ColumnDef<Job, any>[]>(() => {
    return [
      columnHelper.accessor('postedDate', {
        header: 'Posted',
        enableColumnFilter: false,
      }),
      columnHelper.accessor('title', {
        header: 'Role',
        enableColumnFilter: false,
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
      columnHelper.accessor('location', {
        header: 'Location',
        filterFn: defaultTextFilter,
        cell: (info) => info.getValue() || '—',
      }),
      columnHelper.accessor('modality', {
        header: 'Modality',
        filterFn: multiSelectFilter,
        meta: { filterType: 'multi', options: filterOptions.modality, title: 'Modality' },
        cell: (info) => (
          <span className="inline-flex items-center rounded-full bg-sky-50 px-2 py-0.5 text-xs font-semibold text-sky-700">
            {info.getValue() ?? 'Unknown'}
          </span>
        ),
      }),
      columnHelper.accessor('minYearsExperience', {
        header: 'YOE',
        filterFn: numericMaxFilter,
        cell: (info) => info.getValue() ?? '—',
      }),
      columnHelper.accessor('languages', {
        header: 'Languages',
        filterFn: multiSelectFilter,
        meta: { filterType: 'multi', options: filterOptions.languages, title: 'Languages' },
        cell: (info) => {
          const langs = info.getValue() ?? [];
          return <ClampedText text={langs.length ? langs.join(', ') : ''} />;
        },
      }),
      columnHelper.accessor('domain', {
        header: 'Domain',
        filterFn: multiSelectFilter,
        meta: { filterType: 'multi', options: filterOptions.domain, title: 'Domain' },
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
        filterFn: multiSelectFilter,
        meta: { filterType: 'multi', options: filterOptions.technologies, title: 'Technologies' },
        cell: (info) => {
          const tech = info.getValue() ?? [];
          return <ClampedText text={tech.length ? tech.join(', ') : ''} />;
        },
      }),
      columnHelper.accessor('minDegree', {
        header: 'Degree',
        filterFn: multiSelectFilter,
        meta: { filterType: 'multi', options: filterOptions.degrees, title: 'Degree' },
        cell: (info) => info.getValue() || 'Unspecified',
      }),
  columnHelper.accessor('url', {
    header: 'Listing',
    enableColumnFilter: false,
    enableSorting: false,
    cell: (info) => (
      <a href={info.getValue()} target="_blank" rel="noreferrer" className="text-sky-600 hover:underline">
        View
      </a>
    ),
  }),
  columnHelper.accessor('parsedDescription', {
    header: 'Parsed Description',
    enableColumnFilter: false,
    enableSorting: false,
    cell: (info) => {
      return <ClampedText text={info.getValue() ?? ''} lines={3} className="text-slate-600" />;
    },
  }),
    ];
  }, [filterOptions]);

  const table = useReactTable({
    data: jobs,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    onSortingChange: setSorting,
    onPaginationChange: setPagination,
    onColumnFiltersChange: setColumnFilters,
    state: { sorting, pagination, columnFilters },
    autoResetPageIndex: false,
  });

  const activeColumn = activeFilterColumnId ? table.getColumn(activeFilterColumnId) : undefined;
  const activeSelections = Array.isArray(activeColumn?.getFilterValue())
    ? (activeColumn?.getFilterValue() as string[])
    : [];
  const activeOptions = (activeColumn?.columnDef.meta?.options ?? []) as string[];
  const activeTitle = activeColumn?.columnDef.meta?.title ?? 'Filter';
  const hasActiveFilters = columnFilters.length > 0;
  const filterChips = columnFilters
    .map((filter) => {
      const column = table.getColumn(filter.id);
      if (!column) return undefined;
      const valueLabel = formatFilterValue(filter.value);
      if (!valueLabel) return undefined;
      const title =
        column.columnDef.meta?.title ??
        (typeof column.columnDef.header === 'string' ? column.columnDef.header : column.id);
      return {
        id: filter.id,
        label: `${title}: ${valueLabel}`,
        clear: () => column.setFilterValue(undefined),
      };
    })
    .filter((chip): chip is { id: string; label: string; clear: () => void } => Boolean(chip));

  return (
    <div className="mt-6">
      <div className="flex flex-wrap items-end justify-between gap-3 px-2 pb-3">
        <div className="flex flex-1 flex-wrap gap-3">
          {textFilterControls.map((control) => {
            const column = table.getColumn(control.id);
            if (!column) return null;
            const value = typeof column.getFilterValue() === 'string' ? (column.getFilterValue() as string) : '';
            return (
              <label
                key={control.id}
                className="flex flex-col items-center gap-1 text-center text-[11px] font-semibold uppercase text-slate-500"
              >
                <span>{control.label}</span>
                <input
                  type={control.type ?? 'text'}
                  value={value}
                  onChange={(event) => column.setFilterValue(event.target.value || undefined)}
                  placeholder={control.placeholder}
                  className="w-28 rounded-md border border-slate-200 px-2 py-1 text-xs font-medium text-slate-600 placeholder:text-slate-400 focus:border-sky-300 focus:outline-none focus:ring-1 focus:ring-sky-200"
                />
              </label>
            );
          })}
          {modalFilterButtons.map((filter) => {
            const column = table.getColumn(filter.id);
            if (!column) return null;
            const selections = Array.isArray(column.getFilterValue()) ? (column.getFilterValue() as string[]) : [];
            return (
              <label
                key={filter.id}
                className="flex flex-col items-center gap-1 text-center text-[11px] font-semibold uppercase text-slate-500"
              >
                <span>{filter.label}</span>
                <button
                  type="button"
                  onClick={() => setActiveFilterColumnId(filter.id)}
                  className="flex w-28 items-center justify-between rounded-md border border-slate-200 bg-white px-2 py-1 text-xs font-semibold text-slate-600 transition hover:border-slate-300 hover:text-slate-800"
                >
                  <span>Select…</span>
                  {selections.length > 0 && (
                    <span className="text-[11px] text-slate-400">({selections.length})</span>
                  )}
                </button>
              </label>
            );
          })}
        </div>
        <button
          type="button"
          onClick={() => {
            table.resetColumnFilters();
            setActiveFilterColumnId(undefined);
          }}
          className="rounded-md border border-slate-200 bg-white px-3 py-1 text-xs font-semibold text-slate-600 transition hover:border-slate-300 hover:text-slate-800 disabled:opacity-50"
          disabled={!hasActiveFilters}
        >
          Reset filters
        </button>
      </div>
      {filterChips.length > 0 && (
        <div className="flex flex-wrap gap-2 px-2 pb-3">
          {filterChips.map((chip) => (
            <button
              key={chip.id}
              type="button"
              onClick={chip.clear}
              className="inline-flex items-center gap-1 rounded-full border border-sky-200 bg-sky-50 px-3 py-1 text-xs font-semibold text-sky-700 shadow-sm transition hover:border-sky-300 hover:text-sky-900"
            >
              <span>{chip.label}</span>
              <span className="text-sky-500">×</span>
            </button>
          ))}
        </div>
      )}
      <div className="overflow-hidden rounded-2xl border border-slate-100">
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
              {table.getRowModel().rows.map((row, rowIdx) => (
                <tr key={row.id} className={rowIdx % 2 === 0 ? 'bg-white' : 'bg-slate-50'}>
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
      {activeColumn && (
        <FilterModal
          title={activeTitle}
          options={activeOptions}
          selected={activeSelections}
          onClose={() => setActiveFilterColumnId(undefined)}
          onApply={(values) => activeColumn.setFilterValue(values.length ? values : undefined)}
        />
      )}
    </div>
  );
}
