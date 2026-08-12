import { useState } from 'react'
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
  PerformanceAreaChart,
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

const MOCK_CURRICULUM_PERFORMANCE = [
  { label: 'Jan', value: 12 },
  { label: 'Feb', value: 18 },
  { label: 'Mar', value: 24 },
  { label: 'Apr', value: 38 },
  { label: 'May', value: 45 },
  { label: 'June', value: 54 },
]

const MOCK_CURRICULUM_DONUT = [
  { name: 'Approved', value: 72, color: '#2d2d2e' },
  { name: 'Under Review', value: 18, color: '#6b7280' },
  { name: 'Parsing/Pending', value: 10, color: '#e5e7eb' },
]

const MOCK_CURRICULUM_SCHEDULE = [
  {
    id: 'cs1',
    title: 'Grade 11 Physics Review',
    time: '10 AM - 12 PM',
    subtitle: 'Officer: Abebe Demissie',
    category: 'schedule' as const,
  },
  {
    id: 'cs2',
    title: 'Chemistry Syllabus Versioning',
    time: '02 PM (Today)',
    subtitle: 'Supersede Code V2',
    category: 'upcoming' as const,
  },
  {
    id: 'cs3',
    title: 'Math AST Verification',
    time: '11 AM (Tomorrow)',
    subtitle: 'CLO Vector Embeddings',
    category: 'upcoming' as const,
  },
]

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

  const tableData: JobTableRow[] = data?.items.map((job) => ({
    id: job.jobId,
    fileName: job.fileName,
    subjectCode: job.subjectCode,
    gradeLevel: job.gradeLevel,
    academicYear: job.academicYear,
    status: job.status,
    createdAt: job.createdAt,
    approvedAt: job.approvedAt,
  })) ?? [
    {
      id: 'job-101',
      fileName: 'Ethiopian_Physics_Grade11_Syllabus.pdf',
      subjectCode: 'PHY',
      gradeLevel: 11,
      academicYear: '2026',
      status: 'approved',
      createdAt: '2026-08-01T10:00:00Z',
      approvedAt: '2026-08-02T14:30:00Z',
    },
    {
      id: 'job-102',
      fileName: 'Chemistry_Curriculum_Grade11_Draft.docx',
      subjectCode: 'CHEM',
      gradeLevel: 11,
      academicYear: '2026',
      status: 'review',
      createdAt: '2026-08-05T09:15:00Z',
    },
    {
      id: 'job-103',
      fileName: 'Mathematics_Calculus_Spec.pdf',
      subjectCode: 'MATH',
      gradeLevel: 12,
      academicYear: '2026',
      status: 'parsed',
      createdAt: '2026-08-08T11:00:00Z',
    },
  ]

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
            value="8 Jobs"
            change="Requirements"
            trend="down"
            periodText="Action Needed"
            icon={Clock}
          />
          <StatMetricCard
            title="Promoted Subjects"
            value="36 Subjects"
            change="94.2%"
            trend="up"
            periodText="Graph Synced"
            icon={CheckCircle2}
          />
          <StatMetricCard
            title="AI Extraction Rate"
            value="98.6%"
            change="2.4%"
            trend="up"
            periodText="AST Accuracy"
            icon={UploadCloud}
          />
        </div>

        {/* Charts & Schedule Asymmetric Grid */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          <div className="lg:col-span-5">
            <PerformanceAreaChart
              title="Curriculum Processing Volume"
              subtitle="Monthly Parse & Promotion Throughput"
              data={MOCK_CURRICULUM_PERFORMANCE}
            />
          </div>

          <div className="lg:col-span-4">
            <DistributionDonutChart
              title="Pipeline Approval Status"
              centerPercentage="72%"
              centerLabel="Approved"
              totalValue="54 Total Specs"
              segments={MOCK_CURRICULUM_DONUT}
              dateLabel="Active Cycle"
            />
          </div>

          <div className="lg:col-span-3">
            <ScheduleCalendarWidget
              monthLabel="April 2026"
              scheduleItems={MOCK_CURRICULUM_SCHEDULE}
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
