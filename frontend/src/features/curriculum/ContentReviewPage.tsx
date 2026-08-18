import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronRight, RefreshCw } from 'lucide-react'
import { Link } from '@tanstack/react-router'

import { AppShell } from '@components/layout'
import { Spinner, Banner, EmptyState } from '@components/ui'
import { listCurriculumJobs } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { JobListItem } from '@/types/api'

const STATUS_COLOR: Record<string, string> = {
  approved: 'bg-emerald-100 text-emerald-800',
  parsed: 'bg-teal-100 text-teal-800',
  pending: 'bg-slate-100 text-slate-600',
  parsing: 'bg-blue-100 text-blue-800',
  review: 'bg-amber-100 text-amber-800',
  rejected: 'bg-rose-100 text-rose-800',
  failed: 'bg-rose-100 text-rose-800',
}

export function ContentReviewPage() {
  const qc = useQueryClient()
  const [page] = useState(1)
  const [selected, setSelected] = useState<JobListItem | null>(null)

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.curriculumJobs(page),
    queryFn: () => listCurriculumJobs(page, 30),
    select: (r) => r.items,
  })

  const jobs = data ?? []

  return (
    <AppShell
      title="Content & AST Review Desk"
      description="Audit AI-extracted curriculum trees, textbook parsing accuracy, and verification flags."
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Job Queue */}
        <div className="space-y-3">
          <h3 className="text-xs font-bold uppercase tracking-wider text-slate-500">Document Parsing Queue</h3>
          {isLoading && (
            <div className="flex items-center gap-2 py-8 text-xs text-slate-500">
              <Spinner /> Loading jobs...
            </div>
          )}
          {isError && <Banner tone="error">{(error as Error).message ?? 'Could not load jobs.'}</Banner>}
          {!isLoading && jobs.length === 0 && (
            <EmptyState title="No jobs yet" description="Upload a curriculum document to start." />
          )}
          <div className="space-y-2">
            {jobs.map((job) => {
              const isSelected = selected?.jobId === job.jobId
              return (
                <div
                  key={job.jobId}
                  onClick={() => setSelected(job)}
                  className={`cursor-pointer rounded-xl border p-4 transition-all ${
                    isSelected
                      ? 'border-teal-600 bg-teal-50/50 shadow-sm'
                      : 'border-slate-200 bg-white hover:border-slate-300'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-mono text-xs font-bold text-teal-700 truncate max-w-[70%]">{job.subjectCode}</span>
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-bold capitalize ${STATUS_COLOR[job.status] ?? 'bg-slate-100 text-slate-600'}`}>
                      {job.status}
                    </span>
                  </div>
                  <p className="mt-1 font-bold text-xs text-slate-900">
                    {job.subjectCode} · Grade {job.gradeLevel}
                  </p>
                  <div className="mt-2 flex items-center justify-between text-[11px] text-slate-500">
                    <span>{job.academicYear}</span>
                    <span>{new Date(job.createdAt).toLocaleDateString()}</span>
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        {/* Detail Panel */}
        <div className="space-y-4 lg:col-span-2">
          {selected ? (
            <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
              <div className="flex items-start justify-between border-b border-slate-100 pb-4">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-xs font-bold text-teal-700">{selected.subjectCode}</span>
                    <span className="text-xs text-slate-400">
                      · {new Date(selected.createdAt).toLocaleDateString()}
                    </span>
                  </div>
                  <h2 className="mt-1 text-lg font-bold text-slate-900">
                    {selected.subjectCode} — Grade {selected.gradeLevel}, {selected.academicYear}
                  </h2>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <Link
                    to="/curriculum/jobs/$jobId"
                    params={{ jobId: selected.jobId }}
                    className="rounded-lg bg-teal-700 px-3 py-1.5 text-xs font-semibold text-white hover:bg-teal-800 flex items-center gap-1"
                  >
                    <ChevronRight className="h-3.5 w-3.5" />
                    Open Review
                  </Link>
                </div>
              </div>

              <div className="mt-4 grid grid-cols-3 gap-3">
                <div className="rounded-lg bg-slate-50 p-3 text-xs">
                  <span className="text-slate-500">Status</span>
                  <p className={`mt-0.5 text-base font-bold font-mono capitalize ${
                    selected.status === 'approved' ? 'text-emerald-600' :
                    selected.status === 'failed' || selected.status === 'rejected' ? 'text-rose-600' :
                    'text-amber-600'
                  }`}>{selected.status}</p>
                </div>
                <div className="rounded-lg bg-slate-50 p-3 text-xs">
                  <span className="text-slate-500">Grade Level</span>
                  <p className="mt-0.5 text-base font-bold text-slate-900 font-mono">{selected.gradeLevel}</p>
                </div>
                <div className="rounded-lg bg-slate-50 p-3 text-xs">
                  <span className="text-slate-500">Academic Year</span>
                  <p className="mt-0.5 text-base font-bold text-slate-900">{selected.academicYear}</p>
                </div>
              </div>

              <div className="mt-5 pt-4 border-t border-slate-100 space-y-2">
                <p className="text-xs text-slate-500">
                  {selected.status === 'parsed' || selected.status === 'review'
                    ? 'This job has been parsed and is ready for human review. Open the review page to inspect the extracted topic tree, edit if needed, and approve.'
                    : selected.status === 'approved'
                    ? 'This job has been approved and its topics/CLOs are live in the curriculum graph.'
                    : selected.status === 'pending' || selected.status === 'parsing'
                    ? 'This job is currently being processed by the AI extraction pipeline. Refresh to check status.'
                    : 'This job could not be processed. Try re-uploading the document.'}
                </p>
                {(selected.status === 'parsed' || selected.status === 'review') && (
                  <Link
                    to="/curriculum/jobs/$jobId"
                    params={{ jobId: selected.jobId }}
                    className="mt-2 inline-flex items-center gap-1 rounded-lg bg-slate-900 px-4 py-2 text-xs font-semibold text-white hover:bg-slate-800"
                  >
                    <ChevronRight className="h-3.5 w-3.5" />
                    Review & Approve Curriculum Job
                  </Link>
                )}
                {(selected.status === 'pending' || selected.status === 'parsing') && (
                  <button
                    type="button"
                    onClick={() => void qc.invalidateQueries({ queryKey: queryKeys.curriculumJobs(page) })}
                    className="mt-2 inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50"
                  >
                    <RefreshCw className="h-3.5 w-3.5" />
                    Refresh Status
                  </button>
                )}
              </div>
            </div>
          ) : (
            <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-sm flex flex-col items-center justify-center text-center text-slate-400 min-h-[300px]">
              <ChevronRight className="h-10 w-10 mb-3 stroke-[1.5]" />
              <p className="text-sm font-medium">Select a job from the queue to review details.</p>
              <p className="text-xs mt-1">All uploaded curriculum documents appear in the left panel once processing begins.</p>
            </div>
          )}
        </div>
      </div>
    </AppShell>
  )
}
