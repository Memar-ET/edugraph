import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { AlertTriangle, ArrowRight, UploadCloud } from 'lucide-react'
import { useState } from 'react'

import { AppShell } from '@components/layout'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, Input, Spinner } from '@components/ui'
import { HeatmapGrid } from '@components/charts'
import { apiErrorMessage } from '@lib/api/client'
import { getClassHeatmap } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function TeacherDashboardPage() {
  const navigate = useNavigate()
  const [examId, setExamId] = useState('')
  const [subjectCode, setSubjectCode] = useState('')
  const [gradeLevel, setGradeLevel] = useState('')
  const [scope, setScope] = useState<{ subjectCode: string; gradeLevel: number } | null>(null)

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.classHeatmap(scope?.subjectCode ?? '', scope?.gradeLevel ?? 0),
    queryFn: () => getClassHeatmap(scope!.subjectCode, scope!.gradeLevel),
    enabled: Boolean(scope),
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

  return (
    <AppShell
      title="Class heatmap"
      description={data ? `${data.subjectCode} · Grade ${data.gradeLevel} · ${data.classSize} students` : undefined}
      actions={
        <Button size="sm" onClick={() => void navigate({ to: '/teacher/exams/upload' })}>
          <UploadCloud className="h-4 w-4" aria-hidden />
          Upload exam
        </Button>
      }
    >
      <div className="grid gap-6 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          {data?.alerts && data.alerts.length > 0 && (
            <div className="space-y-2">
              {data.alerts.map((alert) => (
                <Banner key={alert.topicId} tone="warning">
                  <div className="flex items-start gap-2">
                    <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
                    <span>{alert.message}</span>
                  </div>
                </Banner>
              ))}
            </div>
          )}

          <Card>
            <CardHeader>
              <CardTitle>Topics ranked by struggle</CardTitle>
              <p className="mt-1 text-sm text-gray-500">
                Share of your class struggling with each topic, based on gap records traced from exam
                attempts. Pick the subject and grade you teach to load it.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap items-end gap-2">
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-500" htmlFor="subjectCode">
                    Subject code
                  </label>
                  <Input
                    id="subjectCode"
                    value={subjectCode}
                    onChange={(e) => setSubjectCode(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && loadHeatmap()}
                    placeholder="BIO"
                    className="w-28"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-500" htmlFor="gradeLevel">
                    Grade
                  </label>
                  <Input
                    id="gradeLevel"
                    type="number"
                    min={1}
                    max={12}
                    value={gradeLevel}
                    onChange={(e) => setGradeLevel(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && loadHeatmap()}
                    className="w-20"
                  />
                </div>
                <Button variant="secondary" onClick={loadHeatmap}>
                  Load heatmap
                </Button>
              </div>

              {isLoading && (
                <div className="flex items-center gap-2 text-sm text-gray-500">
                  <Spinner /> Loading heatmap…
                </div>
              )}
              {isError && <Banner tone="error">{apiErrorMessage(error, 'Could not load the class heatmap.')}</Banner>}
              {!scope && (
                <EmptyState title="Pick a subject and grade" description="Load the heatmap for a class you teach." />
              )}
              {scope && data && data.topics.length === 0 && (
                <EmptyState
                  title="No gap data yet"
                  description="Once students take published exams, struggling topics will surface here automatically."
                />
              )}
              {data && data.topics.length > 0 && <HeatmapGrid topics={data.topics} />}
            </CardContent>
          </Card>
        </div>

        <Card className="h-fit">
          <CardHeader>
            <CardTitle className="text-base">Jump to an exam</CardTitle>
            <p className="mt-1 text-sm text-gray-500">Paste an exam link or ID to review, publish, or grade it.</p>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Input
                value={examId}
                onChange={(e) => setExamId(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && openExam()}
                placeholder="/teacher/exams/…"
                aria-label="Exam link or ID"
              />
              <Button variant="secondary" size="md" onClick={openExam} aria-label="Open exam">
                <ArrowRight className="h-4 w-4" aria-hidden />
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </AppShell>
  )
}
