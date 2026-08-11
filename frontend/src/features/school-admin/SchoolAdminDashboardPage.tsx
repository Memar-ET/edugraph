import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Send, ShieldCheck, Users } from 'lucide-react'
import { useState } from 'react'
import type { FormEvent } from 'react'

import { AppShell } from '@components/layout'
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
      <AppShell title="School dashboard">
        <Banner tone="error">Your account isn&apos;t linked to a school yet. Contact your regional admin.</Banner>
      </AppShell>
    )
  }

  return (
    <AppShell title={school?.name ?? 'School dashboard'} description="Quality scores, roster, and notifications.">
      <div className="space-y-8">
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
    <section>
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-gray-500">
        Quality scores by subject &amp; grade
      </h2>
      {isLoading && (
        <div className="flex items-center gap-2 text-sm text-gray-500">
          <Spinner /> Loading…
        </div>
      )}
      {isError && <Banner tone="error">{apiErrorMessage(error, 'Could not load quality scores.')}</Banner>}
      {data && data.scores.length === 0 && (
        <EmptyState
          icon={ShieldCheck}
          title="No quality scores yet"
          description="Scores compute once exams have been published and graded for a subject and grade."
        />
      )}
      {data && data.scores.length > 0 && <QualityScoreGrid scores={data.scores} />}
    </section>
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
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>Roster</CardTitle>
        <Users className="h-4 w-4 text-gray-400" aria-hidden />
      </CardHeader>
      <CardContent>
        {(loadingTeachers || loadingStudents) && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <Spinner /> Loading…
          </div>
        )}
        {!loadingTeachers && !loadingStudents && (
          <div className="grid grid-cols-2 gap-4 text-center">
            <div className="rounded-lg border border-gray-200 py-4">
              <p className="font-mono text-2xl font-semibold text-gray-900">{teachers?.length ?? 0}</p>
              <p className="text-xs text-gray-500">Teachers</p>
            </div>
            <div className="rounded-lg border border-gray-200 py-4">
              <p className="font-mono text-2xl font-semibold text-gray-900">{students?.length ?? 0}</p>
              <p className="text-xs text-gray-500">Students</p>
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
    <Card>
      <CardHeader>
        <CardTitle>Send a notification</CardTitle>
      </CardHeader>
      <CardContent>
        <form className="space-y-3" onSubmit={handleSubmit}>
          {send.isSuccess && <Banner tone="success">Notification sent.</Banner>}
          {send.isError && <Banner tone="error">{apiErrorMessage(send.error, 'Could not send notification.')}</Banner>}
          <div>
            <Label htmlFor="recipient">Recipient</Label>
            <Select id="recipient" value={userId} onChange={(e) => setUserId(e.target.value)}>
              <option value="">Select a person…</option>
              {options.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </Select>
          </div>
          <div>
            <Label htmlFor="notif-title">Title</Label>
            <Input id="notif-title" value={title} onChange={(e) => setTitle(e.target.value)} />
          </div>
          <div>
            <Label htmlFor="notif-body">Message</Label>
            <textarea
              id="notif-body"
              value={body}
              onChange={(e) => setBody(e.target.value)}
              rows={3}
              className="flex w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
            />
          </div>
          <Button type="submit" isLoading={send.isPending} className="w-full">
            <Send className="h-4 w-4" aria-hidden />
            Send
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
