import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Award,
  BookOpen,
  CheckCircle2,
  ClipboardList,
} from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  DistributionDonutChart,
  ManagementTableCard,
  PerformanceAreaChart,
  ScheduleCalendarWidget,
  StatMetricCard,
} from '@components/dashboard'
import { StatusPill } from '@components/ui'
import {
  generateStudyPlan,
  getAvailableExams,
  getMySubjectProfiles,
} from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'

interface ExamRowItem {
  id: string
  name: string
  admitDate: string
  type: string
  status: 'Mastered' | 'In Progress' | 'Needs Review' | 'Active'
}

export function StudentDashboardPage() {
  const user = useAuthStore((s) => s.user)
  const firstName = user?.full_name.split(' ')[0] ?? 'Abebe'
  const queryClient = useQueryClient()

  const { data: subjectProfiles } = useQuery({
    queryKey: queryKeys.mySubjectProfiles(),
    queryFn: getMySubjectProfiles,
  })

  const { data: availableExams } = useQuery({
    queryKey: queryKeys.availableExams(),
    queryFn: getAvailableExams,
  })

  const performanceData = subjectProfiles?.map((sp) => ({
    label: sp.subjectName.slice(0, 4),
    value: Math.round(sp.currentMasteryPct),
  })) ?? []

  const masteredCount = subjectProfiles?.filter((sp) => sp.currentMasteryPct >= 75).length ?? 0
  const inProgressCount = subjectProfiles?.filter((sp) => sp.currentMasteryPct >= 40 && sp.currentMasteryPct < 75).length ?? 0
  const needsReviewCount = subjectProfiles?.filter((sp) => sp.currentMasteryPct < 40).length ?? 0
  const totalSubjects = (masteredCount + inProgressCount + needsReviewCount) || 1
  const donutSegments = [
    { name: 'Mastered', value: Math.round((masteredCount / totalSubjects) * 100), color: '#2d2d2e' },
    { name: 'In Progress', value: Math.round((inProgressCount / totalSubjects) * 100), color: '#6b7280' },
    { name: 'Needs Review', value: Math.round((needsReviewCount / totalSubjects) * 100), color: '#e5e7eb' },
  ]

  const avgMastery = subjectProfiles?.length
    ? Math.round(subjectProfiles.reduce((s, p) => s + p.currentMasteryPct, 0) / subjectProfiles.length)
    : 0

  const scheduleItems = (availableExams?.exams ?? [])
    .filter((e) => !e.alreadyAttempted)
    .slice(0, 3)
    .map((e) => ({
      id: e.examId,
      title: e.title,
      time: e.closesAt ? `Closes ${new Date(e.closesAt).toLocaleDateString()}` : 'Open',
      subtitle: `${e.subjectCode} · Grade ${e.gradeLevel}`,
      category: 'upcoming' as const,
    }))

  const generatePlan = useMutation({
    mutationFn: () => generateStudyPlan({}),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.myStudyPlans() })
    },
  })

  const tableData: ExamRowItem[] = subjectProfiles?.map((sp, idx) => ({
    id: `#403${22 + idx}`,
    name: sp.subjectName,
    admitDate: `Grade ${sp.gradeLevel}`,
    type: `${sp.examsAnalyzed} Exams`,
    status: sp.currentMasteryPct >= 75 ? 'Mastered' : sp.currentMasteryPct >= 50 ? 'In Progress' : 'Needs Review',
  })) ?? [
    { id: '#40322', name: 'Physics Grade 11', admitDate: '9/4/26', type: 'Kinematics', status: 'Mastered' },
    { id: '#40323', name: 'Chemistry Grade 11', admitDate: '4/4/26', type: 'Organic', status: 'In Progress' },
    { id: '#40324', name: 'Mathematics Grade 11', admitDate: '1/28/26', type: 'Calculus', status: 'Mastered' },
    { id: '#40325', name: 'Biology Grade 11', admitDate: '1/31/26', type: 'Genetics', status: 'Needs Review' },
  ]

  const tableColumns = [
    {
      key: 'id',
      header: 'ID',
      sortable: true,
      render: (item: ExamRowItem) => (
        <span className="font-mono font-medium text-gray-500">{item.id}</span>
      ),
    },
    {
      key: 'name',
      header: 'Subject & Unit',
      sortable: true,
      render: (item: ExamRowItem) => (
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-gray-100 font-bold text-gray-800 text-xs">
            {item.name.charAt(0)}
          </div>
          <span className="font-semibold text-gray-900">{item.name}</span>
        </div>
      ),
    },
    {
      key: 'admitDate',
      header: 'Enrolled / Grade',
      sortable: true,
      render: (item: ExamRowItem) => <span className="text-gray-600">{item.admitDate}</span>,
    },
    {
      key: 'type',
      header: 'Category',
      render: (item: ExamRowItem) => <span className="text-gray-600 font-medium">{item.type}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      render: (item: ExamRowItem) => {
        const tone =
          item.status === 'Mastered'
            ? 'health'
            : item.status === 'In Progress'
              ? 'seal'
              : 'alert'
        return <StatusPill tone={tone}>{item.status}</StatusPill>
      },
    },
    {
      key: 'details',
      header: 'Details',
      render: () => (
        <button
          type="button"
          className="rounded-lg border border-gray-200 bg-white px-2.5 py-1 text-[11px] font-semibold text-gray-700 hover:bg-gray-50"
        >
          Details
        </button>
      ),
    },
  ]

  return (
    <AppShell
      title={`Welcome back, ${firstName} 👋`}
      description="Welcome to your personalized EduGraph AI learning management dashboard."
    >
      <div className="space-y-6">
        {/* Row 1: Top Stat Cards (4 Grid Layout matching Inspo) */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatMetricCard
            title="Total Subjects"
            value={`${subjectProfiles?.length ?? 0}`}
            change=""
            trend="up"
            periodText="Tracked by AI"
            icon={BookOpen}
          />
          <StatMetricCard
            title="Exams Analyzed"
            value={`${subjectProfiles?.reduce((s, p) => s + p.examsAnalyzed, 0) ?? 0}`}
            change=""
            trend="up"
            periodText="Total submissions"
            icon={ClipboardList}
          />
          <StatMetricCard
            title="Available Exams"
            value={`${availableExams?.exams.filter((e) => !e.alreadyAttempted).length ?? 0}`}
            change=""
            trend="up"
            periodText="Ready to take"
            icon={CheckCircle2}
          />
          <StatMetricCard
            title="Average Mastery"
            value={`${avgMastery}%`}
            change=""
            trend={avgMastery >= 60 ? 'up' : 'down'}
            periodText="Across all subjects"
            icon={Award}
          />
        </div>

        {/* Row 2: Charts & Schedule Grid Layout */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          {/* Left Chart: Performance Area Chart */}
          <div className="lg:col-span-5">
            <PerformanceAreaChart
              title="Subject Mastery Overview"
              subtitle="Current mastery percentage per subject"
              data={performanceData}
            />
          </div>

          {/* Middle Donut Chart: Distribution Donut */}
          <div className="lg:col-span-4">
            <DistributionDonutChart
              title="Mastery & Gap Overview"
              centerPercentage={`${avgMastery}%`}
              centerLabel="Avg Mastery"
              totalValue={`${avgMastery}% Avg`}
              segments={donutSegments}
              dateLabel={new Date().toLocaleDateString('en-GB')}
            />
          </div>

          {/* Right Schedule & Events Widget */}
          <div className="lg:col-span-3">
            <ScheduleCalendarWidget
              monthLabel={new Date().toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
              scheduleItems={scheduleItems}
              onAddNew={() => generatePlan.mutate()}
            />
          </div>
        </div>

        {/* Row 3: Management Data Table */}
        <div className="grid grid-cols-1 gap-6">
          <ManagementTableCard
            title="Subjects & Assessment Management"
            searchPlaceholder="Search subject name, ID..."
            columns={tableColumns}
            data={tableData}
          />
        </div>
      </div>
    </AppShell>
  )
}
