import { useState } from 'react'
import { Search, SlidersHorizontal, ArrowUpDown, X, Filter } from 'lucide-react'
import { cn } from '@lib/utils/cn'
import { ThreeDCard } from '@components/ui'

export interface TableColumn<T> {
  key: string
  header: string
  render: (item: T) => React.ReactNode
  sortable?: boolean
}

export interface ManagementTableCardProps<T extends { id: string }> {
  title?: string
  searchPlaceholder?: string
  columns: TableColumn<T>[]
  data: T[]
  onRowClick?: (item: T) => void
  onAdvanceFilter?: () => void
  className?: string
}

export function ManagementTableCard<T extends { id: string }>({
  title = 'Performance & Records Management',
  searchPlaceholder = 'Search name, ID, or subject...',
  columns,
  data,
  onRowClick,
  className,
}: ManagementTableCardProps<T>) {
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [showFilterModal, setShowFilterModal] = useState(false)
  const [selectedRowDetail, setSelectedRowDetail] = useState<T | null>(null)
  const [statusFilter, setStatusFilter] = useState<string>('all')

  const filteredData = data.filter((item) => {
    const jsonStr = JSON.stringify(item).toLowerCase()
    const matchesSearch = jsonStr.includes(searchTerm.toLowerCase())
    if (statusFilter === 'all') return matchesSearch
    return matchesSearch && jsonStr.includes(statusFilter.toLowerCase())
  })

  const toggleSelectAll = () => {
    if (selectedIds.size === filteredData.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(filteredData.map((d) => d.id)))
    }
  }

  const toggleSelectRow = (id: string) => {
    const next = new Set(selectedIds)
    if (next.has(id)) {
      next.delete(id)
    } else {
      next.add(id)
    }
    setSelectedIds(next)
  }

  return (
    <>
      <ThreeDCard
        className={cn('flex flex-col border-gray-100/90 bg-white p-5', className)}
      >
        {/* Title & Actions Bar */}
        <div className="flex items-center justify-between pb-4">
          <h3 className="font-display text-lg font-bold tracking-tight text-gray-900">{title}</h3>
          <button
            type="button"
            onClick={() => setStatusFilter('all')}
            className="text-xs font-bold text-gray-500 hover:text-gray-900 transition-colors"
          >
            Reset Filters ({filteredData.length})
          </button>
        </div>

        {/* Filter and Search Bar Row */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-3 pb-4">
          <div className="relative w-full sm:w-80">
            <Search className="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder={searchPlaceholder}
              className="w-full rounded-xl border border-gray-200 bg-gray-50/80 pl-10 pr-4 py-2 text-xs font-semibold text-gray-900 placeholder-gray-400 outline-none transition-all focus:border-gray-900 focus:bg-white"
            />
          </div>

          <button
            type="button"
            onClick={() => setShowFilterModal(true)}
            className="inline-flex w-full sm:w-auto items-center justify-center gap-2 rounded-xl bg-gray-900 px-4 py-2 text-xs font-bold text-white shadow-sm transition-all hover:bg-gray-800 hover:scale-105 active:scale-95 cursor-pointer"
          >
            <SlidersHorizontal className="h-4 w-4" />
            <span>Advance Filter</span>
          </button>
        </div>

        {/* Responsive Table */}
        <div className="overflow-x-auto rounded-xl border border-gray-100/90">
          <table className="w-full text-left text-xs">
            <thead className="bg-gray-50/90 text-gray-500 font-bold border-b border-gray-100">
              <tr>
                <th className="p-3.5 w-10 text-center">
                  <input
                    type="checkbox"
                    checked={filteredData.length > 0 && selectedIds.size === filteredData.length}
                    onChange={toggleSelectAll}
                    className="rounded border-gray-300 text-gray-900 focus:ring-gray-900 cursor-pointer"
                  />
                </th>
                {columns.map((col) => (
                  <th key={col.key} className="p-3.5 font-bold text-gray-700">
                    <div className="flex items-center gap-1">
                      <span>{col.header}</span>
                      {col.sortable && <ArrowUpDown className="h-3 w-3 text-gray-400" />}
                    </div>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 text-gray-800">
              {filteredData.map((item) => {
                const isSelected = selectedIds.has(item.id)
                return (
                  <tr
                    key={item.id}
                    onClick={() => {
                      setSelectedRowDetail(item)
                      if (onRowClick) onRowClick(item)
                    }}
                    className={cn(
                      'transition-colors hover:bg-gray-50/90 cursor-pointer',
                      isSelected && 'bg-gray-50/90 font-medium',
                    )}
                  >
                    <td className="p-3.5 text-center" onClick={(e) => e.stopPropagation()}>
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => toggleSelectRow(item.id)}
                        className="rounded border-gray-300 text-gray-900 focus:ring-gray-900 cursor-pointer"
                      />
                    </td>
                    {columns.map((col) => (
                      <td key={col.key} className="p-3.5">
                        {col.render(item)}
                      </td>
                    ))}
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </ThreeDCard>

      {/* Interactive Filter Drawer Modal */}
      {showFilterModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-gray-900/40 backdrop-blur-sm p-4 animate-in fade-in">
          <div className="w-full max-w-md rounded-2xl border border-gray-100 bg-white p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between border-b border-gray-100 pb-3">
              <div className="flex items-center gap-2">
                <Filter className="h-5 w-5 text-gray-900" />
                <h3 className="font-display text-lg font-bold text-gray-900">Advance Filter Parameters</h3>
              </div>
              <button
                type="button"
                onClick={() => setShowFilterModal(false)}
                className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-900"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="space-y-3">
              <label className="block text-xs font-bold text-gray-700">Filter by Status</label>
              <div className="grid grid-cols-2 gap-2">
                {['all', 'mastered', 'in progress', 'needs review', 'approved', 'review'].map((st) => (
                  <button
                    key={st}
                    type="button"
                    onClick={() => {
                      setStatusFilter(st)
                      setShowFilterModal(false)
                    }}
                    className={cn(
                      'rounded-xl border px-3 py-2 text-xs font-bold capitalize transition-all',
                      statusFilter === st
                        ? 'bg-gray-900 text-white border-gray-900 shadow-sm'
                        : 'bg-gray-50 text-gray-700 border-gray-200 hover:bg-gray-100',
                    )}
                  >
                    {st}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => {
                  setStatusFilter('all')
                  setShowFilterModal(false)
                }}
                className="rounded-xl border border-gray-200 bg-white px-4 py-2 text-xs font-bold text-gray-700 hover:bg-gray-50"
              >
                Reset All
              </button>
              <button
                type="button"
                onClick={() => setShowFilterModal(false)}
                className="rounded-xl bg-gray-900 px-4 py-2 text-xs font-bold text-white hover:bg-gray-800"
              >
                Apply Filters
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Interactive Row Detail Inspection Modal */}
      {selectedRowDetail && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-gray-900/40 backdrop-blur-sm p-4 animate-in fade-in">
          <div className="w-full max-w-lg rounded-2xl border border-gray-100 bg-white p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between border-b border-gray-100 pb-3">
              <div>
                <span className="text-[10px] font-bold uppercase tracking-wider text-gray-400">Record Inspection</span>
                <h3 className="font-display text-lg font-bold text-gray-900">Record #{selectedRowDetail.id}</h3>
              </div>
              <button
                type="button"
                onClick={() => setSelectedRowDetail(null)}
                className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-900"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="rounded-xl bg-gray-50 p-4 border border-gray-100 space-y-2 text-xs">
              <pre className="whitespace-pre-wrap font-mono text-gray-800 text-[11px] overflow-x-auto max-h-60">
                {JSON.stringify(selectedRowDetail, null, 2)}
              </pre>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setSelectedRowDetail(null)}
                className="rounded-xl bg-gray-900 px-4 py-2 text-xs font-bold text-white hover:bg-gray-800"
              >
                Done Inspecting
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
