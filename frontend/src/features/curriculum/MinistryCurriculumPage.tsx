import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { BookOpen, GitBranch, Layers, ListTree } from 'lucide-react'

import { AppShell } from '@components/layout'
import { ManagementTableCard, StatMetricCard } from '@components/dashboard'
import { Banner, Spinner, StatusPill } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { listSubjects } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { SubjectListItem } from '@/types/api'

interface SubjectRow extends SubjectListItem {
  id: string
}

function formatDate(iso: string | undefined): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

// Ministry-wide curriculum browser: every promoted subject across every
// curriculum officer's uploads (contrast with CurriculumDashboardPage,
// which is one officer's own job history). Clicking a row goes to
// MinistryCurriculumDetailPage for the full detail + graph.
export function MinistryCurriculumPage() {
  const navigate = useNavigate()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.subjects(),
    queryFn: listSubjects,
  })

  const rows: SubjectRow[] = (data ?? []).map((s) => ({ ...s, id: s.code }))
  const totalTopics = rows.reduce((sum, s) => sum + s.topicCount + s.subtopicCount, 0)
  const totalClos = rows.reduce((sum, s) => sum + s.cloCount, 0)
  const totalUnits = rows.reduce((sum, s) => sum + s.unitCount, 0)

  const columns = [
    {
      key: 'code',
      header: 'Subject',
      sortable: true,
      render: (item: SubjectRow) => (
        <div className="flex items-center gap-2">
          <BookOpen className="h-4 w-4 shrink-0 text-gray-400" />
          <div>
            <div className="font-semibold text-gray-900">{item.code}</div>
            <div className="text-xs text-gray-500">
              Grade {item.gradeLevel} · {item.academicYear}
            </div>
          </div>
        </div>
      ),
    },
    {
      key: 'version',
      header: 'Version',
      render: (item: SubjectRow) => (
        <StatusPill tone={item.isCurrent ? 'health' : 'neutral'}>
          v{item.version}
          {item.isCurrent ? '' : ' (superseded)'}
        </StatusPill>
      ),
    },
    {
      key: 'scale',
      header: 'Units / Topics / CLOs',
      render: (item: SubjectRow) => (
        <span className="text-gray-700">
          {item.unitCount} units · {item.topicCount + item.subtopicCount} topics · {item.cloCount} CLOs
        </span>
      ),
    },
    {
      key: 'uploadedByName',
      header: 'Uploaded By',
      render: (item: SubjectRow) => <span className="text-gray-500">{item.uploadedByName ?? '—'}</span>,
    },
    {
      key: 'approvedAt',
      header: 'Approved',
      sortable: true,
      render: (item: SubjectRow) => <span className="text-gray-500">{formatDate(item.approvedAt)}</span>,
    },
  ]

  return (
    <AppShell
      title="Curriculum by Subject"
      description="Every promoted curriculum subject system-wide. Select one to see its full structure and knowledge graph."
    >
      <div className="space-y-6">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatMetricCard title="Subjects" value={`${rows.length}`} periodText="System-wide" icon={BookOpen} />
          <StatMetricCard title="Units" value={`${totalUnits}`} periodText="Across all subjects" icon={Layers} />
          <StatMetricCard title="Topics & Subtopics" value={`${totalTopics}`} periodText="Across all subjects" icon={ListTree} />
          <StatMetricCard title="CLOs" value={`${totalClos}`} periodText="Across all subjects" icon={GitBranch} />
        </div>

        {isLoading && (
          <div className="flex items-center justify-center py-12 text-xs text-gray-500">
            <Spinner /> Loading curriculum subjects...
          </div>
        )}
        {isError && <Banner tone="error">{apiErrorMessage(error, 'Could not load curriculum subjects.')}</Banner>}
        {!isLoading && !isError && rows.length === 0 && (
          <Banner tone="info">No curriculum subjects have been promoted yet.</Banner>
        )}
        {!isLoading && rows.length > 0 && (
          <ManagementTableCard
            title="Promoted Curriculum Subjects"
            searchPlaceholder="Search subject code, grade..."
            columns={columns}
            data={rows}
            onRowClick={(item) =>
              void navigate({ to: '/ministry/curriculum/$code', params: { code: item.code } })
            }
          />
        )}
      </div>
    </AppShell>
  )
}
