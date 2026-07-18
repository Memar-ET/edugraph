import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'

import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Input, Label } from '@components/ui'
import { AppHeader } from '@components/layout/AppHeader'

// There's no "list published exams for my grade" endpoint yet -- a
// teacher shares the exam link (shown on ExamReviewPage once published)
// directly with students. This page just accepts that exam id and
// forwards to the actual exam-taking page.
export function StudentExamListPage() {
  const navigate = useNavigate()
  const [examId, setExamId] = useState('')
  const [error, setError] = useState<string | null>(null)

  const handleGo = () => {
    const trimmed = examId.trim()
    if (!trimmed) {
      setError('Enter the exam link or ID your teacher shared.')
      return
    }
    // Accept either a bare id or a pasted "/student/exams/<id>" link.
    const id = trimmed.includes('/') ? (trimmed.split('/').filter(Boolean).pop() ?? trimmed) : trimmed
    void navigate({ to: '/student/exams/$examId', params: { examId: id } })
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <AppHeader />
      <main className="mx-auto max-w-md px-4 py-8">
        <Card>
          <CardHeader>
            <CardTitle>Take an exam</CardTitle>
            <p className="mt-1 text-sm text-gray-500">
              Paste the exam link or ID your teacher shared with you.
            </p>
          </CardHeader>
          <CardContent className="space-y-4">
            {error && <Banner tone="error">{error}</Banner>}
            <div>
              <Label htmlFor="examId">Exam link or ID</Label>
              <Input
                id="examId"
                value={examId}
                onChange={(e) => setExamId(e.target.value)}
                placeholder="/student/exams/…"
              />
            </div>
            <Button className="w-full" onClick={handleGo}>
              Continue
            </Button>
          </CardContent>
        </Card>
      </main>
    </div>
  )
}
