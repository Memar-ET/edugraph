import { useQuery } from '@tanstack/react-query'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, Spinner } from '@components/ui'
import { getSchoolQualityScores } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'
import { cn } from '@lib/utils/cn'
import { ScoreGauge } from '@components/charts'

export function SchoolQualityPage() {
  const user = useAuthStore((s) => s.user)
  const schoolId = user?.school_id ?? ''

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.schoolQuality(schoolId),
    queryFn: () => getSchoolQualityScores(schoolId),
    enabled: !!schoolId,
  })

  const scores = data?.scores ?? []
  const score = scores.length > 0
    ? scores.reduce((sum, s) => sum + s.compositeScore, 0) / scores.length
    : null
  const latestComputedAt = scores.length > 0
    ? scores.reduce((latest, s) => (s.computedAt > latest ? s.computedAt : latest), scores[0]!.computedAt)
    : null

  function scoreColor(s: number | null) {
    if (s === null) return 'text-gray-400'
    if (s >= 0.75) return 'text-health-700'
    if (s >= 0.5) return 'text-seal-700'
    return 'text-alert-700'
  }

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">School Quality Score</h1>
          <p className="mt-1 text-sm text-gray-500">
            Composite score updated nightly from CLO coverage, mastery, exam quality, and compliance.
          </p>
        </div>

        {isError && (
          <Banner variant="error" title="Failed to load quality score" description="Try refreshing." />
        )}

        {isLoading ? (
          <div className="flex justify-center py-16">
            <Spinner />
          </div>
        ) : (
          <>
            {/* Overall gauge */}
            <Card>
              <CardContent className="flex flex-col items-center py-8 gap-4">
                {score !== null ? (
                  <>
                    <ScoreGauge value={score} label="Composite Score" size={160} />
                    <p className={cn('text-4xl font-bold', scoreColor(score))}>
                      {Math.round(score * 100)}
                      <span className="text-lg font-normal text-gray-400">/100</span>
                    </p>
                    <p className="text-sm text-gray-500">
                      Last computed: {latestComputedAt ? new Date(latestComputedAt).toLocaleString() : '—'}
                    </p>
                  </>
                ) : (
                  <p className="text-gray-400">No score computed yet.</p>
                )}
              </CardContent>
            </Card>

            {/* Per subject+grade breakdown */}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {scores.map((s) => (
                <Card key={`${s.subjectCode}-${s.gradeLevel}`}>
                  <CardContent className="py-4 space-y-2">
                    <div className="flex items-center justify-between">
                      <p className="text-sm font-bold text-gray-900">
                        {s.subjectCode} · Grade {s.gradeLevel}
                      </p>
                      {s.flaggedForReview && (
                        <span className="rounded-full bg-alert-100 px-2 py-0.5 text-[10px] font-bold text-alert-700">
                          Flagged
                        </span>
                      )}
                    </div>
                    <p className={cn('text-2xl font-bold', scoreColor(s.compositeScore))}>
                      {Math.round(s.compositeScore * 100)}
                      <span className="text-sm font-normal text-gray-400">/100</span>
                    </p>
                    <div className="grid grid-cols-3 gap-2 text-center text-xs text-gray-500">
                      <div>
                        <p className="font-bold text-gray-900">{Math.round(s.cloCoveragePct * 100)}%</p>
                        <p>CLO coverage</p>
                      </div>
                      <div>
                        <p className="font-bold text-gray-900">{Math.round(s.studentMasteryPct * 100)}%</p>
                        <p>Mastery</p>
                      </div>
                      <div>
                        <p className="font-bold text-gray-900">{Math.round(s.examQualityAvg * 100)}%</p>
                        <p>Exam quality</p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </>
        )}
      </div>
    </AppShell>
  )
}
