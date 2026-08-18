import { useState } from 'react'
import { Download, FileText, RefreshCw } from 'lucide-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { generateReport, listReports } from '@lib/api/endpoints'
import type { ReportType } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

const REPORT_TEMPLATES: { type: ReportType; title: string; description: string }[] = [
  {
    type: 'clo_coverage',
    title: 'CLO Coverage Analysis',
    description: 'Comprehensive CLO coverage by topic, showing which learning outcomes are assessed and which are missing.',
  },
  {
    type: 'school_monthly',
    title: 'Monthly Class Progress Report',
    description: 'Student mastery summaries and exam performance trends for the current academic month.',
  },
  {
    type: 'national_heatmap',
    title: 'National Gap Heatmap',
    description: 'Cross-grade topic struggle heatmap. Ministry-level view of prerequisite gaps across regions.',
  },
]

const STATUS_COLOR: Record<string, string> = {
  pending: 'bg-amber-100 text-amber-800',
  running: 'bg-blue-100 text-blue-800',
  ready: 'bg-emerald-100 text-emerald-800',
  failed: 'bg-rose-100 text-rose-800',
}

export function TeacherReportsPage() {
  const qc = useQueryClient()
  const [generating, setGenerating] = useState<ReportType | null>(null)

  const { data: reports, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.reports(),
    queryFn: listReports,
    refetchInterval: (query) =>
      query.state.data?.some((r) => r.status === 'pending' || r.status === 'running') ? 5000 : false,
  })

  const generateMut = useMutation({
    mutationFn: (type: ReportType) => generateReport({ reportType: type }),
    onMutate: (type) => setGenerating(type),
    onSettled: () => {
      setGenerating(null)
      void qc.invalidateQueries({ queryKey: queryKeys.reports() })
    },
  })

  return (
    <AppShell
      title="Academic Reports Exporter"
      description="Generate class mastery summaries, CLO coverage reports, and psychometric exam analytics."
    >
      <div className="space-y-6">
        {/* Generate panel */}
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          {REPORT_TEMPLATES.map((tpl) => (
            <div key={tpl.type} className="flex flex-col justify-between rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
              <div>
                <div className="flex items-center gap-2 mb-2">
                  <FileText className="h-4 w-4 text-teal-600" />
                  <span className="font-mono text-xs font-bold text-teal-700 uppercase">{tpl.type.replace('_', ' ')}</span>
                </div>
                <h3 className="text-sm font-bold text-slate-900">{tpl.title}</h3>
                <p className="mt-1 text-xs text-slate-600 leading-relaxed">{tpl.description}</p>
              </div>
              <div className="mt-4 pt-3 border-t border-slate-100">
                <button
                  type="button"
                  disabled={generating === tpl.type || generateMut.isPending}
                  onClick={() => generateMut.mutate(tpl.type)}
                  className="flex w-full items-center justify-center gap-1.5 rounded-lg bg-teal-700 px-3 py-1.5 text-xs font-semibold text-white hover:bg-teal-800 disabled:opacity-50"
                >
                  {generating === tpl.type ? (
                    <><Spinner className="h-3.5 w-3.5" /> Generating…</>
                  ) : (
                    <><Download className="h-3.5 w-3.5" /> Generate Report</>
                  )}
                </button>
              </div>
            </div>
          ))}
        </div>

        {/* History */}
        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="font-bold text-sm text-slate-900">Report History</h3>
            <button
              type="button"
              onClick={() => void qc.invalidateQueries({ queryKey: queryKeys.reports() })}
              className="text-slate-400 hover:text-slate-600"
            >
              <RefreshCw className="h-4 w-4" />
            </button>
          </div>

          {isLoading && (
            <div className="flex items-center gap-2 py-6 text-xs text-slate-500">
              <Spinner /> Loading reports…
            </div>
          )}
          {isError && <Banner tone="error">{(error as Error).message}</Banner>}
          {!isLoading && (!reports || reports.length === 0) && (
            <EmptyState title="No reports yet" description="Generate a report above to see it here." />
          )}
          {reports && reports.length > 0 && (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="border-b border-slate-100 bg-slate-50 text-[10px] uppercase font-bold text-slate-400">
                  <tr>
                    <th className="p-3">Type</th>
                    <th className="p-3">Status</th>
                    <th className="p-3">Requested</th>
                    <th className="p-3">Generated</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 font-medium text-slate-700">
                  {reports.map((r) => (
                    <tr key={r.id} className="hover:bg-slate-50">
                      <td className="p-3 font-mono text-teal-700">{r.reportType.replace('_', ' ')}</td>
                      <td className="p-3">
                        <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-bold uppercase ${STATUS_COLOR[r.status] ?? 'bg-slate-100 text-slate-600'}`}>
                          {r.status}
                        </span>
                      </td>
                      <td className="p-3 text-slate-400">{new Date(r.createdAt).toLocaleDateString()}</td>
                      <td className="p-3 text-slate-400">{r.generatedAt ? new Date(r.generatedAt).toLocaleDateString() : '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </AppShell>
  )
}
