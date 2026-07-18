import { zodResolver } from '@hookform/resolvers/zod'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { useForm } from 'react-hook-form'

import { apiErrorMessage } from '@lib/api/client'
import { uploadExam } from '@lib/api/endpoints'
import { uploadExamSchema, type UploadExamFormValues } from '@lib/validations/exam'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Input, Label } from '@components/ui'
import { AppHeader } from '@components/layout/AppHeader'

export function ExamUploadPage() {
  const navigate = useNavigate()
  const [serverError, setServerError] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<UploadExamFormValues>({ resolver: zodResolver(uploadExamSchema) })

  const onSubmit = async (values: UploadExamFormValues) => {
    setServerError(null)
    const file = values.file[0]
    if (!file) {
      setServerError('Select a file to upload.')
      return
    }
    try {
      const res = await uploadExam({
        file,
        title: values.title,
        academicYear: values.academicYear,
        term: values.term,
        totalMarks: values.totalMarks,
      })
      void navigate({ to: '/teacher/exams/$examId', params: { examId: res.examId } })
    } catch (err) {
      setServerError(apiErrorMessage(err, 'Upload failed. Please try again.'))
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <AppHeader />
      <main className="mx-auto max-w-xl px-4 py-8">
        <Card>
          <CardHeader>
            <CardTitle>Upload exam</CardTitle>
            <p className="mt-1 text-sm text-gray-500">
              Step 1: submit an exam PDF or DOCX for AI parsing. The subject, grade level, and exam type
              (Unit Test / Midterm / Final Exam) are read from the title below -- include all three,
              e.g. <span className="font-medium">&quot;Grade 11 Biology Unit Test - Cell Biology&quot;</span>.
            </p>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={handleSubmit(onSubmit)} noValidate>
              {serverError && <Banner tone="error">{serverError}</Banner>}

              <div>
                <Label htmlFor="file">Exam file (PDF or DOCX, max 25MB)</Label>
                <Input id="file" type="file" accept=".pdf,.docx" {...register('file')} />
                {errors.file && <p className="mt-1 text-sm text-red-600">{errors.file.message as string}</p>}
              </div>

              <div>
                <Label htmlFor="title">Title</Label>
                <Input
                  id="title"
                  placeholder="Grade 11 Biology Unit Test - Cell Biology"
                  {...register('title')}
                />
                {errors.title && <p className="mt-1 text-sm text-red-600">{errors.title.message}</p>}
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="academicYear">Academic year</Label>
                  <Input id="academicYear" placeholder="2018 E.C." {...register('academicYear')} />
                  {errors.academicYear && (
                    <p className="mt-1 text-sm text-red-600">{errors.academicYear.message}</p>
                  )}
                </div>
                <div>
                  <Label htmlFor="term">Term (optional)</Label>
                  <Input id="term" type="number" min={1} max={4} {...register('term')} />
                  {errors.term && <p className="mt-1 text-sm text-red-600">{errors.term.message}</p>}
                </div>
              </div>

              <div>
                <Label htmlFor="totalMarks">Total marks</Label>
                <Input id="totalMarks" type="number" min={1} {...register('totalMarks')} />
                <p className="mt-1 text-xs text-gray-500">
                  Unit Test: 10 or 15 &middot; Midterm: 15 or 20 &middot; Final Exam: 40
                </p>
                {errors.totalMarks && <p className="mt-1 text-sm text-red-600">{errors.totalMarks.message}</p>}
              </div>

              <Button type="submit" className="w-full" isLoading={isSubmitting}>
                Upload and queue for parsing
              </Button>
            </form>
          </CardContent>
        </Card>
      </main>
    </div>
  )
}
