import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FileText, Plus, RefreshCw } from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  Banner,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  EmptyState,
  Spinner,
  StatusPill,
  useToast,
} from '@components/ui'
import { generateReport, listReports } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function SchoolReportsPage() {
  const qc = useQueryClient()
  const { toast } = useToast()
  const [generating, setGenerating] = useState(false)

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.reports(),
    queryFn: () => listReports(),
  })

  const genMutation = useMutation({
    mutationFn: () =>
      generateReport({
        reportType: 'school_monthly',
        params: {},
      }),
    onMutate: () => setGenerating(true),
    onSuccess: () => {
      toast({ title: 'Report generation started', variant: 'success' })
      void qc.invalidateQueries({ queryKey: queryKeys.reports() })
    },
    onError: () => {
      toast({ title: 'Failed to start report generation', variant: 'error' })
    },
    onSettled: () => setGenerating(false),
  })

  const reports = data ?? []

  return (
    <AppShell>
      <div className="space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">School Reports</h1>
            <p className="mt-1 text-sm text-gray-500">
              Generate and download summary reports for your school.
            </p>
          </div>
          <button
            onClick={() => genMutation.mutate()}
            disabled={generating}
            className="flex items-center gap-2 rounded-lg bg-primary-700 px-4 py-2 text-sm font-medium text-white hover:bg-primary-600 focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:opacity-50"
          >
            {generating ? (
              <RefreshCw className="h-4 w-4 animate-spin" aria-hidden />
            ) : (
              <Plus className="h-4 w-4" aria-hidden />
            )}
            Generate report
          </button>
        </div>

        {isError && <Banner variant="error" title="Failed to load reports" description="Try refreshing." />}

        <Card>
          <CardHeader><CardTitle>Report History</CardTitle></CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="flex justify-center py-12"><Spinner /></div>
            ) : reports.length === 0 ? (
              <EmptyState
                title="No reports yet"
                description="Generate your first school summary report above."
              />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-100 bg-gray-50 text-left text-xs font-medium text-gray-500">
                      <th className="px-4 py-3">Report</th>
                      <th className="px-4 py-3">Status</th>
                      <th className="px-4 py-3">Requested</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-50">
                    {reports.map((report) => (
                      <tr key={report.id} className="hover:bg-gray-50">
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <FileText className="h-4 w-4 text-gray-400" aria-hidden />
                            <span className="font-medium capitalize text-gray-900">
                              {report.reportType.replace(/_/g, ' ')}
                            </span>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <StatusPill value={report.status} />
                        </td>
                        <td className="px-4 py-3 text-gray-500">
                          {new Date(report.createdAt).toLocaleDateString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </AppShell>
  )
}
