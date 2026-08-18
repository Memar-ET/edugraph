import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'

import { AppShell } from '@components/layout'
import {
  Banner,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  EmptyState,
  Spinner,
} from '@components/ui'
import { getQMatrixQuality } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function QMatrixQualityPage() {
  const [subjectCode, setSubjectCode] = useState('')
  const [submitted, setSubmitted] = useState('')

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.qmatrixQuality(submitted),
    queryFn: () => getQMatrixQuality(submitted),
    enabled: !!submitted,
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (subjectCode.trim()) setSubmitted(subjectCode.trim().toUpperCase())
  }

  const issues = data
    ? [
        ...data.missingMappings.map((i) => ({ ...i, issueType: 'missing_mapping' as const })),
        ...data.lowConfidenceMappings.map((i) => ({ ...i, issueType: 'low_confidence' as const })),
        ...data.ambiguousMappings.map((i) => ({ ...i, issueType: 'ambiguous' as const })),
      ]
    : []
  const summary = data
    ? {
        totalItems: data.totalQuestions,
        unmapped: data.missingMappings.length,
        lowConfidence: data.lowConfidenceMappings.length,
        multiTopic: data.ambiguousMappings.length,
      }
    : undefined

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Q-Matrix Quality</h1>
          <p className="mt-1 text-sm text-gray-500">
            Review item–skill mapping coverage and confidence issues for a subject.
          </p>
        </div>

        {/* Subject picker */}
        <Card>
          <CardContent className="py-4">
            <form onSubmit={handleSubmit} className="flex items-center gap-3">
              <label htmlFor="subject-code" className="shrink-0 text-sm font-medium text-gray-700">
                Subject code
              </label>
              <input
                id="subject-code"
                value={subjectCode}
                onChange={(e) => setSubjectCode(e.target.value)}
                placeholder="e.g. BIO-G9"
                className="flex-1 rounded-lg border border-gray-200 px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
              <button
                type="submit"
                disabled={!subjectCode.trim()}
                className="rounded-lg bg-primary-700 px-4 py-2 text-sm font-medium text-white hover:bg-primary-600 focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:opacity-50"
              >
                Analyse
              </button>
            </form>
          </CardContent>
        </Card>

        {isError && <Banner variant="error" title="Failed to load Q-matrix quality" description="Check the subject code and try again." />}

        {isLoading && <div className="flex justify-center py-16"><Spinner /></div>}

        {submitted && !isLoading && !isError && (
          <>
            {/* Summary */}
            {summary && (
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                {[
                  { label: 'Total items', value: summary.totalItems },
                  { label: 'Unmapped', value: summary.unmapped },
                  { label: 'Low confidence', value: summary.lowConfidence },
                  { label: 'Multi-topic', value: summary.multiTopic },
                ].map((k) => (
                  <Card key={k.label}>
                    <CardContent className="py-4 text-center">
                      <p className="text-2xl font-bold text-primary-700">{k.value ?? '—'}</p>
                      <p className="mt-0.5 text-xs text-gray-500">{k.label}</p>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}

            {/* Issues table */}
            <Card>
              <CardHeader><CardTitle>Issues ({issues.length})</CardTitle></CardHeader>
              <CardContent className="p-0">
                {issues.length === 0 ? (
                  <EmptyState
                    title="No issues found"
                    description="All items have good Q-matrix coverage for this subject."
                  />
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-gray-100 bg-gray-50 text-left text-xs font-medium text-gray-500">
                          <th className="px-4 py-3">Question</th>
                          <th className="px-4 py-3">Issue</th>
                          <th className="px-4 py-3">Reason</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-50">
                        {issues.map((issue) => (
                          <tr key={`${issue.issueType}-${issue.questionId}`} className="hover:bg-gray-50">
                            <td className="px-4 py-3 text-gray-900 max-w-xs truncate">
                              {issue.questionText}
                            </td>
                            <td className="px-4 py-3">
                              <span className="rounded-full bg-seal-50 border border-seal-200 px-2 py-0.5 text-xs text-seal-700 capitalize">
                                {issue.issueType.replace(/_/g, ' ')}
                              </span>
                            </td>
                            <td className="px-4 py-3 text-gray-600">
                              {issue.reason}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </AppShell>
  )
}
