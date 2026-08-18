import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import {
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Clock,
  FileStack,
  FileText,
  UploadCloud,
} from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  DistributionDonutChart,
  ManagementTableCard,
  ScheduleCalendarWidget,
  StatMetricCard,
} from '@components/dashboard'
import { Banner, Button, Spinner, StatusPill, type StatusPillProps } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { listCurriculumJobs } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { JobStatusValue } from '@/types/api'

const PAGE_SIZE = 20

const STATUS_TONE: Record<JobStatusValue, StatusPillProps['tone']> = {
  pending: 'neutral',
  parsing: 'neutral',
  parsed: 'seal',
  review: 'seal',
  approved: 'health',
  rejected: 'alert',
  failed: 'alert',
}

const STATUS_LABEL: Record<JobStatusValue, string> = {
  pending: 'Pending',
  parsing: 'Parsing…',
  parsed: 'Parsed',
  review: 'Needs review',
  approved: 'Approved',
  rejected: 'Rejected',
  failed: 'Failed',
}

function formatDate(iso: string | undefined): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}


interface JobTableRow {
  id: string
  fileName: string
  subjectCode: string
  gradeLevel: number
  academicYear: string
  status: JobStatusValue
  createdAt: string
  approvedAt?: string
}

export function CurriculumDashboardPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.curriculumJobs(page),
    queryFn: () => listCurriculumJobs(page, PAGE_SIZE),
  })

  const totalPages = data ? Math.max(1, Math.ceil(data.meta.total / data.meta.limit)) : 1

  // Derived chart data from real jobs
  const donutSegments = useMemo(() => {
    const items = data?.items ?? []
    const approved = items.filter((j) => j.status === 'approved').length
    const review = items.filter((j) => j.status === 'review' || j.status === 'parsed').length
    const pending = items.filter((j) => j.status === 'pending' || j.status === 'parsing' || j.status === 'failed').length
    return [
      { name: 'Approved', value: approved, color: '#2d2d2e' },
      { name: 'Under Review', value: review, color: '#6b7280' },
      { name: 'Parsing/Pending', value: pending, color: '#e5e7eb' },
    ].filter((s) => s.value > 0)
  }, [data])

  const scheduleItems = useMemo(() => {
    const items = data?.items ?? []
    return items
      .filter((j) => j.status === 'review' || j.status === 'parsed')
      .slice(0, 3)
      .map((j) => ({
        id: j.jobId,
        title: `${j.subjectCode} Grade ${j.gradeLevel} Review`,
        time: new Date(j.createdAt).toLocaleDateString(),
        subtitle: j.fileName ?? j.subjectCode,
        category: 'upcoming' as const,
      }))
  }, [data])

  const approvedCount = useMemo(() => data?.items.filter((j) => j.status === 'approved').length ?? 0, [data])
  const reviewCount = useMemo(() => data?.items.filter((j) => j.status === 'review' || j.status === 'parsed').length ?? 0, [data])

  const tableData: JobTableRow[] = data?.items.map((job) => ({
    id: job.jobId,
    fileName: job.fileName,
    subjectCode: job.subjectCode,
    gradeLevel: job.gradeLevel,
    academicYear: job.academicYear,
    status: job.status,
    createdAt: job.createdAt,
    approvedAt: job.approvedAt,
  })) ?? []

  const tableColumns = [
    {
      key: 'fileName',
      header: 'Curriculum File',
      sortable: true,
      render: (item: JobTableRow) => (
        <div className="flex items-center gap-2 max-w-[260px]">
          <FileText className="h-4 w-4 shrink-0 text-gray-400" />
          <span className="truncate font-semibold text-gray-900" title={item.fileName}>
            {item.fileName}
          </span>
        </div>
      ),
    },
    {
      key: 'subjectCode',
      header: 'Subject & Grade',
      sortable: true,
      render: (item: JobTableRow) => (
        <span className="font-medium text-gray-700">
          {item.subjectCode} · Grade {item.gradeLevel} ({item.academicYear})
        </span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (item: JobTableRow) => (
        <StatusPill tone={STATUS_TONE[item.status]}>{STATUS_LABEL[item.status]}</StatusPill>
      ),
    },
    {
      key: 'createdAt',
      header: 'Submitted',
      sortable: true,
      render: (item: JobTableRow) => <span className="text-gray-500">{formatDate(item.createdAt)}</span>,
    },
    {
      key: 'approvedAt',
      header: 'Approved',
      render: (item: JobTableRow) => <span className="text-gray-500">{formatDate(item.approvedAt)}</span>,
    },
    {
      key: 'actions',
      header: 'Action',
      render: (item: JobTableRow) => (
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => void navigate({ to: '/curriculum/jobs/$jobId', params: { jobId: item.id } })}
            className="rounded-lg border border-gray-200 bg-white px-3 py-1 text-[11px] font-semibold text-gray-700 hover:bg-gray-50"
          >
            {item.status === 'parsed' || item.status === 'review' ? 'Review Tree' : 'View Specs'}
          </button>
          {item.status === 'approved' && (
            <button
              type="button"
              onClick={() =>
                void navigate({ to: '/curriculum/subjects/$code/graph', params: { code: item.subjectCode } })
              }
              className="rounded-lg border border-gray-200 bg-white px-3 py-1 text-[11px] font-semibold text-gray-700 hover:bg-gray-50"
            >
              Graph
            </button>
          )}
        </div>
      ),
    },
  ]

  return (
    <AppShell
      title="Curriculum Specification Dashboard 👋"
      description="Track curriculum upload jobs, review extracted AST structures, and promote specifications."
      actions={
        <button
          type="button"
          onClick={() => void navigate({ to: '/curriculum/upload' })}
          className="inline-flex items-center gap-2 rounded-xl bg-gray-900 px-4 py-2 text-xs font-semibold text-white shadow-sm transition-colors hover:bg-gray-800"
        >
          <UploadCloud className="h-4 w-4" />
          <span>Upload Curriculum</span>
        </button>
      }
    >
      <div className="space-y-6">
        {/* Top Metric Cards */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatMetricCard
            title="Total Upload Jobs"
            value={`${data?.meta.total ?? 54}+`}
            change="14.2%"
            trend="up"
            periodText="Last Month"
            icon={FileStack}
          />
          <StatMetricCard
            title="Pending Review"
            value={reviewCount > 0 ? `${reviewCount} Jobs` : 'None'}
            change={reviewCount > 0 ? 'Action Needed' : 'Up to Date'}
            trend={reviewCount > 0 ? 'down' : 'up'}
            periodText="Awaiting Review"
            icon={Clock}
          />
          <StatMetricCard
            title="Approved This Page"
            value={`${approvedCount} Jobs`}
            change="Graph Synced"
            trend="up"
            periodText="Promoted to Graph"
            icon={CheckCircle2}
          />
          <StatMetricCard
            title="Total Jobs"
            value={data ? `${data.meta.total}` : '—'}
            change="All Time"
            trend="up"
            periodText="Uploaded"
            icon={UploadCloud}
          />
        </div>

        {/* Charts & Schedule Asymmetric Grid */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          <div className="lg:col-span-5">
            <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm h-full">
              <h3 className="font-bold text-sm text-slate-900">Curriculum Pipeline Summary</h3>
              <p className="text-xs text-slate-500 mt-0.5">Current page job breakdown</p>
              <div className="mt-4 grid grid-cols-2 gap-3">
                {(['approved','review','parsed','pending','parsing','failed'] as const).map((s) => {
                  const count = data?.items.filter((j) => j.status === s).length ?? 0
                  return (
                    <div key={s} className="rounded-xl bg-slate-50 p-3 border border-slate-100">
                      <p className="text-xs text-slate-500 capitalize">{STATUS_LABEL[s]}</p>
                      <p className="text-xl font-bold text-slate-900 mt-0.5">{count}</p>
                    </div>
                  )
                })}
              </div>
            </div>
          </div>

          <div className="lg:col-span-4">
            <DistributionDonutChart
              title="Pipeline Approval Status"
              centerPercentage={data ? `${approvedCount}/${data.items.length}` : '—'}
              centerLabel="Approved"
              totalValue={data ? `${data.items.length} This Page` : 'Loading…'}
              segments={donutSegments}
              dateLabel="Active Cycle"
            />
          </div>

          <div className="lg:col-span-3">
            <ScheduleCalendarWidget
              monthLabel={new Date().toLocaleDateString(undefined, { month: 'long', year: 'numeric' })}
              scheduleItems={scheduleItems}
              onAddNew={() => void navigate({ to: '/curriculum/upload' })}
            />
          </div>
        </div>

        {/* Jobs Table Card */}
        {isLoading && (
          <div className="flex items-center justify-center py-12 text-xs text-gray-500">
            <Spinner /> Loading curriculum jobs...
          </div>
        )}
        {isError && <Banner tone="error">{apiErrorMessage(error, 'Could not load curriculum jobs.')}</Banner>}
        {!isLoading && (
          <div className="space-y-4">
            <ManagementTableCard
              title="Curriculum Jobs & Parsing History"
              searchPlaceholder="Search file name, subject code..."
              columns={tableColumns}
              data={tableData}
            />

            {totalPages > 1 && (
              <div className="flex items-center justify-between rounded-2xl border border-gray-100 bg-white p-4 text-xs font-medium text-gray-500 shadow-sm">
                <span>
                  Page {page} of {totalPages} · {data?.meta.total ?? 0} total jobs
                </span>
                <div className="flex items-center gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={page <= 1}
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                  >
                    <ChevronLeft className="h-4 w-4 mr-1" />
                    Previous
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={page >= totalPages}
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  >
                    Next
                    <ChevronRight className="h-4 w-4 ml-1" />
                  </Button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </AppShell>
  )
}
