import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Search, ExternalLink, BarChart2 } from 'lucide-react'
import { Link } from '@tanstack/react-router'

import { AppShell } from '@components/layout'
import { Spinner, Banner, EmptyState } from '@components/ui'
import { listSubjects, getQMatrixQuality, getPrerequisiteQuality } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function CloManagementPage() {
  const [selectedCode, setSelectedCode] = useState<string>('')
  const [search, setSearch] = useState('')

  const { data: subjects, isLoading: loadingSubjects, isError: subjectsError } = useQuery({
    queryKey: queryKeys.subjects(),
    queryFn: listSubjects,
  })

  const { data: qmReport, isLoading: loadingReport, isError: reportError } = useQuery({
    queryKey: queryKeys.qmatrixQuality(selectedCode),
    queryFn: () => getQMatrixQuality(selectedCode),
    enabled: Boolean(selectedCode),
  })

  const { data: prereqReport, isLoading: loadingPrereq } = useQuery({
    queryKey: queryKeys.prerequisiteQuality(selectedCode),
    queryFn: () => getPrerequisiteQuality(selectedCode),
    enabled: Boolean(selectedCode),
  })

  const filteredSubjects = (subjects ?? []).filter((s) =>
    !search ||
    s.code.toLowerCase().includes(search.toLowerCase()) ||
    (s.nameEn ?? s.code).toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <AppShell
      title="CLO Management Hub"
      description="Curriculum Learning Outcomes alignment, Q-matrix quality, and AI confidence analysis."
    >
      <div className="space-y-5">
        {/* Header KPIs from real data */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <p className="text-xs font-medium text-slate-500">Loaded Subjects</p>
            <p className="mt-1 text-2xl font-bold text-slate-900 font-display">
              {loadingSubjects ? '—' : (subjects?.length ?? 0)}
            </p>
            <p className="mt-1 text-[11px] text-slate-500">From approved curriculum</p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <p className="text-xs font-medium text-slate-500">
              {selectedCode ? `Q-Matrix Issues (${selectedCode})` : 'Q-Matrix Issues'}
            </p>
            <p className="mt-1 text-2xl font-bold font-display text-amber-600">
              {loadingReport ? '—' : qmReport ? (
                qmReport.missingMappings.length + qmReport.ambiguousMappings.length
              ) : '—'}
            </p>
            <p className="mt-1 text-[11px] text-slate-500">Select a subject to see details</p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <p className="text-xs font-medium text-slate-500">Orphaned Topics</p>
            <p className="mt-1 text-2xl font-bold font-display text-rose-600">
              {loadingPrereq ? '—' : prereqReport ? prereqReport.orphanedTopics.length : '—'}
            </p>
            <p className="mt-1 text-[11px] text-slate-500">Topics with no prerequisite edges</p>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          {/* Subject Selector */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-bold uppercase tracking-wider text-slate-500">Select Subject</h3>
            </div>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                placeholder="Search subjects..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-xl border border-slate-200 bg-slate-50 py-1.5 pl-8 pr-3 text-xs text-slate-800 focus:border-teal-500 focus:outline-none"
              />
            </div>
            {loadingSubjects && (
              <div className="flex items-center gap-2 py-6 text-xs text-slate-500">
                <Spinner /> Loading subjects...
              </div>
            )}
            {subjectsError && <Banner tone="error">Could not load subjects.</Banner>}
            {!loadingSubjects && filteredSubjects.length === 0 && (
              <EmptyState title="No subjects found" description="Upload and approve curriculum to populate subjects." />
            )}
            <div className="space-y-1.5 max-h-[480px] overflow-y-auto">
              {filteredSubjects.map((s) => (
                <button
                  key={s.code}
                  type="button"
                  onClick={() => setSelectedCode(s.code)}
                  className={`w-full text-left rounded-xl border px-4 py-3 transition-all text-xs ${
                    selectedCode === s.code
                      ? 'border-teal-600 bg-teal-50/50 shadow-sm'
                      : 'border-slate-200 bg-white hover:border-slate-300'
                  }`}
                >
                  <span className="font-mono font-bold text-teal-700 block">{s.code}</span>
                  <span className="text-slate-700 mt-0.5 block">{s.nameEn ?? s.code}</span>
                  <span className="text-slate-400 text-[11px]">Grade {s.gradeLevel}</span>
                </button>
              ))}
            </div>
          </div>

          {/* Q-Matrix Quality Detail */}
          <div className="lg:col-span-2 space-y-4">
            {!selectedCode ? (
              <div className="rounded-2xl border border-slate-200 bg-white p-8 flex flex-col items-center justify-center text-center text-slate-400 min-h-[300px]">
                <BarChart2 className="h-10 w-10 mb-3 stroke-[1.5]" />
                <p className="text-sm font-medium">Select a subject to view CLO quality analysis</p>
                <p className="text-xs mt-1">Q-matrix quality reports show mapping coverage and orphaned topics.</p>
              </div>
            ) : (
              <>
                {loadingReport && (
                  <div className="rounded-2xl border border-slate-200 bg-white p-8 flex items-center justify-center gap-2 text-xs text-slate-500">
                    <Spinner /> Loading Q-matrix quality...
                  </div>
                )}
                {reportError && <Banner tone="error">Could not load Q-matrix quality for {selectedCode}.</Banner>}
                {qmReport && !loadingReport && (
                  <div className="space-y-4">
                    <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
                      <div className="flex items-center justify-between mb-3">
                        <h3 className="font-bold text-sm text-slate-900">Q-Matrix Quality: {selectedCode}</h3>
                        <Link
                          to="/curriculum/qmatrix-quality"
                          className="flex items-center gap-1 text-xs text-teal-700 font-semibold hover:underline"
                        >
                          <ExternalLink className="h-3.5 w-3.5" />
                          Full Report
                        </Link>
                      </div>

                      {qmReport.missingMappings.length === 0 &&
                      qmReport.ambiguousMappings.length === 0 &&
                      (prereqReport?.orphanedTopics.length ?? 0) === 0 ? (
                        <div className="text-xs text-emerald-700 bg-emerald-50 rounded-xl p-4 font-medium">
                          ✓ No Q-matrix quality issues found for this subject.
                        </div>
                      ) : null}

                      {qmReport.missingMappings.length > 0 && (
                        <div>
                          <h4 className="text-xs font-bold uppercase tracking-wider text-slate-500 mb-2">
                            Questions Missing Skill Mappings ({qmReport.missingMappings.length})
                          </h4>
                          <div className="space-y-1.5 max-h-[200px] overflow-y-auto">
                            {qmReport.missingMappings.slice(0, 20).map((item) => (
                              <div key={item.questionId} className="flex items-center justify-between rounded-lg bg-amber-50 border border-amber-100 px-3 py-2 text-xs">
                                <span className="font-medium text-slate-800 truncate">{item.questionText}</span>
                                <span className="text-amber-700 font-bold ml-2 shrink-0">Missing</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}

                      {(prereqReport?.orphanedTopics.length ?? 0) > 0 && (
                        <div className="mt-4">
                          <h4 className="text-xs font-bold uppercase tracking-wider text-slate-500 mb-2">
                            Orphaned Topics ({prereqReport!.orphanedTopics.length})
                          </h4>
                          <div className="space-y-1.5 max-h-[200px] overflow-y-auto">
                            {prereqReport!.orphanedTopics.slice(0, 20).map((item) => (
                              <div key={item.topicId} className="flex items-center justify-between rounded-lg bg-rose-50 border border-rose-100 px-3 py-2 text-xs">
                                <span className="font-medium text-slate-800 truncate">{item.title}</span>
                                <span className="text-rose-700 font-bold ml-2 shrink-0">No prereqs</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    </AppShell>
  )
}
