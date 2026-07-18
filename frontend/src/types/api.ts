// Mirrors the Go backend's JSON contracts exactly (see
// backend/internal/*/dto/dto.go and pkg/middleware.Envelope). Keep field
// names identical to the wire format -- these are not re-cased.

/** Every handler response is wrapped in this envelope by middleware.WriteJSON/WriteError. */
export interface Envelope<T> {
  success: boolean
  data?: T
  error?: string
  meta?: unknown
}

export type Role =
  | 'student'
  | 'teacher'
  | 'school_admin'
  | 'regional_admin'
  | 'ministry_admin'
  | 'curriculum_officer'

// ── Auth (backend/internal/auth/dto) ────────────────────────────

export interface LoginRequest {
  email: string
  password: string
}

export interface UserResponse {
  id: string
  email: string
  full_name: string
  role: Role
  phone?: string
  region_id?: string
  school_id?: string
  created_at: string
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: UserResponse
}

// ── Curriculum (backend/internal/curriculum/dto) ────────────────

export type JobStatusValue =
  | 'pending'
  | 'parsing'
  | 'parsed'
  | 'review'
  | 'approved'
  | 'rejected'
  | 'failed'

export interface UploadResponse {
  jobId: string
  status: JobStatusValue
  message: string
}

export interface ParsedClo {
  code: string
  description: string
  bloomLevel?: string
  mandatory: boolean
  keyConcept?: string
  evidence?: string
}

export interface ParsedTopic {
  sequenceOrder: number
  titleEn: string
  keyConcepts: string[]
  learningOutcomes: string[]
  clos: ParsedClo[]
  rawText?: string
}

export interface ParsedUnit {
  number: number
  titleEn: string
  topics: ParsedTopic[]
  /** Extra fields lifted from a unit's metadata table (subjectCode, focus,
   * indicativeCloCount, ...) that don't map to a curriculum.units column --
   * review context only, not promoted into Postgres. */
  metadata?: Record<string, string>
}

export interface ParsedStructurePayload {
  units: ParsedUnit[]
}

export interface JobStatus {
  jobId: string
  status: JobStatusValue
  fileName: string
  subjectCode: string
  gradeLevel: number
  academicYear: string
  parsedStructure?: ParsedStructurePayload
  approvedBy?: string
  approvedAt?: string
  error?: string
}

export interface ApproveRequest {
  parsedStructure?: ParsedStructurePayload
}

export interface ApproveResponse {
  jobId: string
  status: JobStatusValue
  subjectCode: string
  unitsPromoted: number
  topicsPromoted: number
  closPromoted: number
  graphSynced: boolean
  graphSyncError?: string
}

// ── Assessment (backend/internal/assessment/dto) ────────────────
// Capability 2A: upload + AI parsing; 2B: AI validation report;
// 2C: student submission (dual flow) + teacher bulk grading.

export type ExamStatusValue =
  | 'pending'
  | 'parsing'
  | 'draft'
  | 'validation_pending'
  | 'published'
  | 'closed'
  | 'failed'

export type ExamScope = 'unit_test' | 'midterm' | 'final_exam'

export interface UploadExamResponse {
  examId: string
  status: ExamStatusValue
  message: string
}

export interface CLOCoverageReport {
  totalMandatoryClos: number
  coveredMandatoryClos: number
  missingMandatoryClos: string[]
  totalClos: number
  coveredClos: number
  summary: string
}

export interface BloomBalanceReport {
  counts: Record<string, number>
  percentages: Record<string, number>
  unclassifiedQuestions: number
  higherOrderPercent: number
  minimumHigherOrderPercent: number
  meetsMinimumHigherOrder: boolean
  summary: string
}

export interface DifficultyDistributionReport {
  counts: Record<string, number>
  percentages: Record<string, number>
  unclassifiedQuestions: number
  hardPercent: number
  maxHardPercentAllowed: number
  exceedsMaxHard: boolean
  summary: string
}

export interface TopicCoverageEntry {
  topicTitle: string
  unitNumber: number
  questionCount: number
}

export interface PrerequisiteWarningEntry {
  topicTitle: string
  prerequisiteTitle: string
  prerequisiteGrade: number
  isCrossGrade: boolean
  message: string
}

export interface ValidationReport {
  generatedAt: string
  scope: string
  cloCoverage: CLOCoverageReport
  bloomBalance: BloomBalanceReport
  difficultyDistribution: DifficultyDistributionReport
  topicCoverage: TopicCoverageEntry[]
  prerequisiteWarnings: PrerequisiteWarningEntry[]
}

export interface ExamStatus {
  examId: string
  status: ExamStatusValue
  title: string
  subjectCode: string
  gradeLevel: number
  examScope: ExamScope
  academicYear: string
  totalMarks: number
  questionCount: number
  parseError?: string
  createdAt: string
  validationReport?: ValidationReport
}

export interface PublishResponse {
  examId: string
  status: ExamStatusValue
}

export interface UploadAnswerKeyResponse {
  examId: string
  message: string
}

/** One MCQ choice, split out of questionText during parsing so it renders
 * as a real labeled choice instead of raw prose. */
export interface QuestionOption {
  letter: string
  text: string
}

/** Question shape returned by GET /exams/:id/questions -- deliberately
 * has no answer_key/clo_code/clo_align_* fields, those aren't part of any
 * response the frontend surfaces to a student. */
export interface ExamQuestion {
  id: string
  sequenceNumber: number
  questionText: string
  questionType: 'mcq' | 'short_answer' | 'long_answer' | 'essay' | 'calculation'
  marks: number
  partLabel?: string
  options?: QuestionOption[]
}

/** Teacher-facing counterpart of ExamQuestion -- includes answerKey as
 * grading reference (not a leak; the person seeing this is the grader). */
export interface GradingQuestion {
  id: string
  sequenceNumber: number
  questionText: string
  questionType: 'mcq' | 'short_answer' | 'long_answer' | 'essay' | 'calculation'
  marks: number
  partLabel?: string
  answerKey?: Record<string, string>
  options?: QuestionOption[]
}

// ── Students (backend/internal/student/dto) ─────────────────────
// Only the fields the grading spreadsheet needs.

export interface StudentResponse {
  id: string
  user_id: string
  school_id: string
  admission_no: string
  grade_level: number
  created_at: string
}

export interface AnswerInput {
  questionId: string
  response: string
}

export interface SubmitExamRequest {
  answers: AnswerInput[]
}

export interface SubmitExamResponse {
  attemptId: string
  gradedCount: number
  pendingGradingCount: number
  totalScore?: number
  percentage?: number
  passed?: boolean
}

/** value is an MCQ option letter (e.g. "B") when the question is mcq, or
 * the teacher's already-graded numeric marks (e.g. "2") for everything
 * else -- see service.gradeTeacherEntry on the Go side. */
export interface GradeEntry {
  studentId: string
  questionId: string
  value: string
}

export interface BulkGradeRequest {
  entries: GradeEntry[]
}

export interface BulkGradeResponse {
  attemptsTouched: number
  answersSaved: number
}
