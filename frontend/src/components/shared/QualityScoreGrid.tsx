import { ScoreGauge } from '@components/charts'
import { Card, CardContent, Seal, StatusPill } from '@components/ui'
import { formatDate } from '@lib/utils/date'
import { formatPercent } from '@lib/utils/format'
import type { SchoolQualityScore } from '@/types/api'

export interface QualityScoreGridProps {
  scores: SchoolQualityScore[]
}

/** Capability 4C composite quality cards -- shared by the school admin,
 * regional, and ministry dashboards so a score reads identically no
 * matter which level of oversight is looking at it. */
export function QualityScoreGrid({ scores }: QualityScoreGridProps) {
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {scores.map((score) => (
        <Card key={`${score.subjectCode}-${score.gradeLevel}`}>
          <CardContent className="flex items-start gap-4 pt-6">
            <ScoreGauge value={score.compositeScore} tone={score.flaggedForReview ? 'alert' : 'health'} size={72} />
            <div className="min-w-0 flex-1">
              <p className="font-display text-base font-semibold text-gray-900">
                {score.subjectCode} · Grade {score.gradeLevel}
              </p>
              <dl className="mt-1 grid grid-cols-2 gap-x-3 gap-y-0.5 text-xs text-gray-500">
                <dt>CLO coverage</dt>
                <dd className="text-right font-mono">{formatPercent(score.cloCoveragePct)}</dd>
                <dt>Student mastery</dt>
                <dd className="text-right font-mono">{formatPercent(score.studentMasteryPct)}</dd>
                <dt>Exam quality</dt>
                <dd className="text-right font-mono">{formatPercent(score.examQualityAvg)}</dd>
                <dt>Curriculum compliance</dt>
                <dd className="text-right font-mono">{formatPercent(score.curriculumCompliance)}</dd>
              </dl>
              <div className="mt-2">
                {score.flaggedForReview ? (
                  <StatusPill tone="alert">Flagged for ministry review</StatusPill>
                ) : (
                  <Seal label="Certified" meta={formatDate(score.computedAt)} tone="health" className="h-14 w-14" />
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
