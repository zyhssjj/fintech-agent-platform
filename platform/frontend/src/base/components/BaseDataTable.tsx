import React from "react";
import { ArrowDown, ArrowUp, Search } from "lucide-react";

export interface BaseDataTableColumn<T> {
  key: keyof T | string;
  title: string;
  sortable?: boolean;
  filterable?: boolean;
  width?: string;
  render?: (row: T) => React.ReactNode;
}

export interface BaseDataTableProps<T> {
  rows: T[];
  columns: Array<BaseDataTableColumn<T>>;
  rowKey: (row: T) => string;
  loading?: boolean;
  page?: number;
  pageSize?: number;
  total?: number;
  onPageChange?: (page: number) => void;
  onSortChange?: (key: string, direction: "asc" | "desc") => void;
  onFilterChange?: (filters: Record<string, string>) => void;
}

export function BaseDataTable<T>({
  rows,
  columns,
  rowKey,
  loading = false,
  page = 1,
  pageSize = 10,
  total = rows.length,
  onPageChange,
  onSortChange,
  onFilterChange,
}: BaseDataTableProps<T>) {
  const [sortKey, setSortKey] = React.useState("");
  const [sortDirection, setSortDirection] = React.useState<"asc" | "desc">("asc");
  const [filters, setFilters] = React.useState<Record<string, string>>({});
  const pageCount = Math.max(1, Math.ceil(total / pageSize));

  function updateSort(key: string) {
    const nextDirection = sortKey === key && sortDirection === "asc" ? "desc" : "asc";
    setSortKey(key);
    setSortDirection(nextDirection);
    onSortChange?.(key, nextDirection);
  }

  function updateFilter(key: string, value: string) {
    const next = { ...filters, [key]: value };
    setFilters(next);
    onFilterChange?.(next);
  }

  return (
    <div className="overflow-hidden rounded-[8px] border border-slate-200 bg-white">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="bg-slate-50 text-slate-600">
            <tr>
              {columns.map((column) => {
                const key = String(column.key);
                return (
                  <th key={key} className="px-4 py-3 font-semibold" style={{ width: column.width }}>
                    <button
                      className="table-sort-button inline-flex items-center gap-1 disabled:cursor-default"
                      disabled={!column.sortable}
                      onClick={() => updateSort(key)}
                      type="button"
                    >
                      <span>{column.title}</span>
                      {column.sortable && sortKey === key && (
                        sortDirection === "asc" ? <ArrowUp className="h-3.5 w-3.5" /> : <ArrowDown className="h-3.5 w-3.5" />
                      )}
                    </button>
                    {column.filterable && (
                      <label className="mt-2 flex h-8 items-center gap-2 rounded-[6px] border border-slate-200 bg-white px-2">
                        <Search className="h-3.5 w-3.5 text-slate-400" />
                        <input
                          className="min-w-0 flex-1 bg-transparent text-xs outline-none"
                          onChange={(event) => updateFilter(key, event.target.value)}
                          placeholder="筛选"
                          value={filters[key] ?? ""}
                        />
                      </label>
                    )}
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td className="px-4 py-10 text-center text-slate-500" colSpan={columns.length}>加载中...</td>
              </tr>
            ) : rows.length ? (
              rows.map((row) => (
                <tr key={rowKey(row)} className="table-row border-t border-slate-100 hover:bg-slate-50">
                  {columns.map((column) => (
                    <td key={String(column.key)} className="px-4 py-3 text-slate-700">
                      {column.render ? column.render(row) : String((row as Record<string, unknown>)[String(column.key)] ?? "")}
                    </td>
                  ))}
                </tr>
              ))
            ) : (
              <tr>
                <td className="px-4 py-10 text-center text-slate-500" colSpan={columns.length}>暂无数据</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      <div className="flex items-center justify-between border-t border-slate-200 px-4 py-3 text-sm text-slate-600">
        <span>共 {total} 条，第 {page} / {pageCount} 页</span>
        <div className="flex items-center gap-2">
          <button className="rounded-[6px] border border-slate-200 px-3 py-1.5 disabled:opacity-50" disabled={page <= 1} onClick={() => onPageChange?.(page - 1)} type="button">上一页</button>
          <button className="rounded-[6px] border border-slate-200 px-3 py-1.5 disabled:opacity-50" disabled={page >= pageCount} onClick={() => onPageChange?.(page + 1)} type="button">下一页</button>
        </div>
      </div>
    </div>
  );
}
