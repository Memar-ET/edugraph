import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { GitBranch, ChevronRight, Link2 } from 'lucide-react'
import { Link } from '@tanstack/react-router'

import { AppShell } from '@components/layout'
import { Spinner, Banner, EmptyState } from '@components/ui'
import { listSubjects, getSubjectVersions, supersedeSubject } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function PublishVersionsPage() {
  const qc = useQueryClient()
  const [selectedCode, setSelectedCode] = useState<string>('')
  const [newCode, setNewCode] = useState('')
  const [oldCode, setOldCode] = useState('')
  const [superseding, setSuperseding] = useState(false)

  const { data: subjects, isLoading: loadingSubjects } = useQuery({
    queryKey: queryKeys.subjects(),
    queryFn: listSubjects,
  })

  const { data: versions, isLoading: loadingVersions, isError: versionsError } = useQuery({
    queryKey: queryKeys.subjectVersions(selectedCode),
    queryFn: () => getSubjectVersions(selectedCode),
    enabled: Boolean(selectedCode),
  })

  const supersedeMut = useMutation({
    mutationFn: () => supersedeSubject(newCode.trim(), oldCode.trim()),
    onSuccess: () => {
      setSuperseding(false)
      setNewCode('')
      setOldCode('')
      void qc.invalidateQueries({ queryKey: queryKeys.subjects() })
      if (selectedCode) void qc.invalidateQueries({ queryKey: queryKeys.subjectVersions(selectedCode) })
    },
  })

  return (
    <AppShell
      title="Curriculum Version Control"
      description="Browse subject lineages, link superseding versions, and inspect downstream impact."
    >
      <div className="grid grid-cols-1 gap-5 lg:grid-cols-3">
        {/* Subject list */}
        <div className="space-y-3">
          <h3 className="text-xs font-bold uppercase tracking-wider text-slate-500">Subjects</h3>
          {loadingSubjects ? (
            <div className="flex items-center gap-2 py-6 text-xs text-slate-500"><Spinner /> Loading...</div>
          ) : (
            <div className="space-y-1.5 max-h-[500px] overflow-y-auto">
              {(subjects ?? []).map((s) => (
                <button
                  key={s.code}
                  type="button"
                  onClick={() => setSelectedCode(s.code)}
                  className={`w-full text-left rounded-xl border px-4 py-3 text-xs transition-all ${
                    selectedCode === s.code
                      ? 'border-teal-600 bg-teal-50/50 shadow-sm'
                      : 'border-slate-200 bg-white hover:border-slate-300'
                  }`}
                >
                  <span className="font-mono font-bold text-teal-700 block">{s.code}</span>
                  <span className="text-slate-700 mt-0.5 block">{s.nameEn ?? s.code}</span>
                  <span className="text-slate-400 text-[11px]">
                    Grade {s.gradeLevel}
                    {s.isCurrent === false && (
                      <span className="ml-2 text-rose-500 font-bold">Superseded</span>
                    )}
                  </span>
                </button>
              ))}
              {(subjects ?? []).length === 0 && (
                <EmptyState title="No subjects" description="Approve curriculum jobs to create subjects." />
              )}
            </div>
          )}
        </div>

        {/* Version lineage + supersede panel */}
        <div className="lg:col-span-2 space-y-4">
          {!selectedCode ? (
            <div className="rounded-2xl border border-slate-200 bg-white p-8 flex flex-col items-center justify-center text-center text-slate-400 min-h-[300px]">
              <GitBranch className="h-10 w-10 mb-3 stroke-[1.5]" />
              <p className="text-sm font-medium">Select a subject to view its version lineage</p>
            </div>
          ) : (
            <>
              {loadingVersions && (
                <div className="flex items-center gap-2 py-6 text-xs text-slate-500"><Spinner /> Loading versions...</div>
              )}
              {versionsError && <Banner tone="error">Could not load versions for {selectedCode}.</Banner>}
              {versions && !loadingVersions && (
                <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm space-y-4">
                  <div className="flex items-center justify-between">
                    <h3 className="font-bold text-sm text-slate-900">Version Lineage: {selectedCode}</h3>
                    <Link
                      to="/curriculum/versions"
                      className="flex items-center gap-1 text-xs text-teal-700 font-semibold hover:underline"
                    >
                      <ChevronRight className="h-3.5 w-3.5" />
                      Full Version Manager
                    </Link>
                  </div>

                  {versions.length === 0 ? (
                    <EmptyState title="No version history" description="This subject has no linked predecessors." />
                  ) : (
                    <div className="space-y-3">
                      {versions.map((v) => (
                        <div key={v.code} className={`rounded-xl border p-4 text-xs ${v.isCurrent ? 'border-teal-200 bg-teal-50/40' : 'border-slate-100 bg-slate-50'}`}>
                          <div className="flex items-center justify-between">
                            <span className="font-mono font-bold text-teal-700">{v.code}</span>
                            <span className={`rounded-full px-2 py-0.5 text-[10px] font-bold uppercase ${v.isCurrent ? 'bg-emerald-100 text-emerald-800' : 'bg-slate-200 text-slate-600'}`}>
                              {v.isCurrent ? 'Current' : 'Archived'}
                            </span>
                          </div>
                          <p className="text-slate-500 mt-1">Version {v.version ?? 1}</p>
                          {v.supersededAt && (
                            <p className="text-rose-600 mt-0.5">Superseded: {new Date(v.supersededAt).toLocaleDateString()}</p>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              {/* Link a new version */}
              <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
                <div className="flex items-center gap-2 mb-4">
                  <Link2 className="h-4 w-4 text-teal-700" />
                  <h3 className="font-bold text-sm text-slate-900">Link a Superseding Version</h3>
                </div>
                {superseding ? (
                  <div className="space-y-3">
                    <div>
                      <label className="text-xs font-semibold text-slate-700 block mb-1">New Subject Code (the replacement)</label>
                      <input
                        type="text"
                        value={newCode}
                        onChange={(e) => setNewCode(e.target.value)}
                        placeholder="e.g. BIO-G9-2027"
                        className="w-full rounded-xl border border-slate-200 px-3 py-2 text-xs focus:border-teal-500 focus:outline-none"
                      />
                    </div>
                    <div>
                      <label className="text-xs font-semibold text-slate-700 block mb-1">Previous Subject Code (being replaced)</label>
                      <input
                        type="text"
                        value={oldCode}
                        onChange={(e) => setOldCode(e.target.value)}
                        placeholder="e.g. BIO-G9"
                        className="w-full rounded-xl border border-slate-200 px-3 py-2 text-xs focus:border-teal-500 focus:outline-none"
                      />
                    </div>
                    {supersedeMut.isError && <Banner tone="error">Could not link versions. Check that both codes exist.</Banner>}
                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={() => supersedeMut.mutate()}
                        disabled={!newCode.trim() || !oldCode.trim() || supersedeMut.isPending}
                        className="rounded-xl bg-teal-700 px-4 py-1.5 text-xs font-semibold text-white hover:bg-teal-800 disabled:opacity-50"
                      >
                        {supersedeMut.isPending ? 'Linking...' : 'Link Versions'}
                      </button>
                      <button
                        type="button"
                        onClick={() => setSuperseding(false)}
                        className="rounded-xl border border-slate-200 px-4 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50"
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                ) : (
                  <div>
                    <p className="text-xs text-slate-500 mb-3">
                      Use this when a mid-year revision was approved under a new subject code and you need to mark the old version as superseded.
                    </p>
                    <button
                      type="button"
                      onClick={() => setSuperseding(true)}
                      className="rounded-xl bg-slate-900 px-4 py-1.5 text-xs font-semibold text-white hover:bg-slate-800"
                    >
                      Link Superseding Version
                    </button>
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </AppShell>
  )
}
