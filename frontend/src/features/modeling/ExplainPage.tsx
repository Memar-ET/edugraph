import { useQuery } from '@tanstack/react-query'
import { useParams } from '@tanstack/react-router'
import { Brain, TrendingDown, TrendingUp } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, CardHeader, CardTitle, Spinner, StatusPill } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { getExplanation, getSkillStateSnapshots } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

function pct(n: number | undefined): string {
  return n === undefined ? '—' : `${Math.round(n * 100)}%`
}

function trendIcon(trend: string | undefined) {
  if (trend === 'improving') return <TrendingUp className="h-4 w-4 text-health-600" aria-hidden />
  if (trend === 'declining') return <TrendingDown className="h-4 w-4 text-alert-600" aria-hidden />
  return null
}

// EG-GCKT Milestone 11 (spec section 18): the five-part explanation --
// current state, evidence, structural context, confidence, recommendation
// + reason -- for one student's estimated knowledge of one topic.
export function ExplainPage() {
  const { studentId, topicId } = useParams({ strict: false }) as { studentId: string; topicId: string }

  const explainQuery = useQuery({
    queryKey: queryKeys.explanation(studentId, topicId),
    queryFn: () => getExplanation(studentId, topicId),
  })

  const snapshotsQuery = useQuery({
    queryKey: queryKeys.skillStateSnapshots(studentId, topicId),
    queryFn: () => getSkillStateSnapshots(studentId, topicId),
  })

  return (
    <AppShell title="Knowledge State Explanation" description="Why the system believes what it believes about this topic.">
      {explainQuery.isLoading && (
        <div className="flex items-center gap-2 text-sm text-gray-500">
          <Spinner /> Loading explanation…
        </div>
      )}
      {explainQuery.isError && (
        <Banner tone="error">{apiErrorMessage(explainQuery.error, 'Could not load this explanation.')}</Banner>
      )}
      {explainQuery.data && (
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>{explainQuery.data.topicTitle}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap items-center gap-3">
                <StatusPill tone="neutral">Mastery: {pct(explainQuery.data.currentState.masteryProbability)}</StatusPill>
                <StatusPill tone="seal">{explainQuery.data.currentState.masteryStatus}</StatusPill>
                {explainQuery.data.currentState.trend && (
                  <span className="inline-flex items-center gap-1 text-sm text-gray-600">
                    {trendIcon(explainQuery.data.currentState.trend)}
                    {explainQuery.data.currentState.trend}
                  </span>
                )}
                <StatusPill tone="neutral">
                  {explainQuery.data.currentState.evidenceCount} observation(s)
                </StatusPill>
                <StatusPill tone={explainQuery.data.confidence === 'high' ? 'health' : explainQuery.data.confidence === 'low' ? 'alert' : 'seal'}>
                  {explainQuery.data.confidence} confidence
                </StatusPill>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <Brain className="h-4 w-4" aria-hidden />
                Recommendation
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="font-medium text-gray-900">{explainQuery.data.recommendation}</p>
              <p className="mt-1 text-sm text-gray-500">{explainQuery.data.reason}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Evidence</CardTitle>
            </CardHeader>
            <CardContent>
              {explainQuery.data.evidence.length === 0 ? (
                <p className="text-sm text-gray-500">No evidence recorded yet.</p>
              ) : (
                <ul className="space-y-2">
                  {explainQuery.data.evidence.map((e, i) => (
                    <li key={i} className="flex flex-wrap items-center gap-3 text-sm">
                      <StatusPill tone="neutral">{e.provenance}</StatusPill>
                      <span className="text-gray-500">estimate {pct(e.estimate)}</span>
                      {e.reliability !== undefined && <span className="text-gray-500">reliability {e.reliability.toFixed(2)}</span>}
                      <span className="text-gray-400">{new Date(e.createdAt).toLocaleDateString()}</span>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Structural context</CardTitle>
            </CardHeader>
            <CardContent>
              {explainQuery.data.structuralContext.prerequisites.length === 0 ? (
                <p className="text-sm text-gray-500">No prerequisites recorded for this topic.</p>
              ) : (
                <ul className="space-y-2">
                  {explainQuery.data.structuralContext.prerequisites.map((p) => (
                    <li key={`${p.topicId}-${p.edgeType}`} className="flex flex-wrap items-center gap-3 text-sm">
                      <span className="font-medium text-gray-900">{p.title}</span>
                      <StatusPill tone="neutral">{p.edgeType}</StatusPill>
                      <span className="text-gray-500">mastery {pct(p.masteryProbability)}</span>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">History</CardTitle>
            </CardHeader>
            <CardContent>
              {snapshotsQuery.isLoading && (
                <div className="flex items-center gap-2 text-sm text-gray-500">
                  <Spinner /> Loading history…
                </div>
              )}
              {snapshotsQuery.isError && (
                <p className="text-sm text-gray-500">Could not load state history.</p>
              )}
              {snapshotsQuery.data && snapshotsQuery.data.length === 0 && (
                <p className="text-sm text-gray-500">No historical snapshots recorded yet.</p>
              )}
              {snapshotsQuery.data && snapshotsQuery.data.length > 0 && (
                <ul className="space-y-2">
                  {snapshotsQuery.data.map((s) => (
                    <li key={s.id} className="flex flex-wrap items-center gap-3 text-sm">
                      <span className="text-gray-400">{new Date(s.takenAt).toLocaleDateString()}</span>
                      <StatusPill tone="neutral">{pct(s.masteryProbability)}</StatusPill>
                      <StatusPill tone="seal">{s.masteryStatus}</StatusPill>
                      {s.trend && (
                        <span className="inline-flex items-center gap-1 text-gray-600">
                          {trendIcon(s.trend)}
                          {s.trend}
                        </span>
                      )}
                      <span className="text-gray-500">{s.evidenceCount} observation(s)</span>
                      <span className="text-gray-400">{s.snapshotReason}</span>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </AppShell>
  )
}
