import { useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'

import { AppShell } from '@components/layout'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Spinner, StatusPill } from '@components/ui'
import { ScoreGauge } from '@components/charts'
import { apiErrorMessage } from '@lib/api/client'
import { getExamIntegritySummary, getExamQuality } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { formatDateTime } from '@lib/utils/date'
import { formatPercent, formatSignedScore } from '@lib/utils/format'

const CALIBRATION_TONE = { accurate: 'health', mismatch: 'alert', unrated: 'neutral' } as const

const INTEGRITY_EVENT_LABELS: Record<string, string> = {
  tab_hidden: 'Tab switched away',
  tab_visible: 'Tab returned to',
  fullscreen_entered: 'Entered fullscreen',
  fullscreen_exited: 'Exited fullscreen',
  connection_lost: 'Connection lost',
  connection_restored: 'Connection restored',
}

export function ExamQualityPage() {
  const { examId } = useParams({ from: '/teacher/exams/$examId/quality' })
  const navigate = useNavigate()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.examQuality(examId),
    queryFn: () => getExamQuality(examId),
  })

  const integrityQuery = useQuery({
    queryKey: queryKeys.examIntegrity(examId),
    queryFn: () => getExamIntegritySummary(examId),
  })
  const integrityEntries = Object.entries(integrityQuery.data ?? {}).filter(([, count]) => count > 0)

  return (
    <AppShell title="Exam quality report" description="Capability 4B — retroactively grading the exam itself.">
      <div className="space-y-6">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => void navigate({ to: '/teacher/exams/$examId', params: { examId } })}
        >
          ← Back to exam
        </Button>

        {isLoading && (
          <div className="flex items-center gap-2 text-gray-500">
            <Spinner /> Computing quality report…
          </div>
        )}
        {isError && <Banner tone="error">{apiErrorMessage(error, 'Could not load the quality report.')}</Banner>}

        {data && (
          <>
            <Card>
              <CardContent className="flex flex-wrap items-center gap-6 pt-6">
                <ScoreGauge
                  value={data.qualityScore}
                  label="Quality score"
                  tone={data.qualityScore >= 70 ? 'health' : data.qualityScore >= 40 ? 'seal' : 'alert'}
                  size={104}
                />
                <div>
                  <p className="font-display text-lg font-semibold text-gray-900">{data.title}</p>
                  <p className="text-sm text-gray-500">
                    {data.attemptsAnalyzed} attempt{data.attemptsAnalyzed === 1 ? '' : 's'} analyzed · avg
                    discrimination {formatSignedScore(data.discriminationAvg)} · computed{' '}
                    {formatDateTime(data.computedAt)}
                  </p>
                </div>
              </CardContent>
            </Card>

            {/* Integrity signals: plain counts across every attempt on this
                exam, never a per-student breakdown or an accusation -- a
                tab switch can happen for entirely innocent reasons (a
                notification, an accidental click), so this is framed as
                "N visibility changes," not "N students cheated." */}
            {!integrityQuery.isLoading && integrityEntries.length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">Integrity signals</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-wrap gap-3 pt-0">
                  {integrityEntries.map(([type, count]) => (
                    <StatusPill key={type} tone="neutral">
                      {count} {INTEGRITY_EVENT_LABELS[type] ?? type}
                    </StatusPill>
                  ))}
                </CardContent>
              </Card>
            )}

            <div className="grid gap-6 md:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Mandatory CLO coverage</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  <p className="text-gray-600">
                    {data.cloCoverage.mandatoryTested} of {data.cloCoverage.mandatoryTotal} mandatory CLOs
                    tested.
                  </p>
                  {data.cloCoverage.zeroCorrect.length > 0 && (
                    <div>
                      <p className="mb-1 text-xs font-medium uppercase tracking-wide text-gray-400">
                        Zero students correct — reteach now
                      </p>
                      <div className="flex flex-wrap gap-1">
                        {data.cloCoverage.zeroCorrect.map((f) => (
                          <StatusPill key={f.cloCode} tone="alert">
                            {f.cloCode}
                          </StatusPill>
                        ))}
                      </div>
                    </div>
                  )}
                  {data.cloCoverage.untested.length > 0 && (
                    <div>
                      <p className="mb-1 text-xs font-medium uppercase tracking-wide text-gray-400">
                        Never tested
                      </p>
                      <div className="flex flex-wrap gap-1">
                        {data.cloCoverage.untested.map((f) => (
                          <StatusPill key={f.cloCode} tone="seal">
                            {f.cloCode}
                          </StatusPill>
                        ))}
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Time analysis</CardTitle>
                </CardHeader>
                <CardContent className="text-sm text-gray-600">
                  {data.timeAnalysis.evaluated ? (
                    <p>
                      {data.timeAnalysis.questionsWithTiming} question(s) had timing data. Median avg time:{' '}
                      {data.timeAnalysis.medianAvgSecs ? `${Math.round(data.timeAnalysis.medianAvgSecs)}s` : '—'}.
                    </p>
                  ) : (
                    <p>Not enough timing data submitted yet to evaluate.</p>
                  )}
                </CardContent>
              </Card>
            </div>

            <Card>
              <CardHeader>
                <CardTitle className="text-base">Per-question breakdown</CardTitle>
              </CardHeader>
              <CardContent className="overflow-x-auto p-0">
                <table className="w-full min-w-[720px] text-sm">
                  <thead>
                    <tr className="border-b border-gray-200 text-left text-xs uppercase tracking-wide text-gray-400">
                      <th className="px-6 py-2 font-medium">#</th>
                      <th className="px-3 py-2 font-medium">Type</th>
                      <th className="px-3 py-2 font-medium">CLO</th>
                      <th className="px-3 py-2 font-medium">Discrimination</th>
                      <th className="px-3 py-2 font-medium">Calibration</th>
                      <th className="px-3 py-2 font-medium">Avg score</th>
                      <th className="px-6 py-2 font-medium">Time</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100 font-mono text-xs">
                    {data.questions.map((q) => (
                      <tr key={q.questionId}>
                        <td className="px-6 py-2 text-gray-500">{q.sequenceNumber}</td>
                        <td className="px-3 py-2 text-gray-700">{q.questionType}</td>
                        <td className="px-3 py-2 text-gray-700">{q.cloCode ?? '—'}</td>
                        <td className="px-3 py-2 text-gray-700">
                          {formatSignedScore(q.discriminationIndex)}
                          {q.discriminationMethod && (
                            <span className="ml-1 text-gray-400">({q.discriminationMethod})</span>
                          )}
                        </td>
                        <td className="px-3 py-2">
                          <StatusPill tone={CALIBRATION_TONE[q.calibration as keyof typeof CALIBRATION_TONE] ?? 'neutral'}>
                            {q.calibration}
                          </StatusPill>
                        </td>
                        <td className="px-3 py-2 text-gray-700">{formatPercent(q.avgScoreRatio, { fromRatio: true })}</td>
                        <td className="px-6 py-2">
                          {q.timeAnomaly ? (
                            <StatusPill tone="alert">anomaly</StatusPill>
                          ) : (
                            <span className="text-gray-400">—</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </AppShell>
  )
}
