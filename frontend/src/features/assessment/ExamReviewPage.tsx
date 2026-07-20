import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useState } from 'react'
import { useForm } from 'react-hook-form'

import { apiErrorMessage } from '@lib/api/client'
import { getExam, publishExam, uploadAnswerKey, validateExam } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { uploadAnswerKeySchema, type UploadAnswerKeyFormValues } from '@lib/validations/exam'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Input, Spinner } from '@components/ui'
import { AppShell } from '@components/layout'
import type { ValidationReport } from '@/types/api'

const IN_PROGRESS_STATUSES = new Set(['pending', 'parsing'])

export function ExamReviewPage() {
  const { examId } = useParams({ from: '/teacher/exams/$examId' })
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const [actionError, setActionError] = useState<string | null>(null)
  const [validating, setValidating] = useState(false)
  const [publishing, setPublishing] = useState(false)

  const { data: exam, isLoading, error } = useQuery({
    queryKey: queryKeys.exam(examId),
    queryFn: () => getExam(examId),
    refetchInterval: (query) => (IN_PROGRESS_STATUSES.has(query.state.data?.status ?? '') ? 3000 : false),
  })

  const {
    register: registerKey,
    handleSubmit: handleKeySubmit,
    formState: { errors: keyErrors, isSubmitting: uploadingKey },
    reset: resetKeyForm,
  } = useForm<UploadAnswerKeyFormValues>({ resolver: zodResolver(uploadAnswerKeySchema) })

  const onUploadAnswerKey = async (values: UploadAnswerKeyFormValues) => {
    setActionError(null)
    const file = values.file[0]
    if (!file) return
    try {
      await uploadAnswerKey(examId, file)
      resetKeyForm()
      setActionError(null)
    } catch (err) {
      setActionError(apiErrorMessage(err, 'Answer key upload failed.'))
    }
  }

  const handleValidate = async () => {
    setActionError(null)
    setValidating(true)
    try {
      await validateExam(examId)
      await queryClient.invalidateQueries({ queryKey: queryKeys.exam(examId) })
    } catch (err) {
      setActionError(apiErrorMessage(err, 'Validation failed.'))
    } finally {
      setValidating(false)
    }
  }

  const handlePublish = async () => {
    setActionError(null)
    setPublishing(true)
    try {
      await publishExam(examId)
      await queryClient.invalidateQueries({ queryKey: queryKeys.exam(examId) })
    } catch (err) {
      setActionError(apiErrorMessage(err, 'Publish failed.'))
    } finally {
      setPublishing(false)
    }
  }

  return (
    <AppShell title="Review exam" description="Steps 2/3: AI validation and publishing.">
      <div className="mx-auto max-w-3xl space-y-4">
        <Button variant="ghost" size="sm" onClick={() => void navigate({ to: '/teacher/exams/upload' })}>
          ← Upload another exam
        </Button>

        {isLoading && (
          <div className="flex items-center gap-2 text-gray-500">
            <Spinner /> Loading exam…
          </div>
        )}
        {error && <Banner tone="error">{apiErrorMessage(error, 'Could not load this exam.')}</Banner>}

        {exam && (
          <>
            <Card>
              <CardHeader>
                <CardTitle>{exam.title}</CardTitle>
                <p className="mt-1 text-sm text-gray-500">
                  {exam.subjectCode} · Grade {exam.gradeLevel} · {exam.examScope} · {exam.academicYear} ·{' '}
                  {exam.totalMarks} marks · {exam.questionCount} question(s)
                </p>
              </CardHeader>
              <CardContent>
                <StatusBanner status={exam.status} error={exam.parseError} />
              </CardContent>
            </Card>

            {actionError && <Banner tone="error">{actionError}</Banner>}

            {IN_PROGRESS_STATUSES.has(exam.status) && (
              <Banner tone="info">
                <div className="flex items-center gap-2">
                  <Spinner className="h-4 w-4" /> Parsing is in progress — this page refreshes automatically.
                </div>
              </Banner>
            )}

            {(exam.status === 'draft' || exam.status === 'validation_pending') && (
              <Card>
                <CardHeader>
                  <CardTitle>Answer key (optional)</CardTitle>
                  <p className="mt-1 text-sm text-gray-500">
                    Upload a separate Answer Key document to enable instant MCQ auto-grading. Applies
                    asynchronously to matching questions by number.
                  </p>
                </CardHeader>
                <CardContent>
                  <form className="flex items-end gap-3" onSubmit={handleKeySubmit(onUploadAnswerKey)} noValidate>
                    <div className="flex-1">
                      <Input type="file" accept=".pdf,.docx" {...registerKey('file')} />
                      {keyErrors.file && (
                        <p className="mt-1 text-sm text-red-600">{keyErrors.file.message as string}</p>
                      )}
                    </div>
                    <Button type="submit" variant="secondary" isLoading={uploadingKey}>
                      Upload key
                    </Button>
                  </form>
                </CardContent>
              </Card>
            )}

            {exam.status === 'draft' && (
              <div className="flex justify-end">
                <Button size="lg" isLoading={validating} onClick={() => void handleValidate()}>
                  Validate exam
                </Button>
              </div>
            )}

            {exam.validationReport && <ValidationReportView report={exam.validationReport} />}

            {exam.status === 'validation_pending' && (
              <div className="sticky bottom-4 flex justify-end gap-2">
                <Button variant="secondary" isLoading={validating} onClick={() => void handleValidate()}>
                  Re-validate
                </Button>
                <Button size="lg" isLoading={publishing} onClick={() => void handlePublish()}>
                  Publish
                </Button>
              </div>
            )}

            {exam.status === 'published' && (
              <Card>
                <CardContent className="space-y-3 pt-6">
                  <Banner tone="success">This exam is published and visible to matching students.</Banner>
                  <div className="flex flex-wrap gap-2">
                    <Button
                      variant="secondary"
                      onClick={() => void navigate({ to: '/teacher/exams/$examId/grade', params: { examId } })}
                    >
                      Grade this exam
                    </Button>
                    <Button
                      variant="secondary"
                      onClick={() => void navigate({ to: '/teacher/exams/$examId/quality', params: { examId } })}
                    >
                      View quality report
                    </Button>
                  </div>
                  <p className="text-sm text-gray-500">
                    Student link: <code className="rounded bg-gray-100 px-1.5 py-0.5">/student/exams/{examId}</code>
                  </p>
                </CardContent>
              </Card>
            )}
          </>
        )}
      </div>
    </AppShell>
  )
}

function StatusBanner({ status, error }: { status: string; error?: string }) {
  if (status === 'failed') {
    return <Banner tone="error">Parsing failed: {error ?? 'Unknown error.'}</Banner>
  }
  if (status === 'published') {
    return <Banner tone="success">Published.</Banner>
  }
  if (status === 'validation_pending') {
    return <Banner tone="info">Validated — review the report below, then publish.</Banner>
  }
  return <Banner tone="info">Status: {status}</Banner>
}

function pct(n: number): string {
  return `${Math.round(n)}%`
}

function ValidationReportView({ report }: { report: ValidationReport }) {
  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>AI Validation Report</CardTitle>
          <p className="mt-1 text-xs text-gray-500">
            Scope: {report.scope} · Generated {new Date(report.generatedAt).toLocaleString()}
          </p>
        </CardHeader>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">1. CLO Coverage</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <p>{report.cloCoverage.summary}</p>
          <p className="text-gray-500">
            {report.cloCoverage.coveredClos} of {report.cloCoverage.totalClos} CLOs covered overall.
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">2. Bloom&apos;s Taxonomy Balance</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <p className={report.bloomBalance.meetsMinimumHigherOrder ? 'text-green-700' : 'text-amber-700'}>
            {report.bloomBalance.summary}
          </p>
          <div className="flex flex-wrap gap-2 text-xs text-gray-600">
            {Object.entries(report.bloomBalance.percentages).map(([level, value]) => (
              <span key={level} className="rounded bg-gray-100 px-2 py-1">
                {level}: {pct(value)}
              </span>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">3. Difficulty Distribution</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <p className={report.difficultyDistribution.exceedsMaxHard ? 'text-amber-700' : 'text-green-700'}>
            {report.difficultyDistribution.summary}
          </p>
          <div className="flex flex-wrap gap-2 text-xs text-gray-600">
            {Object.entries(report.difficultyDistribution.percentages).map(([level, value]) => (
              <span key={level} className="rounded bg-gray-100 px-2 py-1">
                {level}: {pct(value)}
              </span>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">4. Topic Coverage</CardTitle>
        </CardHeader>
        <CardContent>
          <ul className="divide-y divide-gray-100 text-sm">
            {report.topicCoverage.map((t) => (
              <li key={`${t.unitNumber}-${t.topicTitle}`} className="flex items-center justify-between py-1.5">
                <span>
                  <span className="text-gray-400">Unit {t.unitNumber}</span> · {t.topicTitle}
                </span>
                <span className={t.questionCount === 0 ? 'font-medium text-amber-700' : 'text-gray-600'}>
                  {t.questionCount} question{t.questionCount === 1 ? '' : 's'}
                </span>
              </li>
            ))}
            {report.topicCoverage.length === 0 && <li className="py-1.5 text-gray-500">No topics in scope.</li>}
          </ul>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">5. Cross-Grade Prerequisite Warnings</CardTitle>
        </CardHeader>
        <CardContent>
          {report.prerequisiteWarnings.length === 0 ? (
            <p className="text-sm text-gray-500">
              No prerequisite relationships defined in the curriculum graph yet — nothing to warn about.
            </p>
          ) : (
            <ul className="space-y-1.5 text-sm">
              {report.prerequisiteWarnings.map((w, i) => (
                <li key={i} className="text-amber-700">
                  {w.message}
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
