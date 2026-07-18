import { apiClient } from './client'
import type {
  ApproveRequest,
  ApproveResponse,
  AuthResponse,
  BulkGradeRequest,
  BulkGradeResponse,
  Envelope,
  ExamQuestion,
  ExamStatus,
  GradingQuestion,
  JobStatus,
  LoginRequest,
  PublishResponse,
  StudentResponse,
  SubmitExamRequest,
  SubmitExamResponse,
  UploadAnswerKeyResponse,
  UploadExamResponse,
  UploadResponse,
  ValidationReport,
} from '@/types/api'

function unwrap<T>(envelope: Envelope<T>): T {
  if (!envelope.success || envelope.data === undefined) {
    throw new Error(envelope.error || 'Request failed')
  }
  return envelope.data
}

export async function login(payload: LoginRequest): Promise<AuthResponse> {
  const res = await apiClient.post<Envelope<AuthResponse>>('/auth/login', payload)
  return unwrap(res.data)
}

export interface UploadCurriculumPayload {
  file: File
  subjectCode: string
  gradeLevel: number
  academicYear: string
}

export async function uploadCurriculum(payload: UploadCurriculumPayload): Promise<UploadResponse> {
  const form = new FormData()
  form.append('file', payload.file)
  form.append('subjectCode', payload.subjectCode)
  form.append('gradeLevel', String(payload.gradeLevel))
  form.append('academicYear', payload.academicYear)

  const res = await apiClient.post<Envelope<UploadResponse>>('/curriculum/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return unwrap(res.data)
}

export async function getCurriculumJob(jobId: string): Promise<JobStatus> {
  const res = await apiClient.get<Envelope<JobStatus>>(`/curriculum/jobs/${jobId}`)
  return unwrap(res.data)
}

export async function approveCurriculumJob(
  jobId: string,
  payload: ApproveRequest,
): Promise<ApproveResponse> {
  const res = await apiClient.post<Envelope<ApproveResponse>>(
    `/curriculum/jobs/${jobId}/approve`,
    payload,
  )
  return unwrap(res.data)
}

/**
 * The storage proxy endpoint requires a Bearer token, which a plain <a href>
 * or <img src> can't attach -- so this fetches the file as a blob (with the
 * apiClient auth interceptor doing its job) and hands back an object URL the
 * caller can open in a new tab or embed in an <iframe>. Caller is responsible
 * for URL.revokeObjectURL() when done with it.
 */
export async function fetchCurriculumFileBlobUrl(jobId: string): Promise<string> {
  const res = await apiClient.get(`/storage/files/${jobId}`, { responseType: 'blob' })
  return URL.createObjectURL(res.data as Blob)
}

// ── Assessment (Capabilities 2A/2B/2C) ──────────────────────────

export interface UploadExamPayload {
  file: File
  title: string
  academicYear: string
  term?: number
  totalMarks: number
}

export async function uploadExam(payload: UploadExamPayload): Promise<UploadExamResponse> {
  const form = new FormData()
  form.append('file', payload.file)
  form.append('title', payload.title)
  form.append('academicYear', payload.academicYear)
  if (payload.term !== undefined) form.append('term', String(payload.term))
  form.append('totalMarks', String(payload.totalMarks))

  const res = await apiClient.post<Envelope<UploadExamResponse>>('/exams/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return unwrap(res.data)
}

export async function uploadAnswerKey(examId: string, file: File): Promise<UploadAnswerKeyResponse> {
  const form = new FormData()
  form.append('file', file)

  const res = await apiClient.post<Envelope<UploadAnswerKeyResponse>>(
    `/exams/${examId}/answer-key`,
    form,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return unwrap(res.data)
}

export async function getExam(examId: string): Promise<ExamStatus> {
  const res = await apiClient.get<Envelope<ExamStatus>>(`/exams/${examId}`)
  return unwrap(res.data)
}

export async function validateExam(examId: string): Promise<ValidationReport> {
  const res = await apiClient.post<Envelope<ValidationReport>>(`/exams/${examId}/validate`)
  return unwrap(res.data)
}

export async function publishExam(examId: string): Promise<PublishResponse> {
  const res = await apiClient.post<Envelope<PublishResponse>>(`/exams/${examId}/publish`)
  return unwrap(res.data)
}

export async function listExamQuestions(examId: string): Promise<ExamQuestion[]> {
  const res = await apiClient.get<Envelope<ExamQuestion[]>>(`/exams/${examId}/questions`)
  return unwrap(res.data)
}

export async function submitExam(examId: string, payload: SubmitExamRequest): Promise<SubmitExamResponse> {
  const res = await apiClient.post<Envelope<SubmitExamResponse>>(`/exams/${examId}/submit`, payload)
  return unwrap(res.data)
}

export async function bulkGradeExam(examId: string, payload: BulkGradeRequest): Promise<BulkGradeResponse> {
  const res = await apiClient.post<Envelope<BulkGradeResponse>>(`/exams/${examId}/grades/bulk`, payload)
  return unwrap(res.data)
}

export async function listQuestionsForGrading(examId: string): Promise<GradingQuestion[]> {
  const res = await apiClient.get<Envelope<GradingQuestion[]>>(`/exams/${examId}/grading-questions`)
  return unwrap(res.data)
}

// ── Students (grading spreadsheet's roster) ─────────────────────

export async function listStudents(schoolId: string): Promise<StudentResponse[]> {
  const res = await apiClient.get<Envelope<StudentResponse[]>>('/students', {
    params: { school_id: schoolId, limit: 100 },
  })
  return unwrap(res.data)
}
