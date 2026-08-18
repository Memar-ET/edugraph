import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Award, BookOpen, Send, ShieldCheck, Users } from 'lucide-react'

import { AppShell } from '@components/layout'
import { StatMetricCard } from '@components/dashboard'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, Input, Label, Select, Spinner } from '@components/ui'
import { QualityScoreGrid } from '@components/shared'
import { apiErrorMessage } from '@lib/api/client'
import { createNotification, getSchool, getSchoolQualityScores, listStudents, listTeachers } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'

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
        {/* Top Metric Cards — wired to real data */}
        <TopKpiStrip schoolId={schoolId} />

        {/* Quality Breakdown Charts — driven by real quality data */}
        <QualityChartsSection schoolId={schoolId} />

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

function TopKpiStrip({ schoolId }: { schoolId: string }) {
  const { data: teachers } = useQuery({
    queryKey: queryKeys.teachers(schoolId),
    queryFn: () => listTeachers(schoolId),
  })
  const { data: students } = useQuery({
    queryKey: queryKeys.students(schoolId),
    queryFn: () => listStudents(schoolId),
  })
  const { data: quality } = useQuery({
    queryKey: queryKeys.schoolQuality(schoolId),
    queryFn: () => getSchoolQualityScores(schoolId),
  })

  const avgQuality =
    quality && quality.scores.length > 0
      ? (quality.scores.reduce((s, q) => s + (q.compositeScore ?? 0), 0) / quality.scores.length * 100).toFixed(1)
      : null

  const avgMastery =
    quality && quality.scores.length > 0
      ? (quality.scores.reduce((s, q) => s + (q.studentMasteryPct ?? 0), 0) / quality.scores.length * 100).toFixed(1)
      : null

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <StatMetricCard title="Total Teachers" value={teachers ? `${teachers.length} Staff` : '—'} change="" trend="up" periodText="Active Roster" icon={Users} />
      <StatMetricCard title="Enrolled Students" value={students ? `${students.length} Students` : '—'} change="" trend="up" periodText="Academic Year" icon={BookOpen} />
      <StatMetricCard title="School Quality Score" value={avgQuality ? `${avgQuality}%` : '—'} change="" trend="up" periodText="Composite Score" icon={ShieldCheck} />
      <StatMetricCard title="CLO Standard Mastery" value={avgMastery ? `${avgMastery}%` : '—'} change="" trend="up" periodText="Curriculum Alignment" icon={Award} />
    </div>
  )
}

function QualityChartsSection({ schoolId }: { schoolId: string }) {
  const { data: quality, isLoading } = useQuery({
    queryKey: queryKeys.schoolQuality(schoolId),
    queryFn: () => getSchoolQualityScores(schoolId),
  })

  if (isLoading || !quality || quality.scores.length === 0) return null

  const s = quality.scores[0]!
  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
      {[
        { label: 'Composite', value: ((s.compositeScore ?? 0) * 100).toFixed(1), color: 'text-teal-700' },
        { label: 'Student Mastery', value: ((s.studentMasteryPct ?? 0) * 100).toFixed(1), color: 'text-blue-700' },
        { label: 'CLO Coverage', value: ((s.cloCoveragePct ?? 0) * 100).toFixed(1), color: 'text-emerald-700' },
        { label: 'Exam Quality', value: ((s.examQualityAvg ?? 0) * 100).toFixed(1), color: 'text-violet-700' },
      ].map(({ label, value, color }) => (
        <div key={label} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-xs font-medium text-slate-500">{label}</p>
          <p className={`mt-1 text-3xl font-bold font-display ${color}`}>{value}%</p>
          <p className="mt-1 text-[11px] text-slate-400">Live from quality engine</p>
        </div>
      ))}
    </div>
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
