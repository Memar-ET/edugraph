import { apiClient } from './client'
import type {
  AddPrerequisiteRequest,
  AddPrerequisiteResponse,
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
  JobListItem,
  JobStatus,
  LoginRequest,
  PaginationMeta,
  MinistryOverviewResponse,
  NotificationResponse,
  PrerequisiteLink,
  PublishResponse,
  RegionResponse,
  RegionStatsResponse,
  ResyncPrerequisitesResponse,
  SchoolQualityResponse,
  SchoolResponse,
  StudentResponse,
  StudyPlan,
  SubjectGraph,
  SubjectListItem,
  SubjectProfile,
  SubjectVersion,
  SubmitExamRequest,
  SubmitExamResponse,
  TeacherResponse,
  TopicListItem,
  TutorAskRequest,
  TutorAskResponse,
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

export interface JobListResponse {
  items: JobListItem[]
  meta: PaginationMeta
}

/** Backs the Curriculum Officer dashboard (feature 1.2): the caller's own upload/approval history. */
export async function listCurriculumJobs(page = 1, limit = 20): Promise<JobListResponse> {
  const res = await apiClient.get<Envelope<JobListItem[]>>('/curriculum/jobs', { params: { page, limit } })
  if (!res.data.success || res.data.data === undefined) {
    throw new Error(res.data.error || 'Request failed')
  }
  return { items: res.data.data, meta: res.data.meta as PaginationMeta }
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

// ── Topic prerequisites ───────────────────────────────────────────

export async function listTopicsBySubject(subjectCode: string): Promise<TopicListItem[]> {
  const res = await apiClient.get<Envelope<TopicListItem[]>>(
    `/curriculum/subjects/${encodeURIComponent(subjectCode)}/topics`,
  )
  return unwrap(res.data)
}

export async function listTopicPrerequisites(topicId: string): Promise<PrerequisiteLink[]> {
  const res = await apiClient.get<Envelope<PrerequisiteLink[]>>(`/curriculum/topics/${topicId}/prerequisites`)
  return unwrap(res.data)
}

export async function addTopicPrerequisite(
  topicId: string,
  payload: AddPrerequisiteRequest,
): Promise<AddPrerequisiteResponse> {
  const res = await apiClient.post<Envelope<AddPrerequisiteResponse>>(
    `/curriculum/topics/${topicId}/prerequisites`,
    payload,
  )
  return unwrap(res.data)
}

export async function validatePrerequisite(topicId: string, prereqId: string): Promise<AddPrerequisiteResponse> {
  const res = await apiClient.patch<Envelope<AddPrerequisiteResponse>>(
    `/curriculum/topics/${topicId}/prerequisites/${prereqId}/validate`,
  )
  return unwrap(res.data)
}

export async function resyncPrerequisites(): Promise<ResyncPrerequisitesResponse> {
  const res = await apiClient.post<Envelope<ResyncPrerequisitesResponse>>('/curriculum/prerequisites/resync')
  return unwrap(res.data)
}

// ── Curriculum versioning ─────────────────────────────────────────

export async function getSubjectVersions(subjectCode: string): Promise<SubjectVersion[]> {
  const res = await apiClient.get<Envelope<SubjectVersion[]>>(
    `/curriculum/subjects/${encodeURIComponent(subjectCode)}/versions`,
  )
  return unwrap(res.data)
}

export async function supersedeSubject(newCode: string, previousCode: string): Promise<SubjectVersion> {
  const res = await apiClient.post<Envelope<SubjectVersion>>(
    `/curriculum/subjects/${encodeURIComponent(newCode)}/supersede`,
    { previousCode },
  )
  return unwrap(res.data)
}

export async function listSubjects(): Promise<SubjectListItem[]> {
  const res = await apiClient.get<Envelope<SubjectListItem[]>>('/curriculum/subjects')
  return unwrap(res.data)
}

// ── Curriculum knowledge graph ────────────────────────────────────

export async function getSubjectGraph(subjectCode: string, includeClos = false): Promise<SubjectGraph> {
  const res = await apiClient.get<Envelope<SubjectGraph>>(
    `/curriculum/subjects/${encodeURIComponent(subjectCode)}/graph`,
    { params: { includeClos: includeClos ? 'true' : undefined } },
  )
  return unwrap(res.data)
}
