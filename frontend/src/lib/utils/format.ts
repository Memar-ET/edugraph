/** Formats a 0-100 (or 0-1) number as a percentage string, e.g. "72%". */
export function formatPercent(value: number | null | undefined, opts: { fromRatio?: boolean } = {}): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '—'
  const pct = opts.fromRatio ? value * 100 : value
  return `${Math.round(pct)}%`
}

/** Formats a plain count with thousands separators. */
export function formatNumber(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '—'
  return new Intl.NumberFormat('en-US').format(value)
}

/** Formats a 0..1 discrimination index (-1..1) with a leading sign, e.g. "+0.42". */
export function formatSignedScore(value: number | null | undefined, digits = 2): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '—'
  const sign = value > 0 ? '+' : ''
  return `${sign}${value.toFixed(digits)}`
}

/** Title-cases a snake_case or lower status string, e.g. "validation_pending" -> "Validation pending". */
export function formatStatusLabel(status: string): string {
  const spaced = status.replace(/_/g, ' ')
  return spaced.charAt(0).toUpperCase() + spaced.slice(1)
}

/** Truncates long free text (e.g. LLM explanations) for compact card display. */
export function truncate(text: string, maxLength = 140): string {
  if (text.length <= maxLength) return text
  return `${text.slice(0, maxLength - 1).trimEnd()}…`
}

/** Best-effort label extraction from a JSONB field whose shape is
 * generator-defined (e.g. SubjectProfile.topWeakAreas, tutor relatedTopics)
 * -- pulls strings directly, or the first matching key off each object. */
export function extractLabels(raw: unknown, keys: string[] = ['title', 'topic', 'name', 'label']): string[] {
  if (!raw) return []
  const arr = Array.isArray(raw)
    ? raw
    : typeof raw === 'object'
      ? Object.values(raw as Record<string, unknown>)
      : []
  const labels: string[] = []
  for (const item of arr) {
    if (typeof item === 'string') {
      labels.push(item)
    } else if (item && typeof item === 'object') {
      const obj = item as Record<string, unknown>
      const found = keys.map((k) => obj[k]).find((v) => typeof v === 'string')
      if (found) labels.push(found as string)
    }
  }
  return labels
}
