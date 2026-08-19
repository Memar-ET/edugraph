import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { BookOpen, ChevronDown, ChevronRight, RefreshCw } from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  Banner,
  Card,
  CardContent,
  ConfirmDialog,
  EmptyState,
  Spinner,
  useToast,
} from '@components/ui'
import { generateStudyPlan, listMyStudyPlans } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { cn } from '@lib/utils/cn'
import type { StudyPlanBlock } from '@/types/api'
import { useAuthStore } from '@stores/auth.store'

const ACTION_LABELS: Record<string, string> = {
  practice: 'Practice',
  diagnostic: 'Diagnostic',
  prerequisite_review: 'Prereq Review',
  concept_explanation: 'Concept',
  alternative_representation: 'Alternative',
  spaced_review: 'Spaced Review',
  teacher_escalation: 'Escalate',
}

const MASTERY_COLORS: Record<string, string> = {
  mastered: 'text-health-700 bg-health-50 border-health-200',
  proficient: 'text-health-600 bg-health-50 border-health-100',
  emerging: 'text-seal-700 bg-seal-50 border-seal-200',
  at_risk: 'text-alert-700 bg-alert-50 border-alert-200',
  unknown: 'text-gray-500 bg-gray-50 border-gray-200',
  learning: 'text-seal-700 bg-seal-50 border-seal-200',
}

export function StudyPlanPage() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { toast } = useToast()
  const user = useAuthStore((s) => s.user)
  const [expandedDays, setExpandedDays] = useState<Set<number | string>>(new Set([1]))
  const [confirmRegenerate, setConfirmRegenerate] = useState(false)

  const { data: plans, isLoading, isError } = useQuery({
    queryKey: queryKeys.myStudyPlans(),
    queryFn: listMyStudyPlans,
  })

  const generateMutation = useMutation({
    mutationFn: () => generateStudyPlan({}),
    onSuccess: () => {
      toast({ title: 'Study plan generated', variant: 'success' })
      void qc.invalidateQueries({ queryKey: queryKeys.myStudyPlans() })
      setConfirmRegenerate(false)
    },
    onError: () => {
      toast({
        title: 'Could not generate plan',
        description: 'Make sure you have completed at least one exam first.',
        variant: 'error',
      })
    },
  })

  const activePlan = plans?.[0] ?? null

  const toggleDay = (day: number | string) => {
    setExpandedDays((prev) => {
      const next = new Set(prev)
      if (next.has(day)) { next.delete(day) } else { next.add(day) }
      return next
    })
  }

  return (
    <AppShell>
      <div className="space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">My Study Plan</h1>
            <p className="mt-1 text-sm text-gray-500">
              AI-generated day-by-day learning schedule based on your gap analysis.
            </p>
          </div>
          {activePlan && (
            <button
              onClick={() => setConfirmRegenerate(true)}
              className="flex items-center gap-2 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <RefreshCw className="h-4 w-4" aria-hidden />
              Regenerate
            </button>
          )}
        </div>

        {isError && (
          <Banner tone="error" title="Failed to load study plan" description="Try refreshing the page." />
        )}

        {isLoading ? (
          <div className="flex justify-center py-16">
            <Spinner />
          </div>
        ) : !activePlan ? (
          <EmptyState
            title="No study plan yet"
            description={
              generateMutation.isPending
                ? 'Generating your personalised plan…'
                : 'Complete at least one exam so the AI can analyse your gaps and build a schedule.'
            }
            action={
              !generateMutation.isPending ? (
                <button
                  onClick={() => generateMutation.mutate()}
                  className="rounded-lg bg-primary-700 px-4 py-2 text-sm text-white hover:bg-primary-600 focus:outline-none focus:ring-2 focus:ring-primary-500"
                >
                  Generate my study plan
                </button>
              ) : (
                <Spinner />
              )
            }
          />
        ) : (
          <>
            {/* Plan summary */}
            <div className="grid grid-cols-3 gap-4">
              {[
                { label: 'Total days', value: activePlan.totalDays },
                { label: 'Total hours', value: `${activePlan.totalHours}h` },
                { label: 'Language', value: activePlan.language.toUpperCase() },
              ].map((k) => (
                <Card key={k.label}>
                  <CardContent className="py-4 text-center">
                    <p className="text-2xl font-bold text-primary-700">{k.value}</p>
                    <p className="mt-0.5 text-xs text-gray-500">{k.label}</p>
                  </CardContent>
                </Card>
              ))}
            </div>

            {activePlan.planData.summary && (
              <Card>
                <CardContent className="py-3 text-sm text-gray-700">
                  {activePlan.planData.summary}
                </CardContent>
              </Card>
            )}

            {/* Day accordion */}
            <div className="space-y-3">
              {activePlan.planData.days.map((dayPlan) => {
                const isOpen = expandedDays.has(dayPlan.day)
                return (
                  <Card key={dayPlan.day} className="overflow-hidden">
                    <button
                      onClick={() => toggleDay(dayPlan.day)}
                      aria-expanded={isOpen}
                      className="flex w-full items-center justify-between px-6 py-4 text-left hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-primary-500"
                    >
                      <div className="flex items-center gap-3">
                        <BookOpen className="h-5 w-5 text-primary-600" aria-hidden />
                        <span className="font-semibold text-gray-900">Day {dayPlan.day}</span>
                        <span className="text-sm text-gray-500">
                          {dayPlan.blocks.length} topic{dayPlan.blocks.length !== 1 ? 's' : ''}
                        </span>
                      </div>
                      {isOpen ? (
                        <ChevronDown className="h-4 w-4 text-gray-400" aria-hidden />
                      ) : (
                        <ChevronRight className="h-4 w-4 text-gray-400" aria-hidden />
                      )}
                    </button>
                    {isOpen && (
                      <div className="divide-y divide-gray-50 border-t border-gray-100">
                        {dayPlan.blocks.map((block, i) => (
                          <TopicBlock
                            key={i}
                            block={block}
                            studentId={user?.id}
                            onExplain={(topicId) =>
                              navigate({
                                to: '/students/$studentId/topics/$topicId/explain',
                                params: { studentId: user?.id ?? '', topicId },
                              })
                            }
                          />
                        ))}
                      </div>
                    )}
                  </Card>
                )
              })}
            </div>
          </>
        )}
      </div>

      <ConfirmDialog
        open={confirmRegenerate}
        onOpenChange={setConfirmRegenerate}
        title="Regenerate study plan?"
        description="This will replace your current plan with a new one based on your latest exam results. Your old plan will be lost."
        confirmLabel="Regenerate"
        onConfirm={() => generateMutation.mutate()}
        destructive
        loading={generateMutation.isPending}
      />
    </AppShell>
  )
}

function TopicBlock({
  block,
  studentId,
  onExplain,
}: {
  block: StudyPlanBlock
  studentId?: string
  onExplain: (topicId: string) => void
}) {
  const masteryKey = 'unknown'
  const colorClass = MASTERY_COLORS[masteryKey] ?? MASTERY_COLORS.unknown
  const actionLabel = block.isRootCause ? 'prerequisite_review' : 'practice'

  return (
    <div className="flex items-start gap-4 px-6 py-4">
      <div className="flex-1 space-y-1">
        <p className="font-medium text-gray-900">{block.title ?? 'Topic'}</p>
        {block.why && <p className="text-xs text-gray-500">{block.why}</p>}
        <div className="flex items-center gap-2 pt-1">
          {block.hours !== undefined && (
            <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
              {block.hours}h
            </span>
          )}
          <span className={cn('rounded-full border px-2 py-0.5 text-xs font-medium', colorClass)}>
            {ACTION_LABELS[actionLabel] ?? actionLabel}
          </span>
          {block.isRootCause && (
            <span className="rounded-full bg-alert-50 px-2 py-0.5 text-xs text-alert-700 border border-alert-200">
              Root cause
            </span>
          )}
        </div>
      </div>
      {block.topicId && studentId && (
        <button
          onClick={() => onExplain(block.topicId!)}
          className="shrink-0 rounded-lg border border-primary-200 px-3 py-1.5 text-xs font-medium text-primary-700 hover:bg-primary-50 focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          Explain
        </button>
      )}
    </div>
  )
}
