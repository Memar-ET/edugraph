import { zodResolver } from '@hookform/resolvers/zod'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { useForm } from 'react-hook-form'

import { apiErrorMessage } from '@lib/api/client'
import { uploadCurriculum } from '@lib/api/endpoints'
import {
  uploadCurriculumSchema,
  type UploadCurriculumFormValues,
} from '@lib/validations/curriculum'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Input, Label } from '@components/ui'
import { AppHeader } from '@components/layout/AppHeader'

export function UploadPage() {
  const navigate = useNavigate()
  const [serverError, setServerError] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<UploadCurriculumFormValues>({ resolver: zodResolver(uploadCurriculumSchema) })

  const onSubmit = async (values: UploadCurriculumFormValues) => {
    setServerError(null)
    const file = values.file[0]
    if (!file) {
      // Unreachable in practice -- uploadCurriculumSchema requires exactly
      // one file -- but noUncheckedIndexedAccess types FileList[0] as
      // possibly undefined, so guard explicitly rather than asserting.
      setServerError('Select a file to upload.')
      return
    }
    try {
      const res = await uploadCurriculum({
        file,
        subjectCode: values.subjectCode,
        gradeLevel: values.gradeLevel,
        academicYear: values.academicYear,
      })
      void navigate({ to: '/curriculum/jobs/$jobId', params: { jobId: res.jobId } })
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
            <CardTitle>Upload curriculum document</CardTitle>
            <p className="mt-1 text-sm text-gray-500">
              Step 1: submit a curriculum PDF or DOCX for AI parsing. You&apos;ll review and approve the
              extracted structure once parsing finishes.
            </p>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={handleSubmit(onSubmit)} noValidate>
              {serverError && <Banner tone="error">{serverError}</Banner>}

              <div>
                <Label htmlFor="file">Curriculum file (PDF or DOCX, max 25MB)</Label>
                <Input id="file" type="file" accept=".pdf,.docx" {...register('file')} />
                {errors.file && <p className="mt-1 text-sm text-red-600">{errors.file.message as string}</p>}
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="subjectCode">Subject code</Label>
                  <Input id="subjectCode" placeholder="PHY" {...register('subjectCode')} />
                  {errors.subjectCode && (
                    <p className="mt-1 text-sm text-red-600">{errors.subjectCode.message}</p>
                  )}
                </div>
                <div>
                  <Label htmlFor="gradeLevel">Grade level</Label>
                  <Input id="gradeLevel" type="number" min={1} max={12} {...register('gradeLevel')} />
                  {errors.gradeLevel && (
                    <p className="mt-1 text-sm text-red-600">{errors.gradeLevel.message}</p>
                  )}
                </div>
              </div>

              <div>
                <Label htmlFor="academicYear">Academic year</Label>
                <Input id="academicYear" placeholder="2025-2026" {...register('academicYear')} />
                {errors.academicYear && (
                  <p className="mt-1 text-sm text-red-600">{errors.academicYear.message}</p>
                )}
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
