import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Download, FileText, RefreshCw } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner, useToast } from '@components/ui'
import { generateReport, listReports } from '@lib/api/endpoints'
import type { ReportType } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

const EXPORT_TYPES: { type: ReportType; label: string; description: string }[] = [
  {
    type: 'national_heatmap',
    label: 'National CLO Mastery Heatmap',
    description: 'Anonymized mastery data across all CLO standards, all regions.',
  },
  {
    type: 'school_monthly',
    label: 'School Monthly Performance',
    description: 'Per-school quality scores and assessment completion rates.',
  },
  {
    type: 'clo_coverage',
    label: 'CLO Coverage Report',
    description: 'Curriculum standard coverage rates across active subjects.',
  },
]

export function MinistryDataExportsPage() {
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
      toast({ title: 'Export generation started', variant: 'success' })
      void qc.invalidateQueries({ queryKey: queryKeys.reports() })
    },
    onError: () => {
      toast({ title: 'Failed to start export', variant: 'error' })
    },
    onSettled: () => setGenerating(null),
  })

  const reports = data ?? []

  return (
    <AppShell
      title="National Educational Data Exporter"
      description="Generate anonymized national dataset summaries for policy researchers and regional bureaus."
    >
      <div className="space-y-6">
        {isError && (
          <Banner variant="error" title="Failed to load exports" description="Try refreshing the page." />
        )}

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {EXPORT_TYPES.map((et) => (
            <div key={et.type} className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm space-y-3">
              <h3 className="text-sm font-bold text-slate-900">{et.label}</h3>
              <p className="text-xs text-slate-500">{et.description}</p>
              <button
                type="button"
                disabled={genMutation.isPending}
                onClick={() => genMutation.mutate(et.type)}
                className="flex items-center gap-1.5 rounded-lg bg-teal-700 px-3 py-1.5 text-xs font-semibold text-white hover:bg-teal-800 disabled:opacity-50"
              >
                {generating === et.type ? (
                  <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Download className="h-3.5 w-3.5" />
                )}
                Generate Export
              </button>
            </div>
          ))}
        </div>

        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm space-y-4">
          <h3 className="text-sm font-bold text-slate-900">Export History</h3>
          {isLoading ? (
            <div className="flex justify-center py-8"><Spinner /></div>
          ) : reports.length === 0 ? (
            <EmptyState
              icon={FileText}
              title="No exports yet"
              description="Generate your first export above."
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
