import { useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useMemo, useState } from 'react'

import { apiErrorMessage } from '@lib/api/client'
import { bulkGradeExam, getExam, listQuestionsForGrading, listStudents } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Spinner } from '@components/ui'
import { AppHeader } from '@components/layout/AppHeader'
import { useAuthStore } from '@stores/auth.store'
import type { BulkGradeResponse, GradeEntry } from '@/types/api'

// Flow 2 (paper/teacher-encoded): a spreadsheet -- students down the left,
// questions across the top. Cell value is an MCQ option letter or the
// teacher's own numeric marks (see backend service.gradeTeacherEntry --
// numeric input always wins, for any question type).
export function GradeExamPage() {
  const { examId } = useParams({ from: '/teacher/exams/$examId/grade' })
  const navigate = useNavigate()
  const schoolId = useAuthStore((s) => s.user?.school_id)

  const [grades, setGrades] = useState<Record<string, Record<string, string>>>({})
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saveResult, setSaveResult] = useState<BulkGradeResponse | null>(null)

  const { data: exam } = useQuery({ queryKey: queryKeys.exam(examId), queryFn: () => getExam(examId) })
  const { data: questions, isLoading: loadingQuestions } = useQuery({
    queryKey: queryKeys.gradingQuestions(examId),
    queryFn: () => listQuestionsForGrading(examId),
  })
  const { data: students, isLoading: loadingStudents } = useQuery({
    queryKey: queryKeys.students(schoolId ?? ''),
    queryFn: () => listStudents(schoolId ?? ''),
    enabled: Boolean(schoolId),
  })

  const classStudents = useMemo(
    () => (students ?? []).filter((s) => !exam || s.grade_level === exam.gradeLevel),
    [students, exam],
  )

  const setCell = (studentId: string, questionId: string, value: string) => {
    setGrades((prev) => ({ ...prev, [studentId]: { ...prev[studentId], [questionId]: value } }))
  }

  const handleSaveAll = async () => {
    if (!questions) return
    setSaveError(null)
    setSaveResult(null)

    const entries: GradeEntry[] = []
    for (const [studentId, byQuestion] of Object.entries(grades)) {
      for (const [questionId, value] of Object.entries(byQuestion)) {
        if (value.trim() !== '') entries.push({ studentId, questionId, value: value.trim() })
      }
    }
    if (entries.length === 0) {
      setSaveError('Enter at least one mark before saving.')
      return
    }

    setSaving(true)
    try {
      const res = await bulkGradeExam(examId, { entries })
      setSaveResult(res)
    } catch (err) {
      setSaveError(apiErrorMessage(err, 'Save failed.'))
    } finally {
      setSaving(false)
    }
  }

  const loading = loadingQuestions || loadingStudents

  return (
    <div className="min-h-screen bg-gray-50">
      <AppHeader />
      <main className="mx-auto max-w-6xl px-4 py-8 space-y-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => void navigate({ to: '/teacher/exams/$examId', params: { examId } })}
        >
          ← Back to exam
        </Button>

        <Card>
          <CardHeader>
            <CardTitle>Grade exam{exam ? `: ${exam.title}` : ''}</CardTitle>
            <p className="mt-1 text-sm text-gray-500">
              Type a mark (e.g. &quot;2&quot;) for any question, or an option letter for MCQ. Click
              &quot;Save All&quot; when done — safe to re-save, it only updates what changed.
            </p>
          </CardHeader>
        </Card>

        {saveError && <Banner tone="error">{saveError}</Banner>}
        {saveResult && (
          <Banner tone="success">
            Saved {saveResult.answersSaved} answer(s) across {saveResult.attemptsTouched} student(s).
          </Banner>
        )}

        {loading && (
          <div className="flex items-center gap-2 text-gray-500">
            <Spinner /> Loading roster…
          </div>
        )}

        {questions && classStudents.length === 0 && !loading && (
          <Banner tone="info">No students found for this exam&apos;s grade at your school.</Banner>
        )}

        {questions && classStudents.length > 0 && (
          <Card>
            <CardContent className="overflow-x-auto pt-6">
              <table className="min-w-full border-collapse text-sm">
                <thead>
                  <tr>
                    <th className="sticky left-0 border-b border-gray-200 bg-white px-3 py-2 text-left font-medium text-gray-700">
                      Student
                    </th>
                    {questions.map((q) => (
                      <th
                        key={q.id}
                        className="border-b border-gray-200 px-2 py-2 text-center font-medium text-gray-700"
                        title={q.questionText}
                      >
                        Q{q.sequenceNumber}
                        <div className="text-xs font-normal text-gray-400">
                          {q.questionType === 'mcq' ? 'MCQ' : `/${q.marks}`}
                          {q.answerKey?.correctOption ? ` (${q.answerKey.correctOption})` : ''}
                        </div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {classStudents.map((student) => (
                    <tr key={student.id} className="border-b border-gray-100">
                      <td className="sticky left-0 bg-white px-3 py-1.5 font-medium text-gray-900">
                        {student.admission_no}
                      </td>
                      {questions.map((q) => (
                        <td key={q.id} className="px-2 py-1.5">
                          <input
                            className="w-14 rounded border border-gray-300 px-1.5 py-1 text-center text-sm"
                            value={grades[student.id]?.[q.id] ?? ''}
                            onChange={(e) => setCell(student.id, q.id, e.target.value)}
                            placeholder={q.questionType === 'mcq' ? 'A-D' : '0'}
                          />
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </CardContent>
          </Card>
        )}

        {questions && classStudents.length > 0 && (
          <div className="sticky bottom-4 flex justify-end">
            <Button size="lg" isLoading={saving} onClick={() => void handleSaveAll()}>
              Save All
            </Button>
          </div>
        )}
      </main>
    </div>
  )
}
