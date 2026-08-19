import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FileText, RefreshCw } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner, useToast } from '@components/ui'
import { generateReport, listReports } from '@lib/api/endpoints'
import type { ReportType } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

const REPORT_TYPES: { type: ReportType; label: string; description: string }[] = [
  {
    type: 'school_monthly',
    label: 'Zonal Monthly Report',
    description: 'School quality scores and assessment completion across your region.',
  },
  {
    type: 'clo_coverage',
    label: 'CLO Coverage Report',
    description: 'Curriculum standard coverage rates for schools in your region.',
  },
]

export function RegionalReportsPage() {
  const qc = useQueryClient()
  const { toast } = useToast()
  const [generating, setGenerating] = useState<ReportType | null>(null)

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.reports(),
    queryFn: listReports,
    refetchInterval: (q) => {
      const reports = q.state.data ?? []
      return reports.some((r) => r.status === 'pending' || r.status === 'running') ? 5000 : false
    },
  })

  const genMutation = useMutation({
    mutationFn: (type: ReportType) => generateReport({ reportType: type }),
    onMutate: (type) => setGenerating(type),
    onSuccess: () => {
      toast({ title: 'Report generation started', variant: 'success' })
      void qc.invalidateQueries({ queryKey: queryKeys.reports() })
    },
    onError: () => {
      toast({ title: 'Failed to start report', variant: 'error' })
    },
    onSettled: () => setGenerating(null),
  })

  const reports = data ?? []

  return (
    <AppShell
      title="Regional Reports Builder"
      description="Generate and download regional academic performance summaries and district quality comparisons."
    >
      <div className="space-y-6">
        {isError && (
          <Banner variant="error" title="Failed to load reports" description="Try refreshing." />
        )}

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {REPORT_TYPES.map((rt) => (
            <div key={rt.type} className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm space-y-3">
              <h3 className="text-sm font-bold text-slate-900">{rt.label}</h3>
              <p className="text-xs text-slate-500">{rt.description}</p>
              <div className="pt-2 border-t border-slate-100 flex justify-end">
                <button
                  type="button"
                  disabled={genMutation.isPending}
                  onClick={() => genMutation.mutate(rt.type)}
                  className="flex items-center gap-1.5 rounded-lg bg-teal-700 px-3 py-1 text-xs font-semibold text-white hover:bg-teal-800 disabled:opacity-50"
                >
                  {generating === rt.type ? (
                    <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <FileText className="h-3.5 w-3.5" />
                  )}
                  Generate Report
                </button>
              </div>
            </div>
          ))}
        </div>

        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm space-y-4">
          <h3 className="text-sm font-bold text-slate-900">Report History</h3>
          {isLoading ? (
            <div className="flex justify-center py-8"><Spinner /></div>
          ) : reports.length === 0 ? (
            <EmptyState
              icon={FileText}
              title="No reports yet"
              description="Generated reports will appear here."
            />
          ) : (
            <div className="space-y-2">
              {reports.map((r) => (
                <div key={r.id} className="flex items-center justify-between rounded-lg border border-slate-100 px-4 py-3 text-xs">
                  <span className="font-medium text-slate-900 capitalize">{r.reportType.replace(/_/g, ' ')}</span>
                  <div className="flex items-center gap-4">
                    <span className="font-mono text-slate-400">{new Date(r.createdAt).toLocaleDateString()}</span>
                    <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-bold ${
                      r.status === 'ready' ? 'bg-emerald-100 text-emerald-800'
                      : r.status === 'failed' ? 'bg-rose-100 text-rose-800'
                      : 'bg-amber-100 text-amber-800'
                    }`}>
                      {r.status}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </AppShell>
  )
}
