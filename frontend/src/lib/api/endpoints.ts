import { apiClient } from './client'
import type {
  ApproveRequest,
  ApproveResponse,
  AuthResponse,
  BulkGradeRequest,
  BulkGradeResponse,
  CareerMatchResponse,
  CareerPathResponse,
  ClassHeatmapResponse,
  CreateCareerPathRequest,
  CreateNotificationRequest,
  Envelope,
  ExamInsight,
  ExamInsightListEntry,
  ExamQualityResponse,
  ExamQuestion,
  ExamStatus,
  GenerateStudyPlanRequest,
  GenerateStudyPlanResponse,
  GradingQuestion,
  JobStatus,
  LoginRequest,
  MinistryOverviewResponse,
  NotificationResponse,
  PublishResponse,
  RegionResponse,
  RegionStatsResponse,
  SchoolQualityResponse,
  SchoolResponse,
  StudentResponse,
  StudyPlan,
  SubjectProfile,
  SubmitExamRequest,
  SubmitExamResponse,
  TeacherResponse,
  TutorAskRequest,
  TutorAskResponse,
  UpdateExamScopeRequest,
  UpdateExamScopeResponse,
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

// Capability 2.2: print-ready exam sheet / answer key. Same blob-url
// pattern as fetchCurriculumFileBlobUrl -- a plain <a href> can't attach
// the Bearer token these endpoints require, so we fetch as a blob (the
// apiClient auth interceptor handles that) and open the resulting object
// URL in a new tab, where the page's own "Print / Save as PDF" button
// (baked into the returned HTML) takes over.
export async function fetchExamPrintBlobUrl(examId: string): Promise<string> {
  const res = await apiClient.get(`/exams/${examId}/print`, { responseType: 'blob' })
  return URL.createObjectURL(res.data as Blob)
}

export async function fetchAnswerKeyPrintBlobUrl(examId: string): Promise<string> {
  const res = await apiClient.get(`/exams/${examId}/print/answer-key`, { responseType: 'blob' })
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

// Capability 2D: fix a wrong subject/grade/exam-type/unit-range without
// re-uploading the exam file -- call validateExam() again afterwards to
// refresh the compliance report against the corrected scope.
export async function updateExamScope(
  examId: string,
  payload: UpdateExamScopeRequest,
): Promise<UpdateExamScopeResponse> {
  const res = await apiClient.patch<Envelope<UpdateExamScopeResponse>>(`/exams/${examId}/scope`, payload)
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

// ── Subject health + study plans (Capabilities 3A/3B) ────────────

export async function getMySubjectProfiles(): Promise<SubjectProfile[]> {
  const res = await apiClient.get<Envelope<SubjectProfile[]>>('/students/me/subject-profiles')
  return unwrap(res.data)
}

export async function generateStudyPlan(
  payload: GenerateStudyPlanRequest = {},
): Promise<GenerateStudyPlanResponse> {
  const res = await apiClient.post<Envelope<GenerateStudyPlanResponse>>('/students/me/study-plans', payload)
  return unwrap(res.data)
}

export async function listMyStudyPlans(): Promise<StudyPlan[]> {
  const res = await apiClient.get<Envelope<StudyPlan[]>>('/students/me/study-plans')
  return unwrap(res.data)
}

// ── Career (Capability: career matching) ──────────────────────────

export async function getCareerMatches(studentId: string): Promise<CareerMatchResponse[]> {
  const res = await apiClient.get<Envelope<CareerMatchResponse[]>>(
    `/students/${studentId}/career/matches`,
  )
  return unwrap(res.data)
}

export async function generateCareerMatches(studentId: string): Promise<CareerMatchResponse[]> {
  const res = await apiClient.post<Envelope<CareerMatchResponse[]>>(
    `/students/${studentId}/career/generate`,
  )
  return unwrap(res.data)
}

export async function listCareerPaths(): Promise<CareerPathResponse[]> {
  const res = await apiClient.get<Envelope<CareerPathResponse[]>>('/career/paths')
  return unwrap(res.data)
}

export async function createCareerPath(payload: CreateCareerPathRequest): Promise<CareerPathResponse> {
  const res = await apiClient.post<Envelope<CareerPathResponse>>('/career/paths', payload)
  return unwrap(res.data)
}

// ── AI Tutor (Capability 3C) ──────────────────────────────────────

export async function askTutor(payload: TutorAskRequest): Promise<TutorAskResponse> {
  const res = await apiClient.post<Envelope<TutorAskResponse>>('/tutor/ask', payload)
  return unwrap(res.data)
}

// ── Exam insights (Capability 3A gap layer) ───────────────────────

export async function getMyExamInsight(examId: string): Promise<ExamInsight> {
  const res = await apiClient.get<Envelope<ExamInsight>>(`/exams/${examId}/my-insight`)
  return unwrap(res.data)
}

export async function listExamInsights(examId: string): Promise<ExamInsightListEntry[]> {
  const res = await apiClient.get<Envelope<ExamInsightListEntry[]>>(`/exams/${examId}/insights`)
  return unwrap(res.data)
}

// ── Exam quality (Capability 4B) ──────────────────────────────────

export async function getExamQuality(examId: string): Promise<ExamQualityResponse> {
  const res = await apiClient.get<Envelope<ExamQualityResponse>>(`/exams/${examId}/quality`)
  return unwrap(res.data)
}

// ── Class heatmap (Capability 4A) ─────────────────────────────────

export async function getClassHeatmap(subjectCode: string, gradeLevel: number): Promise<ClassHeatmapResponse> {
  const res = await apiClient.get<Envelope<ClassHeatmapResponse>>('/teachers/me/class-heatmap', {
    params: { subjectCode, gradeLevel },
  })
  return unwrap(res.data)
}

// ── School quality (Capability 4C) ────────────────────────────────

export async function getSchoolQualityScores(schoolId: string): Promise<SchoolQualityResponse> {
  const res = await apiClient.get<Envelope<SchoolQualityResponse>>(`/schools/${schoolId}/quality-scores`)
  return unwrap(res.data)
}

// ── Ministry / regional oversight ─────────────────────────────────

export async function getMinistryOverview(): Promise<MinistryOverviewResponse> {
  const res = await apiClient.get<Envelope<MinistryOverviewResponse>>('/ministry/overview')
  return unwrap(res.data)
}

export async function getRegionStats(regionId: string): Promise<RegionStatsResponse> {
  const res = await apiClient.get<Envelope<RegionStatsResponse>>(`/ministry/regions/${regionId}/stats`)
  return unwrap(res.data)
}

export async function listRegions(): Promise<RegionResponse[]> {
  const res = await apiClient.get<Envelope<RegionResponse[]>>('/regions', { params: { limit: 100 } })
  return unwrap(res.data)
}

export async function listSchools(regionId?: string): Promise<SchoolResponse[]> {
  const res = await apiClient.get<Envelope<SchoolResponse[]>>('/schools', {
    params: { region_id: regionId, limit: 100 },
  })
  return unwrap(res.data)
}

export async function getSchool(schoolId: string): Promise<SchoolResponse> {
  const res = await apiClient.get<Envelope<SchoolResponse>>(`/schools/${schoolId}`)
  return unwrap(res.data)
}

export async function listTeachers(schoolId: string): Promise<TeacherResponse[]> {
  const res = await apiClient.get<Envelope<TeacherResponse[]>>('/teachers', {
    params: { school_id: schoolId, limit: 100 },
  })
  return unwrap(res.data)
}

// ── Notifications ──────────────────────────────────────────────────

export async function listNotifications(): Promise<NotificationResponse[]> {
  const res = await apiClient.get<Envelope<NotificationResponse[]>>('/notifications', {
    params: { limit: 50 },
  })
  return unwrap(res.data)
}

export async function markNotificationRead(id: string): Promise<void> {
  await apiClient.patch(`/notifications/${id}/read`)
}

export async function createNotification(payload: CreateNotificationRequest): Promise<NotificationResponse> {
  const res = await apiClient.post<Envelope<NotificationResponse>>('/notifications', payload)
  return unwrap(res.data)
}
