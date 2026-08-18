import { useState } from 'react'
import { BookOpen } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { listSubjects, getQMatrixQuality, getPrerequisiteQuality } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function CurriculumCoveragePage() {
  const [selectedCode, setSelectedCode] = useState('')

  const { data: subjects, isLoading: subjectsLoading } = useQuery({
    queryKey: queryKeys.subjects(),
    queryFn: listSubjects,
  })

  const { data: quality, isLoading: qualityLoading, isError, error } = useQuery({
    queryKey: queryKeys.qmatrixQuality(selectedCode),
    queryFn: () => getQMatrixQuality(selectedCode),
    enabled: !!selectedCode,
  })

  const { data: prereqQuality } = useQuery({
    queryKey: queryKeys.prerequisiteQuality(selectedCode),
    queryFn: () => getPrerequisiteQuality(selectedCode),
    enabled: !!selectedCode,
  })

  const currentSubjects = subjects?.filter((s) => s.isCurrent) ?? []
  const orphanedTopics = prereqQuality?.orphanedTopics ?? []
  const missingMappings = quality?.missingMappings ?? []

  return (
    <AppShell
      title="Curriculum Syllabus Coverage"
      description="Track assessment coverage and Q-matrix completeness against national curriculum CLO standards."
    >
      <div className="space-y-5">
        {/* Subject selector */}
        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm flex items-center gap-3">
          <BookOpen className="h-4 w-4 text-teal-600 flex-shrink-0" />
          <span className="text-xs font-semibold text-slate-700 flex-shrink-0">Subject:</span>
          {subjectsLoading ? (
            <Spinner />
          ) : (
            <select
              value={selectedCode}
              onChange={(e) => setSelectedCode(e.target.value)}
              className="flex-1 rounded-lg border border-slate-200 bg-slate-50 px-3 py-1.5 text-xs text-slate-800 focus:border-teal-500 focus:outline-none"
            >
              <option value="">— Select a subject —</option>
              {currentSubjects.map((s) => (
                <option key={s.code} value={s.code}>
                  {s.code} — Grade {s.gradeLevel} · {s.academicYear}
                </option>
              ))}
            </select>
          )}
        </div>

        {!selectedCode && (
          <EmptyState
            title="Select a subject"
            description="Choose a subject above to see Q-matrix coverage and assessment completeness."
          />
        )}

        {selectedCode && qualityLoading && (
          <div className="flex items-center gap-2 py-8 text-xs text-slate-500">
            <Spinner /> Loading coverage data…
          </div>
        )}

        {selectedCode && isError && (
          <Banner tone="error">{(error as Error).message ?? 'Could not load coverage data.'}</Banner>
        )}

        {quality && (
          <div className="space-y-4">
            {/* Summary KPIs */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm text-center">
                <p className="text-2xl font-bold text-teal-700">{quality.totalQuestions}</p>
                <p className="text-xs text-slate-500 mt-0.5">Total Assessment Items</p>
              </div>
              <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm text-center">
                <p className="text-2xl font-bold text-amber-600">{missingMappings.length}</p>
                <p className="text-xs text-slate-500 mt-0.5">Items Missing CLO Mapping</p>
              </div>
              <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm text-center">
                <p className="text-2xl font-bold text-rose-600">{orphanedTopics.length}</p>
                <p className="text-xs text-slate-500 mt-0.5">Topics Never Assessed</p>
              </div>
            </div>

            {/* Orphaned topics */}
            {orphanedTopics.length > 0 && (
              <div className="rounded-xl border border-amber-200 bg-amber-50 p-5">
                <h3 className="font-bold text-sm text-amber-900 mb-3">Topics with No Exam Coverage</h3>
                <div className="space-y-2">
                  {orphanedTopics.map((t) => (
                    <div key={t.topicId} className="flex items-center justify-between rounded-lg bg-white border border-amber-100 px-3 py-2 text-xs">
                      <span className="font-semibold text-slate-900">{t.title}</span>
                      <span className="font-mono text-amber-700">{t.topicId.slice(0, 8)}…</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Items with missing mappings */}
            {missingMappings.length > 0 && (
              <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <h3 className="font-bold text-sm text-slate-900 mb-3">Items Needing CLO Mapping</h3>
                <div className="space-y-2 max-h-60 overflow-y-auto">
                  {missingMappings.map((item) => (
                    <div key={item.questionId} className="flex items-center justify-between rounded-lg bg-slate-50 border border-slate-100 px-3 py-2 text-xs">
                      <span className="font-medium text-slate-800 truncate max-w-[60%]">{item.questionText}</span>
                      <span className="font-mono text-slate-500">{item.questionId.slice(0, 8)}…</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {orphanedTopics.length === 0 && missingMappings.length === 0 && (
              <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-5 text-center">
                <p className="font-bold text-sm text-emerald-800">Full Coverage</p>
                <p className="text-xs text-emerald-600 mt-0.5">All topics have exam items and all items are mapped to CLOs.</p>
              </div>
            )}
          </div>
        )}
      </div>
    </AppShell>
  )
}
