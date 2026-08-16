import { useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import {
  AlertTriangle,
  ArrowRight,
  CheckCircle2,
  ClipboardList,
  UploadCloud,
  Users,
} from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  DistributionDonutChart,
  ManagementTableCard,
  PerformanceAreaChart,
  ScheduleCalendarWidget,
  StatMetricCard,
} from '@components/dashboard'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, Input, Spinner, StatusPill } from '@components/ui'
import { HeatmapGrid } from '@components/charts'
import { apiErrorMessage } from '@lib/api/client'
import { getClassHeatmap } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

const MOCK_TEACHER_PERFORMANCE = [
  { label: 'Jan', value: 70 },
  { label: 'Feb', value: 76 },
  { label: 'Mar', value: 74 },
  { label: 'Apr', value: 88 },
  { label: 'May', value: 82 },
  { label: 'June', value: 94 },
]

const MOCK_TEACHER_DONUT = [
  { name: 'Proficient', value: 62, color: '#2d2d2e' },
  { name: 'Moderate Gap', value: 26, color: '#6b7280' },
  { name: 'Severe Gap', value: 12, color: '#e5e7eb' },
]

const MOCK_TEACHER_SCHEDULE = [
  {
    id: 'ts1',
    title: 'Physics Grade 11 Midterm',
    time: '09:00 AM - 11:00 AM',
    subtitle: 'Class Room 102',
    category: 'schedule' as const,
  },
  {
    id: 'ts2',
    title: 'Chemistry Lab Quiz Grading',
    time: '01:30 PM (Today)',
    subtitle: '34 Submissions Pending',
    category: 'upcoming' as const,
  },
  {
    id: 'ts3',
    title: 'Prerequisite Review Session',
    time: '11:00 AM (Tomorrow)',
    subtitle: 'Topic: Kinematics',
    category: 'upcoming' as const,
  },
]

interface TeacherExamRow {
  id: string
  name: string
  admitDate: string
  type: string
  status: 'Published' | 'Draft' | 'Graded' | 'Review Required'
}

const MOCK_EXAM_ROWS: TeacherExamRow[] = [
  { id: '#EX-901', name: 'Physics Grade 11 Kinematics', admitDate: '9/4/26', type: 'Physics (Grade 11)', status: 'Published' },
  { id: '#EX-902', name: 'Chemistry Organic Compounds', admitDate: '4/4/26', type: 'Chemistry (Grade 11)', status: 'Graded' },
  { id: '#EX-903', name: 'Mathematics Calculus Final', admitDate: '1/28/26', type: 'Mathematics (Grade 11)', status: 'Review Required' },
  { id: '#EX-904', name: 'Biology Genetics Unit Quiz', admitDate: '1/31/26', type: 'Biology (Grade 11)', status: 'Draft' },
]

export function TeacherDashboardPage() {
  const navigate = useNavigate()
  const [examId, setExamId] = useState('')
  const [subjectCode, setSubjectCode] = useState('PHY')
  const [gradeLevel, setGradeLevel] = useState('11')
  const [scope, setScope] = useState<{ subjectCode: string; gradeLevel: number }>({
    subjectCode: 'PHY',
    gradeLevel: 11,
  })

  // ExplainPage quick-jump: topic selected from heatmap + student ID input
  const [selectedTopic, setSelectedTopic] = useState<{ id: string; title: string } | null>(null)
  const [studentIdInput, setStudentIdInput] = useState('')
  const studentIdRef = useRef<HTMLInputElement>(null)

  const handleTopicClick = (topicId: string, topicTitle: string) => {
    setSelectedTopic({ id: topicId, title: topicTitle })
    setStudentIdInput('')
    setTimeout(() => studentIdRef.current?.focus(), 50)
  }

  const openExplainPage = () => {
    const sid = studentIdInput.trim()
    if (!sid || !selectedTopic) return
    void navigate({
      to: '/students/$studentId/topics/$topicId/explain',
      params: { studentId: sid, topicId: selectedTopic.id },
    })
    setSelectedTopic(null)
    setStudentIdInput('')
  }

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.classHeatmap(scope.subjectCode, scope.gradeLevel),
    queryFn: () => getClassHeatmap(scope.subjectCode, scope.gradeLevel),
  })

  const loadHeatmap = () => {
    const code = subjectCode.trim().toUpperCase()
    const grade = Number(gradeLevel)
    if (!code || !Number.isInteger(grade) || grade < 1 || grade > 12) return
    setScope({ subjectCode: code, gradeLevel: grade })
  }

  const openExam = () => {
    const trimmed = examId.trim()
    if (!trimmed) return
    const id = trimmed.includes('/') ? (trimmed.split('/').filter(Boolean).pop() ?? trimmed) : trimmed
    void navigate({ to: '/teacher/exams/$examId', params: { examId: id } })
  }

  const tableColumns = [
    {
      key: 'id',
      header: 'Exam ID',
      sortable: true,
      render: (item: TeacherExamRow) => (
        <span className="font-mono font-semibold text-gray-500">{item.id}</span>
      ),
    },
    {
      key: 'name',
      header: 'Exam Title & Topic',
      sortable: true,
      render: (item: TeacherExamRow) => (
        <span className="font-bold text-gray-900">{item.name}</span>
      ),
    },
    {
      key: 'admitDate',
      header: 'Date Created',
      sortable: true,
      render: (item: TeacherExamRow) => <span className="text-gray-600">{item.admitDate}</span>,
    },
    {
      key: 'type',
      header: 'Subject & Grade',
      render: (item: TeacherExamRow) => <span className="text-gray-600 font-medium">{item.type}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      render: (item: TeacherExamRow) => {
        const tone =
          item.status === 'Published'
            ? 'health'
            : item.status === 'Graded'
              ? 'seal'
              : item.status === 'Review Required'
                ? 'alert'
                : 'alert'
        return <StatusPill tone={tone}>{item.status}</StatusPill>
      },
    },
    {
      key: 'details',
      header: 'Action',
      render: (item: TeacherExamRow) => (
        <button
          type="button"
          onClick={() => void navigate({ to: '/teacher/exams/$examId', params: { examId: item.id.replace('#', '') } })}
          className="rounded-lg border border-gray-200 bg-white px-2.5 py-1 text-[11px] font-semibold text-gray-700 hover:bg-gray-50"
        >
          Review Exam
        </button>
      ),
    },
  ]

  return (
    <AppShell
      title="Teacher Assessment & Class Dashboard 👋"
      description="Monitor class performance, create & grade exams, and inspect prerequisite gap heatmaps."
      actions={
        <button
          type="button"
          onClick={() => void navigate({ to: '/teacher/exams/upload' })}
          className="inline-flex items-center gap-2 rounded-xl bg-gray-900 px-4 py-2 text-xs font-semibold text-white shadow-sm transition-colors hover:bg-gray-800"
        >
          <UploadCloud className="h-4 w-4" />
          <span>Upload Exam</span>
        </button>
      }
    >
      <div className="space-y-6">
        {/* Top Metric Cards */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatMetricCard
            title="Total Active Classes"
            value="4 Classes"
            change="12.5%"
            trend="up"
            periodText="Grade 9-12"
            icon={Users}
          />
          <StatMetricCard
            title="Exams Created"
            value="14 Exams"
            change="8.4%"
            trend="up"
            periodText="This Term"
            icon={ClipboardList}
          />
          <StatMetricCard
            title="Class Avg Mastery"
            value="78.2%"
            change="14.1%"
            trend="up"
            periodText="Last Month"
            icon={CheckCircle2}
          />
          <StatMetricCard
            title="Cross-Grade Alerts"
            value="3 Topics"
            change="High Severity"
            trend="down"
            periodText="Prerequisite Gaps"
            icon={AlertTriangle}
          />
        </div>

        {/* Charts & Schedule Asymmetric Grid */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          {/* Performance Area Chart */}
          <div className="lg:col-span-5">
            <PerformanceAreaChart
              title="Class Average Performance"
              subtitle="Monthly Assessment Score Trends"
              data={MOCK_TEACHER_PERFORMANCE}
            />
          </div>

          {/* Donut Chart */}
          <div className="lg:col-span-4">
            <DistributionDonutChart
              title="Class Mastery Breakdown"
              centerPercentage="78%"
              centerLabel="Mastered"
              totalValue="142 Enrolled Students"
              segments={MOCK_TEACHER_DONUT}
              dateLabel="Active Term"
            />
          </div>

          {/* Schedule Widget */}
          <div className="lg:col-span-3">
            <ScheduleCalendarWidget
              monthLabel="April 2026"
              scheduleItems={MOCK_TEACHER_SCHEDULE}
              onAddNew={() => void navigate({ to: '/teacher/exams/upload' })}
            />
          </div>
        </div>

        {/* Heatmap & Direct Jump Row */}
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <Card className="rounded-2xl border-gray-100 shadow-sm">
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <div>
                  <CardTitle className="font-display text-lg font-bold text-gray-900">
                    Topics Ranked by Struggle (Class Heatmap)
                  </CardTitle>
                  <p className="text-xs text-gray-500">
                    Share of your class struggling with each topic, traced from exam attempt gap analysis.
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Input
                    value={subjectCode}
                    onChange={(e) => setSubjectCode(e.target.value)}
                    placeholder="PHY"
                    className="w-20 text-xs"
                  />
                  <Input
                    value={gradeLevel}
                    onChange={(e) => setGradeLevel(e.target.value)}
                    placeholder="11"
                    type="number"
                    className="w-16 text-xs"
                  />
                  <Button size="sm" variant="secondary" onClick={loadHeatmap}>
                    Load
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                {isLoading && (
                  <div className="flex items-center gap-2 py-6 text-xs text-gray-500">
                    <Spinner /> Loading class gap heatmap...
                  </div>
                )}
                {isError && <Banner tone="error">{apiErrorMessage(error, 'Could not load heatmap.')}</Banner>}
                {data && data.topics.length > 0 ? (
                  <>
                    <HeatmapGrid topics={data.topics} onTopicClick={handleTopicClick} />
                    {selectedTopic && (
                      <div className="mt-3 rounded-xl border border-indigo-100 bg-indigo-50 p-3 space-y-2">
                        <p className="text-xs font-semibold text-indigo-800">
                          View learner explanation for "{selectedTopic.title}"
                        </p>
                        <p className="text-[11px] text-indigo-600">
                          Enter the student ID to open their EG-GCKT skill explanation.
                        </p>
                        <div className="flex gap-2">
                          <input
                            ref={studentIdRef}
                            value={studentIdInput}
                            onChange={(e) => setStudentIdInput(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && openExplainPage()}
                            placeholder="Student ID (UUID)…"
                            className="flex-1 rounded-lg border border-indigo-200 bg-white px-2.5 py-1.5 text-xs outline-none focus:border-indigo-400"
                          />
                          <Button size="sm" onClick={openExplainPage} disabled={!studentIdInput.trim()}>
                            Open
                          </Button>
                          <Button size="sm" variant="ghost" onClick={() => setSelectedTopic(null)}>
                            ✕
                          </Button>
                        </div>
                      </div>
                    )}
                  </>
                ) : (
                  <EmptyState title="No gap data for selection" description="Publish an exam to generate class heatmap insights." />
                )}
              </CardContent>
            </Card>
          </div>

          {/* Direct Exam Quick Jump */}
          <div className="space-y-4">
            <Card className="rounded-2xl border-gray-100 shadow-sm">
              <CardHeader>
                <CardTitle className="text-base font-bold">Quick Exam Search & Jump</CardTitle>
                <p className="text-xs text-gray-500">Enter exam ID or URL to review, grade, or inspect quality report.</p>
              </CardHeader>
              <CardContent className="space-y-3">
                <Input
                  value={examId}
                  onChange={(e) => setExamId(e.target.value)}
                  placeholder="Paste exam ID or link..."
                  className="text-xs"
                />
                <Button className="w-full" variant="secondary" onClick={openExam}>
                  <span>Open Exam Page</span>
                  <ArrowRight className="h-4 w-4 ml-2" />
                </Button>
              </CardContent>
            </Card>
          </div>
        </div>

        {/* Exam Management Table */}
        <ManagementTableCard
          title="Exams & Assessment Management"
          searchPlaceholder="Search exam title, ID..."
          columns={tableColumns}
          data={MOCK_EXAM_ROWS}
        />
      </div>
    </AppShell>
  )
}
