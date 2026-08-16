import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Brain, TrendingDown, TrendingUp, Minus } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { getMySkillStates } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'
import type { SkillState } from '@/types/api'

function masteryColor(status: string) {
  switch (status) {
    case 'mastered': return 'bg-emerald-50 text-emerald-700 border-emerald-100'
    case 'learning': return 'bg-blue-50 text-blue-700 border-blue-100'
    case 'at_risk': return 'bg-red-50 text-red-700 border-red-100'
    default: return 'bg-gray-50 text-gray-500 border-gray-100'
  }
}

function TrendIcon({ trend }: { trend?: string }) {
  if (trend === 'improving') return <TrendingUp className="h-3.5 w-3.5 text-emerald-500" />
  if (trend === 'declining') return <TrendingDown className="h-3.5 w-3.5 text-red-500" />
  return <Minus className="h-3.5 w-3.5 text-gray-400" />
}

function SkillCard({ skill, studentId }: { skill: SkillState; studentId: string }) {
  const navigate = useNavigate()

  const pct = skill.masteryProbability !== undefined
    ? Math.round(skill.masteryProbability * 100)
    : null

  return (
    <div
      className={`rounded-xl border p-4 cursor-pointer hover:shadow-sm transition-shadow ${masteryColor(skill.masteryStatus)}`}
      onClick={() =>
        void navigate({
          to: '/students/$studentId/topics/$topicId/explain',
          params: { studentId, topicId: skill.topicId },
        })
      }
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <p className="font-semibold text-sm truncate">{skill.topicTitle}</p>
          <p className="text-[11px] opacity-70 font-mono mt-0.5">
            {skill.subjectCode} · Grade {skill.gradeLevel}
          </p>
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          <TrendIcon trend={skill.trend} />
          {pct !== null && (
            <span className="text-sm font-bold">{pct}%</span>
          )}
        </div>
      </div>
      <div className="mt-2 flex items-center gap-3 text-[11px] opacity-70">
        <span className="capitalize">{skill.masteryStatus.replace('_', ' ')}</span>
        <span>·</span>
        <span>{skill.evidenceCount} evidence point{skill.evidenceCount !== 1 ? 's' : ''}</span>
        {skill.forgettingRisk !== undefined && skill.forgettingRisk > 0.5 && (
          <>
            <span>·</span>
            <span className="text-amber-600 font-medium">Forgetting risk</span>
          </>
        )}
      </div>
    </div>
  )
}

export function StudentSkillStatePage() {
  const user = useAuthStore((s) => s.user)

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.mySkillStates(),
    queryFn: getMySkillStates,
    staleTime: 2 * 60_000,
  })

  const byStatus = data
    ? {
        at_risk: data.skillStates.filter((s) => s.masteryStatus === 'at_risk'),
        learning: data.skillStates.filter((s) => s.masteryStatus === 'learning'),
        mastered: data.skillStates.filter((s) => s.masteryStatus === 'mastered'),
        unknown: data.skillStates.filter(
          (s) => !['at_risk', 'learning', 'mastered'].includes(s.masteryStatus),
        ),
      }
    : null

  const studentId = user?.id ?? ''

  return (
    <AppShell
      title="My Knowledge Map"
      description="Your EG-GCKT learner model — one card per topic where evidence has been gathered. Click any card to see the full explanation."
    >
      <div className="flex items-center gap-2 mb-2">
        <Brain className="h-4 w-4 text-indigo-500" />
        <span className="text-xs text-gray-500 font-medium">
          Powered by EG-GCKT · Cold-start topics (no exams yet) won't appear here
        </span>
      </div>

      {isLoading && (
        <div className="flex items-center gap-2 py-10 text-xs text-gray-500">
          <Spinner /> Loading skill states...
        </div>
      )}
      {isError && (
        <Banner tone="error">{apiErrorMessage(error, 'Could not load skill states.')}</Banner>
      )}

      {byStatus && data && data.skillStates.length === 0 && (
        <EmptyState
          title="No skill data yet"
          description="Complete an exam and get it graded to see your knowledge map here."
        />
      )}

      {byStatus && data && data.skillStates.length > 0 && (
        <div className="space-y-8">
          {byStatus.at_risk.length > 0 && (
            <Section title="Needs Attention" items={byStatus.at_risk} studentId={studentId} />
          )}
          {byStatus.learning.length > 0 && (
            <Section title="In Progress" items={byStatus.learning} studentId={studentId} />
          )}
          {byStatus.mastered.length > 0 && (
            <Section title="Mastered" items={byStatus.mastered} studentId={studentId} />
          )}
          {byStatus.unknown.length > 0 && (
            <Section title="Other" items={byStatus.unknown} studentId={studentId} />
          )}
        </div>
      )}
    </AppShell>
  )
}

function Section({ title, items, studentId }: { title: string; items: SkillState[]; studentId: string }) {
  return (
    <section>
      <h2 className="mb-3 text-xs font-bold uppercase tracking-wide text-gray-500">
        {title} ({items.length})
      </h2>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {items.map((skill) => (
          <SkillCard key={skill.topicId} skill={skill} studentId={studentId} />
        ))}
      </div>
    </section>
  )
}
