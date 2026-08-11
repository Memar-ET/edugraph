import { z } from 'zod'

const ACCEPTED_TYPES = [
  'application/pdf',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
]
const MAX_FILE_BYTES = 25 * 1024 * 1024 // matches backend maxExamUploadBytes (exam_upload.go)

export const uploadExamSchema = z.object({
  file: z
    .custom<FileList>((val) => val instanceof FileList && val.length === 1, 'Select a PDF or DOCX file')
    .refine((files) => ACCEPTED_TYPES.includes(files[0]?.type ?? ''), 'Only PDF and DOCX files are allowed')
    .refine((files) => (files[0]?.size ?? 0) <= MAX_FILE_BYTES, 'File must be 25MB or smaller'),
  // subjectCode/gradeLevel/examScope are NOT form fields -- derived
  // server-side from the title (see backend service/title_parser.go), so
  // the title must name both, e.g. "Grade 11 Biology Unit Test - Cell Biology".
  title: z.string().min(1, 'Title is required'),
  academicYear: z.string().min(1, 'Academic year is required'),
  term: z.coerce.number().int().min(1).max(4).optional(),
  totalMarks: z.coerce.number().int().min(1, 'Total marks is required'),
})

export type UploadExamFormValues = z.infer<typeof uploadExamSchema>

const ANSWER_KEY_MAX_BYTES = 10 * 1024 * 1024 // matches backend maxAnswerKeyUploadBytes

export const uploadAnswerKeySchema = z.object({
  file: z
    .custom<FileList>((val) => val instanceof FileList && val.length === 1, 'Select a PDF or DOCX file')
    .refine((files) => ACCEPTED_TYPES.includes(files[0]?.type ?? ''), 'Only PDF and DOCX files are allowed')
    .refine((files) => (files[0]?.size ?? 0) <= ANSWER_KEY_MAX_BYTES, 'File must be 10MB or smaller'),
})

export type UploadAnswerKeyFormValues = z.infer<typeof uploadAnswerKeySchema>
