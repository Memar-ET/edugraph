import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Award, BookOpen, Send, ShieldCheck, Users } from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  DistributionDonutChart,
  PerformanceAreaChart,
  ScheduleCalendarWidget,
  StatMetricCard,
} from '@components/dashboard'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, Input, Label, Select, Spinner } from '@components/ui'
import { QualityScoreGrid } from '@components/shared'
import { apiErrorMessage } from '@lib/api/client'
import { createNotification, getSchool, getSchoolQualityScores, listStudents, listTeachers } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'

const MOCK_SCHOOL_PERFORMANCE = [
  { label: 'Jan', value: 72 },
  { label: 'Feb', value: 78 },
  { label: 'Mar', value: 81 },
  { label: 'Apr', value: 85 },
  { label: 'May', value: 88 },
  { label: 'June', value: 92 },
]

const MOCK_SCHOOL_DONUT = [
  { name: 'CLO Coverage', value: 84, color: '#2d2d2e' },
  { name: 'Student Mastery', value: 76, color: '#6b7280' },
  { name: 'Exam Quality', value: 88, color: '#e5e7eb' },
]

const MOCK_SCHOOL_SCHEDULE = [
  {
    id: 'sc1',
    title: 'School Quality Score Audit',
    time: '10:00 AM - 12:00 PM',
    subtitle: 'Redis Cached TTL: 1h',
    category: 'schedule' as const,
  },
  {
    id: 'sc2',
    title: 'Teacher Staff Meeting',
    time: '03:00 PM (Today)',
    subtitle: 'Roster: 24 Staff Members',
    category: 'upcoming' as const,
  },
]

export function SchoolAdminDashboardPage() {
  const user = useAuthStore((s) => s.user)
  const schoolId = user?.school_id

  const { data: school } = useQuery({
    queryKey: queryKeys.school(schoolId ?? 'unknown'),
    queryFn: () => getSchool(schoolId as string),
    enabled: Boolean(schoolId),
  })

  if (!schoolId) {
    return (
      <AppShell title="School Admin Dashboard 👋">
        <Banner tone="error">Your account isn&apos;t linked to a school yet. Contact your regional admin.</Banner>
      </AppShell>
    )
  }

  return (
    <AppShell title={school?.name ?? 'School Admin Dashboard 👋'} description="Quality scores, roster breakdown, and notification dispatch.">
      <div className="space-y-6">
        {/* Top Metric Cards */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatMetricCard
            title="Total Teachers"
            value="24 Staff"
            change="4.2%"
            trend="up"
            periodText="Active Roster"
            icon={Users}
          />
          <StatMetricCard
            title="Enrolled Students"
            value="480 Students"
            change="12.8%"
            trend="up"
            periodText="Academic Year"
            icon={BookOpen}
          />
          <StatMetricCard
            title="School Quality Score"
            value="84.2%"
            change="6.4%"
            trend="up"
            periodText="Composite Score"
            icon={ShieldCheck}
          />
          <StatMetricCard
            title="CLO Standard Mastery"
            value="88.0%"
            change="8.2%"
            trend="up"
            periodText="Curriculum Alignment"
            icon={Award}
          />
        </div>

        {/* Charts & Schedule Asymmetric Grid */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          <div className="lg:col-span-5">
            <PerformanceAreaChart
              title="School-Wide Quality Trend"
              subtitle="Monthly Composite Score Progress"
              data={MOCK_SCHOOL_PERFORMANCE}
            />
          </div>

          <div className="lg:col-span-4">
            <DistributionDonutChart
              title="Quality Index Breakdown"
              centerPercentage="84%"
              centerLabel="Quality"
              totalValue="84.2 Composite"
              segments={MOCK_SCHOOL_DONUT}
              dateLabel="Cached 1h"
            />
          </div>

          <div className="lg:col-span-3">
            <ScheduleCalendarWidget
              monthLabel="April 2026"
              scheduleItems={MOCK_SCHOOL_SCHEDULE}
            />
          </div>
        </div>

        {/* Quality Scores Section */}
        <QualityScoresSection schoolId={schoolId} />

        <div className="grid gap-6 lg:grid-cols-2">
          <RosterSection schoolId={schoolId} />
          <NotifySection schoolId={schoolId} />
        </div>
      </div>
    </AppShell>
  )
}

function QualityScoresSection({ schoolId }: { schoolId: string }) {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.schoolQuality(schoolId),
    queryFn: () => getSchoolQualityScores(schoolId),
  })

  return (
    <Card className="rounded-2xl border-gray-100 shadow-sm p-2">
      <CardHeader>
        <CardTitle className="font-display text-lg font-bold">Subject & Grade Quality Score Breakdown</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div className="flex items-center gap-2 py-6 text-xs text-gray-500">
            <Spinner /> Loading quality scores...
          </div>
        )}
        {isError && <Banner tone="error">{apiErrorMessage(error, 'Could not load quality scores.')}</Banner>}
        {data && data.scores.length === 0 && (
          <EmptyState
            icon={ShieldCheck}
            title="No quality scores yet"
            description="Scores compute automatically once exams have been published and graded."
          />
        )}
        {data && data.scores.length > 0 && <QualityScoreGrid scores={data.scores} />}
      </CardContent>
    </Card>
  )
}

function RosterSection({ schoolId }: { schoolId: string }) {
  const { data: teachers, isLoading: loadingTeachers } = useQuery({
    queryKey: queryKeys.teachers(schoolId),
    queryFn: () => listTeachers(schoolId),
  })
  const { data: students, isLoading: loadingStudents } = useQuery({
    queryKey: queryKeys.students(schoolId),
    queryFn: () => listStudents(schoolId),
  })

  return (
    <Card className="rounded-2xl border-gray-100 shadow-sm">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="font-display text-base font-bold">Institutional Roster</CardTitle>
        <Users className="h-4 w-4 text-gray-400" />
      </CardHeader>
      <CardContent>
        {(loadingTeachers || loadingStudents) && (
          <div className="flex items-center gap-2 py-4 text-xs text-gray-500">
            <Spinner /> Loading roster...
          </div>
        )}
        {!loadingTeachers && !loadingStudents && (
          <div className="grid grid-cols-2 gap-4 text-center">
            <div className="rounded-xl border border-gray-100 bg-gray-50/60 p-4">
              <p className="font-display text-2xl font-bold text-gray-900">{teachers?.length ?? 24}</p>
              <p className="text-xs text-gray-500 font-medium mt-1">Teachers</p>
            </div>
            <div className="rounded-xl border border-gray-100 bg-gray-50/60 p-4">
              <p className="font-display text-2xl font-bold text-gray-900">{students?.length ?? 480}</p>
              <p className="text-xs text-gray-500 font-medium mt-1">Students</p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function NotifySection({ schoolId }: { schoolId: string }) {
  const queryClient = useQueryClient()
  const { data: teachers } = useQuery({
    queryKey: queryKeys.teachers(schoolId),
    queryFn: () => listTeachers(schoolId),
  })
  const { data: students } = useQuery({
    queryKey: queryKeys.students(schoolId),
    queryFn: () => listStudents(schoolId),
  })

  const [userId, setUserId] = useState('')
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')

  const send = useMutation({
    mutationFn: createNotification,
    onSuccess: () => {
      setTitle('')
      setBody('')
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications() })
    },
  })

  const options = [
    ...(teachers ?? []).map((t) => ({ value: t.user_id, label: `Teacher · ${t.subject_specialty || t.id.slice(0, 8)}` })),
    ...(students ?? []).map((s) => ({ value: s.user_id, label: `Student · ${s.admission_no}` })),
  ]

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (!userId || !title.trim() || !body.trim()) return
    send.mutate({ user_id: userId, title: title.trim(), body: body.trim() })
  }

  return (
    <Card className="rounded-2xl border-gray-100 shadow-sm">
      <CardHeader>
        <CardTitle className="font-display text-base font-bold">Dispatch System Notification</CardTitle>
      </CardHeader>
      <CardContent>
        <form className="space-y-3" onSubmit={handleSubmit}>
          {send.isSuccess && <Banner tone="success">Notification sent successfully.</Banner>}
          {send.isError && <Banner tone="error">{apiErrorMessage(send.error, 'Could not send notification.')}</Banner>}
          <div>
            <Label htmlFor="recipient" className="text-xs font-semibold">Recipient</Label>
            <Select id="recipient" value={userId} onChange={(e) => setUserId(e.target.value)} className="text-xs">
              <option value="">Select recipient...</option>
              {options.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </Select>
          </div>
          <div>
            <Label htmlFor="notif-title" className="text-xs font-semibold">Notification Title</Label>
            <Input id="notif-title" value={title} onChange={(e) => setTitle(e.target.value)} className="text-xs" />
          </div>
          <div>
            <Label htmlFor="notif-body" className="text-xs font-semibold">Notification Message</Label>
            <textarea
              id="notif-body"
              value={body}
              onChange={(e) => setBody(e.target.value)}
              rows={2}
              className="flex w-full rounded-xl border border-gray-200 bg-gray-50/70 p-2.5 text-xs text-gray-900 focus:border-gray-900 focus:bg-white focus:outline-none"
            />
          </div>
          <Button type="submit" isLoading={send.isPending} className="w-full bg-gray-900 hover:bg-gray-800 text-white rounded-xl">
            <Send className="h-4 w-4 mr-2" />
            Send Notification
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
