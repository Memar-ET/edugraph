import { Brain, MessageSquare, Layers } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { AppShell } from '@components/layout'
import { getMySkillStates } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { SkillState } from '@/types/api'

function statusColor(status: string) {
  switch (status) {
    case 'mastered': return 'bg-emerald-100 text-emerald-800'
    case 'proficient': return 'bg-teal-100 text-teal-800'
    case 'emerging': return 'bg-blue-100 text-blue-800'
    case 'unknown': return 'bg-slate-100 text-slate-500'
    default: return 'bg-amber-100 text-amber-800'
  }
}

function statusLabel(status: string) {
  switch (status) {
    case 'mastered': return 'Mastered'
    case 'proficient': return 'Proficient'
    case 'emerging': return 'Emerging'
    case 'unknown': return 'Not Started'
    default: return status
  }
}

function barColor(status: string) {
  switch (status) {
    case 'mastered': return 'bg-emerald-500'
    case 'proficient': return 'bg-teal-500'
    case 'emerging': return 'bg-blue-500'
    default: return 'bg-slate-300'
  }
}

export function StudentSkillMapPage() {
  const navigate = useNavigate()
  const [selectedSkill, setSelectedSkill] = useState<SkillState | null>(null)

  const { data, isLoading, error } = useQuery({
    queryKey: queryKeys.mySkillStates(),
    queryFn: getMySkillStates,
  })

  const skills = data?.skillStates ?? []

  if (isLoading) {
    return (
      <AppShell title="Interactive Knowledge & Skill Map">
        <div className="space-y-3 animate-pulse">
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className="h-20 bg-slate-200 rounded-xl" />
          ))}
        </div>
      </AppShell>
    )
  }

  if (error) {
    return (
      <AppShell title="Interactive Knowledge & Skill Map">
        <div className="rounded-2xl border border-rose-200 bg-rose-50 p-5 text-rose-800 text-sm">
          Failed to load skill map: {(error as Error).message}
        </div>
      </AppShell>
    )
  }

  if (skills.length === 0) {
    return (
      <AppShell
        title="Interactive Knowledge & Skill Map"
        description="Visual dependency graph of your curriculum topic mastery and knowledge chains."
        actions={
          <button
            type="button"
            onClick={() => void navigate({ to: '/student/tutor' })}
            className="flex items-center gap-1.5 rounded-lg bg-teal-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-teal-800"
          >
            <Brain className="h-3.5 w-3.5" /> Launch AI Tutor
          </button>
        }
      >
        <div className="rounded-2xl border border-slate-200 bg-white p-10 text-center shadow-sm">
          <Layers className="mx-auto h-10 w-10 text-slate-300 mb-3" />
          <p className="font-semibold text-slate-700">No skill data yet</p>
          <p className="text-xs text-slate-500 mt-1 max-w-sm mx-auto">
            Your skill states will appear here after you take and submit exams. Each answer builds your personal knowledge map.
          </p>
          <button
            type="button"
            onClick={() => void navigate({ to: '/student/exams' })}
            className="mt-4 inline-flex items-center gap-1.5 rounded-xl bg-teal-700 px-4 py-2 text-xs font-semibold text-white hover:bg-teal-800"
          >
            View Available Exams
          </button>
        </div>
      </AppShell>
    )
  }

  const active = selectedSkill ?? skills[0]!

  return (
    <AppShell
      title="Interactive Knowledge & Skill Map"
      description="Visual dependency graph of your curriculum topic mastery, knowledge chains, and root cause gaps."
      actions={
        <button
          type="button"
          onClick={() => void navigate({ to: '/student/tutor' })}
          className="flex items-center gap-1.5 rounded-lg bg-teal-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-teal-800"
        >
          <Brain className="h-3.5 w-3.5" /> Launch AI Tutor
        </button>
      }
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="space-y-3 lg:col-span-2">
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <h3 className="text-sm font-bold text-slate-900 mb-4">Your Skill States ({skills.length} topics)</h3>
            <div className="space-y-3 max-h-[60vh] overflow-y-auto pr-1">
              {skills.map((node) => {
                const isSelected = active.topicId === node.topicId
                const pct = node.masteryProbability != null ? Math.round(node.masteryProbability * 100) : 0
                return (
                  <div
                    key={node.topicId}
                    onClick={() => setSelectedSkill(node)}
                    className={`cursor-pointer rounded-xl border p-4 transition-all ${
                      isSelected
                        ? 'border-teal-600 bg-teal-50/40 shadow-sm ring-1 ring-teal-600'
                        : 'border-slate-200 bg-white hover:border-slate-300'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2 min-w-0">
                        <span className="font-mono text-xs font-bold text-teal-700 shrink-0">{node.subjectCode}</span>
                        <h4 className="font-bold text-xs text-slate-900 truncate">{node.topicTitle}</h4>
                      </div>
                      <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-bold shrink-0 ml-2 ${statusColor(node.masteryStatus)}`}>
                        {statusLabel(node.masteryStatus)}
                      </span>
                    </div>
                    <div className="mt-3 flex items-center justify-between gap-3">
                      <div className="h-2 flex-1 overflow-hidden rounded-full bg-slate-100">
                        <div
                          className={`h-full rounded-full ${barColor(node.masteryStatus)}`}
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                      <span className="font-mono text-xs font-bold text-slate-700 shrink-0">{pct}%</span>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm h-fit">
          <span className="font-mono text-xs font-bold text-teal-700">{active.subjectCode} · G{active.gradeLevel}</span>
          <h3 className="text-base font-bold text-slate-900 mt-1">{active.topicTitle}</h3>

          <div className="mt-4 space-y-2 text-xs">
            <div className="flex justify-between rounded-lg bg-slate-50 p-2 text-slate-600">
              <span>Mastery</span>
              <span className="font-bold text-slate-900">
                {active.masteryProbability != null ? `${Math.round(active.masteryProbability * 100)}%` : 'Unknown'}
              </span>
            </div>
            <div className="flex justify-between rounded-lg bg-slate-50 p-2 text-slate-600">
              <span>Status</span>
              <span className="font-semibold text-slate-900">{statusLabel(active.masteryStatus)}</span>
            </div>
            <div className="flex justify-between rounded-lg bg-slate-50 p-2 text-slate-600">
              <span>Evidence</span>
              <span className="font-bold text-slate-900">{active.evidenceCount} answers</span>
            </div>
            {active.trend && (
              <div className="flex justify-between rounded-lg bg-slate-50 p-2 text-slate-600">
                <span>Trend</span>
                <span className={`font-semibold capitalize ${
                  active.trend === 'improving' ? 'text-emerald-700' : active.trend === 'declining' ? 'text-rose-700' : 'text-slate-700'
                }`}>{active.trend}</span>
              </div>
            )}
          </div>

          <div className="mt-5">
            <button
              type="button"
              onClick={() => void navigate({ to: '/student/tutor' })}
              className="w-full flex items-center justify-center gap-2 rounded-lg bg-teal-700 py-2 text-xs font-semibold text-white hover:bg-teal-800"
            >
              <MessageSquare className="h-3.5 w-3.5" /> Ask AI Tutor
            </button>
          </div>
        </div>
      </div>
    </AppShell>
  )
}
