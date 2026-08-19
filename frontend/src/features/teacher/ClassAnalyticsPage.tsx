import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Users, TrendingDown } from 'lucide-react'
import { AppShell } from '@components/layout'
import { getClassHeatmap, listExams } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { ClassHeatmapResponse } from '@/types/api'

function severity(pct: number): string {
  if (pct >= 0.6) return 'bg-rose-500'
  if (pct >= 0.4) return 'bg-amber-400'
  if (pct >= 0.2) return 'bg-yellow-300'
  return 'bg-emerald-300'
}

function HeatmapGrid({ data }: { data: ClassHeatmapResponse }) {
  const sorted = [...data.topics].sort((a, b) => b.strugglingPct - a.strugglingPct)
  return (
    <div className="space-y-2">
      {sorted.slice(0, 20).map((topic) => (
        <div key={topic.topicId} className="flex items-center gap-3">
          <div className="w-48 shrink-0 truncate text-xs text-slate-700 font-medium" title={topic.title}>
            {topic.title}
          </div>
          <div className="flex-1 h-5 rounded-full bg-slate-100 overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${severity(topic.strugglingPct)}`}
              style={{ width: `${Math.round(topic.strugglingPct * 100)}%` }}
            />
          </div>
          <div className="w-12 shrink-0 text-right text-xs font-mono text-slate-500">
            {Math.round(topic.strugglingPct * 100)}%
          </div>
          {topic.alert && (
            <span title={topic.alert.message}>
              <AlertTriangle className="h-4 w-4 shrink-0 text-amber-500" />
            </span>
          )}
        </div>
      ))}
    </div>
  )
}

export function ClassAnalyticsPage() {
  const [selectedSubject, setSelectedSubject] = useState('')
  const [selectedGrade, setSelectedGrade] = useState(0)

  const { data: examsData, isLoading: examsLoading } = useQuery({
    queryKey: queryKeys.exams(),
    queryFn: () => listExams(1, 100),
    select: (d) => {
      const combos = new Map<string, { subjectCode: string; gradeLevel: number }>()
      for (const e of d.items) {
        const key = `${e.subjectCode}-${e.gradeLevel}`
        if (!combos.has(key)) combos.set(key, { subjectCode: e.subjectCode, gradeLevel: e.gradeLevel })
      }
      return Array.from(combos.values())
    },
  })

  const combos = examsData ?? []
  const effectiveSubject = selectedSubject || combos[0]?.subjectCode || ''
  const effectiveGrade = selectedGrade || combos[0]?.gradeLevel || 0

  const { data: heatmap, isLoading: heatmapLoading, isError } = useQuery({
    queryKey: queryKeys.classHeatmap(effectiveSubject, effectiveGrade),
    queryFn: () => getClassHeatmap(effectiveSubject, effectiveGrade),
    enabled: !!effectiveSubject && !!effectiveGrade,
  })

  const masteredCount = heatmap
    ? heatmap.topics.filter((t) => t.strugglingPct < 0.2).length
    : 0
  const atRiskCount = heatmap
    ? heatmap.topics.filter((t) => t.strugglingPct >= 0.4).length
    : 0

  return (
    <AppShell
      title="Class Mastery Analytics"
      description="Class-wide gap heatmap and AI intervention recommendations."
    >
      <div className="space-y-6">
        {/* Subject/Grade selector */}
        <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
          <div className="flex flex-wrap items-center gap-3">
            <span className="text-xs font-bold text-slate-700">Select class:</span>
            {examsLoading ? (
              <div className="h-8 w-48 animate-pulse rounded-xl bg-slate-100" />
            ) : combos.length === 0 ? (
              <span className="text-xs text-slate-400">No exams found — upload an exam to see analytics.</span>
            ) : (
              combos.map((c) => {
                const key = `${c.subjectCode}-${c.gradeLevel}`
                const active =
                  (effectiveSubject === c.subjectCode && effectiveGrade === c.gradeLevel)
                return (
                  <button
                    key={key}
                    type="button"
                    onClick={() => {
                      setSelectedSubject(c.subjectCode)
                      setSelectedGrade(c.gradeLevel)
                    }}
                    className={`rounded-xl px-3 py-1.5 text-xs font-semibold transition-colors ${
                      active
                        ? 'bg-teal-700 text-white'
                        : 'border border-slate-200 text-slate-700 hover:bg-slate-50'
                    }`}
                  >
                    {c.subjectCode} G{c.gradeLevel}
                  </button>
                )
              })
            )}
          </div>
        </div>

        {/* KPI strip */}
        {heatmap && (
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
              <p className="text-xs font-medium text-slate-500">Class Size</p>
              <p className="mt-1 text-2xl font-bold text-slate-900 font-display">{heatmap.classSize}</p>
              <p className="mt-1 text-[11px] text-slate-500">{heatmap.subjectCode} G{heatmap.gradeLevel}</p>
            </div>
            <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
              <p className="text-xs font-medium text-slate-500">Topics Tracked</p>
              <p className="mt-1 text-2xl font-bold text-slate-900 font-display">{heatmap.topics.length}</p>
              <p className="mt-1 text-[11px] text-slate-500">via {heatmap.source}</p>
            </div>
            <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
              <p className="text-xs font-medium text-slate-500">Mastered Topics</p>
              <p className="mt-1 text-2xl font-bold text-emerald-700 font-display">{masteredCount}</p>
              <p className="mt-1 text-[11px] text-emerald-600">&lt;20% struggling</p>
            </div>
            <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
              <p className="text-xs font-medium text-slate-500">At-Risk Topics</p>
              <p className="mt-1 text-2xl font-bold text-rose-600 font-display">{atRiskCount}</p>
              <p className="mt-1 text-[11px] text-amber-600">≥40% struggling</p>
            </div>
          </div>
        )}

        {/* Alerts */}
        {heatmap && heatmap.alerts.length > 0 && (
          <div className="space-y-2">
            {heatmap.alerts.map((alert, i) => (
              <div
                key={i}
                className="rounded-2xl border border-amber-200 bg-amber-50 p-4 flex items-start gap-3"
              >
                <AlertTriangle className="h-5 w-5 text-amber-600 shrink-0 mt-0.5" />
                <div className="text-xs text-amber-900">
                  <p className="font-bold">{alert.topicTitle}</p>
                  <p>{alert.message}</p>
                  <p className="text-amber-700 mt-1">
                    Root cause: <strong>{alert.rootCauseTitle}</strong> (G{alert.rootCauseGradeLevel}){' '}
                    — {alert.rootCauseStudentsStruggling} students affected
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Heatmap */}
        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <div className="flex items-center gap-2 mb-4">
            <TrendingDown className="h-4 w-4 text-teal-700" />
            <h3 className="font-bold text-sm text-slate-900">Topic Struggle Heatmap</h3>
            {heatmap && (
              <span className="ml-auto text-[10px] text-slate-400 font-mono">
                {heatmap.topics.length} topics
              </span>
            )}
          </div>

          {isError && (
            <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800">
              Failed to load heatmap. Ensure students have completed graded exams for this subject.
            </div>
          )}

          {heatmapLoading ? (
            <div className="space-y-2">
              {[...Array(8)].map((_, i) => (
                <div key={i} className="flex items-center gap-3">
                  <div className="h-4 w-48 animate-pulse rounded bg-slate-100" />
                  <div className="h-5 flex-1 animate-pulse rounded-full bg-slate-100" />
                  <div className="h-4 w-10 animate-pulse rounded bg-slate-100" />
                </div>
              ))}
            </div>
          ) : !heatmap || heatmap.topics.length === 0 ? (
            <div className="py-12 text-center">
              <Users className="mx-auto h-10 w-10 text-slate-200 mb-3" />
              <p className="text-sm font-bold text-slate-500">No struggle data yet</p>
              <p className="text-xs text-slate-400 mt-1">
                Heatmap populates once students submit graded exams for this subject and grade.
              </p>
            </div>
          ) : (
            <HeatmapGrid data={heatmap} />
          )}

          {/* Legend */}
          <div className="flex items-center gap-4 mt-4 pt-4 border-t border-slate-100">
            <span className="text-[10px] font-bold text-slate-400 uppercase">Struggle %</span>
            {[
              { label: '<20%', cls: 'bg-emerald-300' },
              { label: '20-40%', cls: 'bg-yellow-300' },
              { label: '40-60%', cls: 'bg-amber-400' },
              { label: '≥60%', cls: 'bg-rose-500' },
            ].map((item) => (
              <div key={item.label} className="flex items-center gap-1.5">
                <div className={`h-3 w-3 rounded-sm ${item.cls}`} />
                <span className="text-[10px] text-slate-500">{item.label}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </AppShell>
  )
}
