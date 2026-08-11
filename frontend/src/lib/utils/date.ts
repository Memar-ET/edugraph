import { format, formatDistanceToNow, isValid, parseISO } from 'date-fns'

function toDate(value: string | Date): Date {
  return typeof value === 'string' ? parseISO(value) : value
}

/** Formats an ISO timestamp as "Jul 20, 2026". Returns the raw input if unparseable. */
export function formatDate(value: string | Date | null | undefined, pattern = 'MMM d, yyyy'): string {
  if (!value) return '—'
  const date = toDate(value)
  if (!isValid(date)) return typeof value === 'string' ? value : '—'
  return format(date, pattern)
}

/** Formats an ISO timestamp as "Jul 20, 2026, 3:45 PM". */
export function formatDateTime(value: string | Date | null | undefined): string {
  return formatDate(value, 'MMM d, yyyy, h:mm a')
}

/** Formats an ISO timestamp as a relative time, e.g. "3 days ago". */
export function formatRelative(value: string | Date | null | undefined): string {
  if (!value) return '—'
  const date = toDate(value)
  if (!isValid(date)) return '—'
  return formatDistanceToNow(date, { addSuffix: true })
}
