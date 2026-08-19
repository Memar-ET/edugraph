import { useRef, useMemo, useState } from 'react'
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
import { getClassHeatmap, listExams } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { ExamListItem } from '@lib/api/endpoints'

type ExamRow = ExamListItem & { id: string }

export function TeacherDashboardPage() {
  const navigate = useNavigate()
  const [examId, setExamId] = useState('')
  const [subjectCode, setSubjectCode] = useState('BIO')
  const [gradeLevel, setGradeLevel] = useState('9')
  const [scope, setScope] = useState<{ subjectCode: string; gradeLevel: number }>({
    subjectCode: 'BIO',
    gradeLevel: 9,
  })

  const [selectedTopic, setSelectedTopic] = useState<{ id: string; title: string } | null>(null)
  const [studentIdInput, setStudentIdInput] = useState('')
  const studentIdRef = useRef<HTMLInputElement>(null)

  const handleTopicClick = (topicId: string, topicTitle: string) => {
    setSelectedTopic({ id: topicId, title: topicTitle })
    setStudentIdInput('')
    setTimeout(() => studentIdRef.current?.focus(), 50)
  }

  const { data: heatmapData, isLoading: heatmapLoading, isError: heatmapError, error: heatmapErr } = useQuery({
    queryKey: queryKeys.classHeatmap(scope.subjectCode, scope.gradeLevel),
    queryFn: () => getClassHeatmap(scope.subjectCode, scope.gradeLevel),
  })

  const { data: examsData, isLoading: examsLoading } = useQuery({
    queryKey: queryKeys.exams(),
    queryFn: () => listExams(1, 50),
  })

  const exams = useMemo(() => examsData?.items ?? [], [examsData])
  const examRows = useMemo<ExamRow[]>(() => exams.map((e) => ({ ...e, id: e.examId })), [exams])

  // Derived KPIs from real exam data
  const totalExams = exams.length
  const publishedCount = exams.filter((e) => e.status === 'published').length
  const draftCount = exams.filter((e) => e.status === 'draft').length

  // Derived donut from heatmap (when available) or exam counts
  const donutSegments = useMemo(() => {
    if (heatmapData && heatmapData.topics.length > 0) {
      const severe = heatmapData.topics.filter((t) => t.strugglingPct > 50).length
      const moderate = heatmapData.topics.filter((t) => t.strugglingPct > 20 && t.strugglingPct <= 50).length
      const proficient = heatmapData.topics.filter((t) => t.strugglingPct <= 20).length
      return [
        { name: 'Proficient', value: proficient, color: '#2d2d2e' },
        { name: 'Moderate Gap', value: moderate, color: '#6b7280' },
        { name: 'Severe Gap', value: severe, color: '#e5e7eb' },
      ]
    }
    return [
      { name: 'Proficient', value: publishedCount, color: '#2d2d2e' },
      { name: 'Drafts', value: draftCount, color: '#6b7280' },
    ]
  }, [heatmapData, publishedCount, draftCount])

  // Derive performance chart from heatmap topics mastery
  const performanceData = useMemo(() => {
    if (!heatmapData || heatmapData.topics.length === 0) return []
    return heatmapData.topics
      .slice(0, 6)
      .map((t) => ({ label: t.title.slice(0, 8), value: Math.round((1 - t.strugglingPct / 100) * 100) }))
  }, [heatmapData])

  // Schedule from real exams (published = scheduled, draft = upcoming)
  const scheduleItems = useMemo(() => {
    if (exams.length === 0) return []
    return exams.slice(0, 4).map((e) => ({
      id: e.examId,
      title: e.title,
      time: new Date(e.createdAt).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
      subtitle: `${e.subjectCode} · Grade ${e.gradeLevel}`,
      category: (e.status === 'published' ? 'schedule' : 'upcoming') as 'schedule' | 'upcoming',
    }))
  }, [exams])

  // Cross-grade alert count from heatmap
  const alertCount = heatmapData?.alerts?.length ?? 0

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

  const tableColumns = [
    {
      key: 'examId',
      header: 'Exam ID',
      sortable: true,
      render: (item: ExamRow) => (
        <span className="font-mono font-semibold text-gray-500">{item.examId.slice(0, 8)}</span>
      ),
    },
    {
      key: 'title',
      header: 'Exam Title',
      sortable: true,
      render: (item: ExamRow) => (
        <span className="font-bold text-gray-900">{item.title}</span>
      ),
    },
    {
      key: 'createdAt',
      header: 'Date Created',
      sortable: true,
      render: (item: ExamRow) => (
        <span className="text-gray-600">{new Date(item.createdAt).toLocaleDateString()}</span>
      ),
    },
    {
      key: 'subject',
      header: 'Subject & Grade',
      render: (item: ExamRow) => (
        <span className="text-gray-600 font-medium">
          {item.subjectCode} (Grade {item.gradeLevel})
        </span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (item: ExamRow) => {
        const tone =
          item.status === 'published' ? 'health'
          : item.status === 'closed' ? 'seal'
          : 'alert'
        return <StatusPill tone={tone}>{item.status}</StatusPill>
      },
    },
    {
      key: 'action',
      header: 'Action',
      render: (item: ExamRow) => (
        <button
          type="button"
          onClick={() => void navigate({ to: '/teacher/exams/$examId', params: { examId: item.examId } })}
          className="rounded-lg border border-gray-200 bg-white px-2.5 py-1 text-[11px] font-semibold text-gray-700 hover:bg-gray-50"
        >
          Review Exam
        </button>
      ),
    },
  ]

  return (
    <AppShell
      title="Teacher Assessment & Class Dashboard"
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
            title="Total Exams"
            value={examsLoading ? '…' : `${totalExams} Exams`}
            change=""
            trend="up"
            periodText="All time"
            icon={ClipboardList}
          />
          <StatMetricCard
            title="Published Exams"
            value={examsLoading ? '…' : `${publishedCount} Active`}
            change=""
            trend="up"
            periodText="Live this term"
            icon={CheckCircle2}
          />
          <StatMetricCard
            title="Topics Tracked"
            value={heatmapData ? `${heatmapData.topics.length} Topics` : '—'}
            change=""
            trend="up"
            periodText={`${scope.subjectCode} G${scope.gradeLevel}`}
            icon={Users}
          />
          <StatMetricCard
            title="Cross-Grade Alerts"
            value={heatmapData ? `${alertCount} ${alertCount === 1 ? 'Alert' : 'Alerts'}` : '—'}
            change={alertCount > 0 ? 'High Severity' : 'None'}
            trend={alertCount > 0 ? 'down' : 'up'}
            periodText="Prerequisite Gaps"
            icon={AlertTriangle}
          />
        </div>

        {/* Charts & Schedule Asymmetric Grid */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          {/* Performance Area Chart — derived from heatmap mastery */}
          <div className="lg:col-span-5">
            {performanceData.length > 0 ? (
              <PerformanceAreaChart
                title="Topic Mastery by Topic"
                subtitle={`${scope.subjectCode} Grade ${scope.gradeLevel} — top topics`}
                data={performanceData}
              />
            ) : (
              <Card className="rounded-2xl border-gray-100 shadow-sm h-full flex items-center justify-center">
                <CardContent className="text-center py-8">
                  <p className="text-xs text-gray-500">Load a class heatmap below to see mastery chart.</p>
                </CardContent>
              </Card>
            )}
          </div>

          {/* Donut Chart */}
          <div className="lg:col-span-4">
            <DistributionDonutChart
              title="Class Mastery Breakdown"
              centerPercentage={heatmapData ? `${Math.round((1 - heatmapData.topics.reduce((s, t) => s + t.strugglingPct, 0) / Math.max(1, heatmapData.topics.length) / 100) * 100)}%` : '—'}
              centerLabel="Mastered"
              totalValue={heatmapData ? `${heatmapData.topics.length} Topics Tracked` : 'Load heatmap below'}
              segments={donutSegments}
              dateLabel="Active Term"
            />
          </div>

          {/* Schedule Widget from real exams */}
          <div className="lg:col-span-3">
            <ScheduleCalendarWidget
              monthLabel={new Date().toLocaleString('default', { month: 'long', year: 'numeric' })}
              scheduleItems={scheduleItems}
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
                    placeholder="BIO"
                    className="w-20 text-xs"
                  />
                  <Input
                    value={gradeLevel}
                    onChange={(e) => setGradeLevel(e.target.value)}
                    placeholder="9"
                    type="number"
                    className="w-16 text-xs"
                  />
                  <Button size="sm" variant="secondary" onClick={loadHeatmap}>
                    Load
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                {heatmapLoading && (
                  <div className="flex items-center gap-2 py-6 text-xs text-gray-500">
                    <Spinner /> Loading class gap heatmap...
                  </div>
                )}
                {heatmapError && <Banner tone="error">{apiErrorMessage(heatmapErr, 'Could not load heatmap.')}</Banner>}
                {heatmapData && heatmapData.topics.length > 0 ? (
                  <>
                    <HeatmapGrid topics={heatmapData.topics} onTopicClick={handleTopicClick} />
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
                  !heatmapLoading && (
                    <EmptyState title="No gap data for selection" description="Enter a subject code and grade level, then click Load to see class heatmap insights." />
                  )
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
          data={examRows}
        />
      </div>
    </AppShell>
  )
}
