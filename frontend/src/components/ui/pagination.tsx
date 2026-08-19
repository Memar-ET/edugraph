import { ChevronLeft, ChevronRight } from 'lucide-react'

import { cn } from '@lib/utils/cn'

interface PaginationProps {
  page: number
  totalPages: number
  onPageChange: (page: number) => void
  className?: string
}

export function Pagination({ page, totalPages, onPageChange, className }: PaginationProps) {
  if (totalPages <= 1) return null

  const pages: (number | '…')[] = []
  if (totalPages <= 7) {
    for (let i = 1; i <= totalPages; i++) pages.push(i)
  } else {
    pages.push(1)
    if (page > 3) pages.push('…')
    for (let i = Math.max(2, page - 1); i <= Math.min(totalPages - 1, page + 1); i++) pages.push(i)
    if (page < totalPages - 2) pages.push('…')
    pages.push(totalPages)
  }

  return (
    <nav
      aria-label="Pagination"
      className={cn('flex items-center gap-1', className)}
    >
      <button
        onClick={() => onPageChange(page - 1)}
        disabled={page === 1}
        aria-label="Previous page"
        className="rounded p-1.5 text-gray-500 hover:bg-gray-100 disabled:opacity-40 focus:outline-none focus:ring-2 focus:ring-primary-500"
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
      </button>

      {pages.map((p, i) =>
        p === '…' ? (
          <span key={`ellipsis-${i}`} className="px-2 text-sm text-gray-400">
            …
          </span>
        ) : (
          <button
            key={p}
            onClick={() => onPageChange(p as number)}
            aria-current={p === page ? 'page' : undefined}
            className={cn(
              'min-w-[2rem] rounded px-2 py-1 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-primary-500',
              p === page
                ? 'bg-primary-700 text-white'
                : 'text-gray-600 hover:bg-gray-100',
            )}
          >
            {p}
          </button>
        ),
      )}

      <button
        onClick={() => onPageChange(page + 1)}
        disabled={page === totalPages}
        aria-label="Next page"
        className="rounded p-1.5 text-gray-500 hover:bg-gray-100 disabled:opacity-40 focus:outline-none focus:ring-2 focus:ring-primary-500"
      >
        <ChevronRight className="h-4 w-4" aria-hidden />
      </button>
    </nav>
  )
}
