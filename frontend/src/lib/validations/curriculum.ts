import { z } from 'zod'

const ACCEPTED_TYPES = [
  'application/pdf',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
]
const MAX_FILE_BYTES = 25 * 1024 * 1024 // matches backend maxUploadBytes (handler.go)

export const uploadCurriculumSchema = z.object({
  file: z
    .custom<FileList>((val) => val instanceof FileList && val.length === 1, 'Select a PDF or DOCX file')
    .refine((files) => ACCEPTED_TYPES.includes(files[0]?.type ?? ''), 'Only PDF and DOCX files are allowed')
    .refine((files) => (files[0]?.size ?? 0) <= MAX_FILE_BYTES, 'File must be 25MB or smaller'),
  subjectCode: z
    .string()
    .min(1, 'Subject code is required')
    .max(20)
    .regex(/^[A-Za-z0-9_-]+$/, 'Use letters, numbers, - or _ only'),
  gradeLevel: z.coerce.number().int().min(1, 'Grade must be 1-12').max(12, 'Grade must be 1-12'),
  academicYear: z
    .string()
    .min(1, 'Academic year is required')
    .regex(/^\d{4}-\d{4}$/, 'Format: YYYY-YYYY, e.g. 2025-2026'),
})

export type UploadCurriculumFormValues = z.infer<typeof uploadCurriculumSchema>
