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

// ── Ministry analytics (backend/internal/ministry/dto) ──────────

export interface WeakTopic {
  topic_id: string
  topic_title: string
  avg_mastery: number
  affected_students: number
}

export interface UnderperformingSchool {
  school_id: string
  school_name: string
  school_code: string
  quality_score: number
  mastery_rate: number
  flagged_topics_count: number
  top_weak_topics: WeakTopic[]
}

export interface UnderperformingResponse {
  schools: UnderperformingSchool[]
}

export interface InsightAlert {
  severity: 'critical' | 'warning' | 'info'
  message: string
}

export interface InsightRecommendation {
  priority: 'high' | 'medium' | 'low'
  text: string
}

export interface NationalInsightsResponse {
  headline: string
  alerts: InsightAlert[]
  recommendations: InsightRecommendation[]
  trend_summary: string
  ai_configured: boolean
}

/** Pagination metadata attached by middleware.WriteJSONMeta (pkg/pagination.Meta). */
export interface PaginationMeta {
  page: number
  limit: number
  total: number
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

// access_token/refresh_token are deliberately absent (checklist 11.1):
// the backend now sets them as HttpOnly cookies (see
// backend/pkg/middleware/middleware.go's SetAuthCookies) and never
// includes them in the JSON body -- there's nothing for this client to
// read.
export interface AuthResponse {
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
  /** A subtopic is itself a ParsedTopic, promoted as its own topics row (see migration V027's parent_topic_id). */
  subtopics?: ParsedTopic[]
  /** Opaque source-document ID (e.g. "G9.U2.T2.2"), review context only -- never promoted into Postgres. */
  externalCode?: string
}

export interface ParsedUnit {
  number: number
  titleEn: string
  topics: ParsedTopic[]
  /** Extra fields lifted from a unit's metadata table (subjectCode, focus,
   * indicativeCloCount, ...) that don't map to a curriculum.units column --
   * review context only, not promoted into Postgres. */
  metadata?: Record<string, string>
  /** CLOs stated at unit level rather than tied to one topic -- mapped to every topic/subtopic in the unit at approval time. */
  unitClos?: ParsedClo[]
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

/** One row of GET /curriculum/jobs -- the Curriculum Officer dashboard's job history. */
export interface JobListItem {
  jobId: string
  status: JobStatusValue
  fileName: string
  subjectCode: string
  gradeLevel: number
  academicYear: string
  createdAt: string
  approvedAt?: string
}

/** One row of GET /curriculum/subjects -- the Ministry curriculum browser (every promoted subject, not just one uploader's). */
export interface SubjectListItem {
  code: string
  nameEn: string
  nameAm?: string
  gradeLevel: number
  academicYear: string
  moeCode?: string
  isMandatory: boolean
  version: number
  isCurrent: boolean
  previousVersionCode?: string
  fileName?: string
  uploadedByName?: string
  approvedAt?: string
  createdAt: string
  unitCount: number
  topicCount: number
  subtopicCount: number
  cloCount: number
}

// ── Topic prerequisites (backend/internal/curriculum/dto/{prerequisites,topic_list}.go) ──

/** One row of GET /curriculum/subjects/{code}/topics -- backs the prerequisites UI's topic picker. */
export interface TopicListItem {
  id: string
  titleEn: string
  unitNumber: number
  sequenceOrder: number
  parentTopicId?: string
}

export type PrerequisiteInferMethod = 'explicit' | 'ai_inferred' | 'manual' | 'moe_document'

// EG-GCKT Milestone 0 (spec section 6.2): typed prerequisite edges.
// 'requires'/'strongly_requires' are hard dependencies (cycle-checked,
// walked by root-cause/study-plan/heatmap traversals); the rest are soft
// associations.
export type PrerequisiteEdgeType =
  | 'requires'
  | 'strongly_requires'
  | 'recommended_before'
  | 'related_to'
  | 'similar_to'
  | 'supports'
  | 'alternative_to'

export interface PrerequisiteLink {
  topicId: string
  topicTitle: string
  prerequisiteTopicId: string
  prerequisiteTitle: string
  prerequisiteGrade: number
  weight: number
  isCrossGrade: boolean
  inferMethod: PrerequisiteInferMethod
  isValidated: boolean
  edgeType: PrerequisiteEdgeType
  confidence?: number
  evidence?: string
  createdBy?: string
  createdByModel?: string
  curriculumVersion?: string
}

export interface AddPrerequisiteRequest {
  prerequisiteTopicId: string
  weight?: number
  inferMethod?: PrerequisiteInferMethod
  edgeType?: PrerequisiteEdgeType
  confidence?: number
  evidence?: string
}

export interface AddPrerequisiteResponse {
  link: PrerequisiteLink
  graphSynced: boolean
  graphError?: string
}

export interface ResyncPrerequisitesResponse {
  synced: number
  failed: number
}

export interface PrerequisiteReviewHistoryEntry {
  id: string
  action: string
  previousValues?: string
  newValues?: string
  reviewedBy?: string
  reviewedAt: string
  notes?: string
}

// ── Curriculum versioning (backend/internal/curriculum/dto/versions.go) ──

export interface SubjectVersion {
  code: string
  version: number
  isCurrent: boolean
  academicYear: string
  previousVersionCode?: string
  supersededAt?: string
  createdAt: string
}

// ── Curriculum knowledge graph (backend/internal/curriculum/dto/graph.go) ──

export type GraphNodeType = 'subject' | 'unit' | 'topic' | 'clo'

export interface GraphNode {
  id: string
  label: string
  type: GraphNodeType
  gradeLevel?: number
  isSubtopic?: boolean
  subjectCode?: string
  /** Belongs to a different subject than the one requested -- the other end of a cross-grade prerequisite edge. */
  external?: boolean
}

export type GraphEdgeType = 'HAS_UNIT' | 'HAS_TOPIC' | 'HAS_SUBTOPIC' | 'HAS_CLO' | 'HAS_PREREQUISITE'

export interface GraphEdge {
  source: string
  target: string
  type: GraphEdgeType
}

export interface SubjectGraph {
  nodes: GraphNode[]
  edges: GraphEdge[]
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
  unitNumbers: number[] | null
  academicYear: string
  totalMarks: number
  questionCount: number
  parseError?: string
  createdAt: string
  validationReport?: ValidationReport
}

// Capability 2D: correct a wrong subject/grade/exam-type/unit-range
// without re-uploading the exam file. Every field optional -- only send
// what needs to change.
export interface UpdateExamScopeRequest {
  subjectCode?: string
  gradeLevel?: number
  examScope?: ExamScope
  unitNumbers?: number[]
}

export interface UpdateExamScopeResponse {
  examId: string
  subjectCode: string
  gradeLevel: number
  examScope: ExamScope
  unitNumbers: number[] | null
  cloRematchQueued: boolean
  message: string
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

// ── Exam insights (Capability 3A gap layer) ─────────────────────

export interface GapRecordEntry {
  questionId: string | null
  symptomTopicId: string
  symptomTopicTitle: string
  cloCode?: string | null
  rootCauseTopicId?: string | null
  rootCauseTopicTitle?: string | null
  rootCauseGrade?: number | null
  severityScore: number
  prerequisiteDepth: number
  llmExplanation?: string | null
}

export interface ExamInsight {
  attemptId: string
  examId: string
  totalScore?: number
  percentage?: number
  passed?: boolean
  gapsFound: number
  summary?: string
  llmModel?: string
  generatedAt: string
  gaps: GapRecordEntry[]
}

export interface ExamInsightListEntry {
  studentId: string
  attemptId: string
  totalScore?: number
  percentage?: number
  passed?: boolean
  gapsFound: number
  summary?: string
  generatedAt: string
}

// ── Subject health (Capability 3A) ──────────────────────────────

export interface SubjectProfile {
  subjectCode: string
  subjectName: string
  gradeLevel: number
  currentMasteryPct: number
  /** Raw JSONB -- shape is generator-defined, render defensively. */
  topWeakAreas: unknown
  examsAnalyzed: number
  lastUpdated: string
}

// ── Study plans (Capability 3B) ─────────────────────────────────

export interface GenerateStudyPlanRequest {
  targetExamId?: string
  language?: 'en' | 'am'
}

export interface GenerateStudyPlanResponse {
  status: string
  message: string
}

export interface StudyPlanBlock {
  topicId?: string
  title?: string
  hours?: number
  why?: string
  isRootCause?: boolean
}

export interface StudyPlanDay {
  day: number | string
  blocks: StudyPlanBlock[]
}

export interface StudyPlanData {
  days: StudyPlanDay[]
  summary?: string
}

export interface StudyPlan {
  id: string
  targetExamId?: string | null
  planData: StudyPlanData
  totalDays: number
  totalHours: number
  language: string
  generatedAt: string
  expiresAt?: string | null
}

// ── AI Tutor (Capability 3C) ─────────────────────────────────────

export interface TutorAskRequest {
  question: string
  language?: 'en' | 'am'
}

export interface TutorAskResponse {
  answer: string
  relatedTopics: unknown
  usedGaps: unknown
  model: string
}

// ── Class heatmap (Capability 4A) ────────────────────────────────

export interface HeatmapAlert {
  topicId: string
  topicTitle: string
  strugglingPct: number
  rootCauseTopicId: string
  rootCauseTitle: string
  rootCauseGradeLevel: number
  rootCauseStudentsStruggling: number
  message: string
}

export interface HeatmapTopic {
  topicId: string
  title: string
  studentsStruggling: number
  strugglingPct: number
  avgSeverity: number
  alert?: HeatmapAlert
}

export interface ClassHeatmapResponse {
  subjectCode: string
  gradeLevel: number
  schoolId: string
  classSize: number
  source: string
  topics: HeatmapTopic[]
  alerts: HeatmapAlert[]
}

// ── Exam quality (Capability 4B) ─────────────────────────────────

export interface QuestionQuality {
  questionId: string
  sequenceNumber: number
  questionType: string
  cloCode?: string
  gradedAnswers: number
  discriminationIndex?: number
  discriminationMethod?: string
  statedDifficulty?: string
  actualDifficulty?: string
  avgScoreRatio: number
  calibration: 'unrated' | 'accurate' | 'mismatch' | string
  avgTimeSecs?: number
  timeAnomaly: boolean
}

export interface CLOFlag {
  cloCode: string
  description?: string
}

export interface CLOCoverageQuality {
  mandatoryTotal: number
  mandatoryTested: number
  zeroCorrect: CLOFlag[]
  untested: CLOFlag[]
}

export interface TimeAnalysisQuality {
  questionsWithTiming: number
  medianAvgSecs?: number
  evaluated: boolean
}

export interface ExamQualityResponse {
  examId: string
  title: string
  attemptsAnalyzed: number
  discriminationAvg?: number
  qualityScore: number
  questions: QuestionQuality[]
  cloCoverage: CLOCoverageQuality
  timeAnalysis: TimeAnalysisQuality
  computedAt: string
}

// ── School quality (Capability 4C) ───────────────────────────────

export interface SchoolQualityScore {
  subjectCode: string
  gradeLevel: number
  cloCoveragePct: number
  studentMasteryPct: number
  examQualityAvg: number
  curriculumCompliance: number
  compositeScore: number
  flaggedForReview: boolean
  computedAt: string
}

export interface SchoolQualityResponse {
  schoolId: string
  source: string
  scores: SchoolQualityScore[]
}

// ── Ministry / regions / schools / teachers ──────────────────────

export interface MinistryOverviewResponse {
  total_regions: number
  total_schools: number
  total_students: number
  total_teachers: number
}

export interface RegionStatsResponse {
  region_id: string
  school_count: number
  student_count: number
  teacher_count: number
  avg_assessment_score: number
}

export interface RegionResponse {
  id: string
  name: string
  code: string
  created_at: string
}

export interface SchoolResponse {
  id: string
  region_id: string
  name: string
  code: string
  address?: string
  created_at: string
}

export interface TeacherResponse {
  id: string
  user_id: string
  school_id: string
  subject_specialty?: string
  created_at: string
}

// ── Career ────────────────────────────────────────────────────────

export interface CareerPathResponse {
  id: string
  title: string
  description?: string
  required_subjects?: string[]
  created_at: string
}

export interface CreateCareerPathRequest {
  title: string
  description?: string
  required_subjects?: string[]
}

export interface CareerMatchResponse {
  career_path_id: string
  title: string
  score: number
}

// ── Notifications ────────────────────────────────────────────────

export interface CreateNotificationRequest {
  user_id: string
  title: string
  body: string
}

export interface NotificationResponse {
  id: string
  title: string
  body: string
  is_read: boolean
  created_at: string
}

// ── EG-GCKT model governance (backend/internal/modeling/dto/model_snapshots.go) ──
// Milestone 9: candidate BKT/DINA/IRT parameter refits produced nightly by
// ai-service's refit_worker.py, pending ministry_admin/curriculum_officer
// review before they can affect the live engines.

export type ModelSnapshotType =
  | 'prerequisite_graph'
  | 'qmatrix'
  | 'irt_calibration'
  | 'mirt_calibration'
  | 'dina_parameters'
  | 'gdina_parameters'
  | 'bkt_parameters'
  | 'dkt_model'
  | 'fusion_policy'
  | 'recommendation_policy'
  | 'student_state_snapshot'

export type ModelSnapshotStatus = 'candidate' | 'validated' | 'active' | 'rejected' | 'superseded'

export interface ModelSnapshot {
  id: string
  modelType: ModelSnapshotType
  version: number
  status: ModelSnapshotStatus
  scope?: string
  config: Record<string, unknown>
  trainingSummary?: Record<string, unknown>
  notes?: string
  createdAt: string
}

// ── EG-GCKT misconception modeling (backend/internal/assessment/dto/misconceptions.go) ──
// Milestone 6, spec section 11: structured, LLM-proposed misconception
// hypotheses pending teacher review.

export type MisconceptionStatus = 'candidate' | 'confirmed' | 'rejected'

export interface MisconceptionHypothesis {
  id: string
  studentId: string
  topicId: string
  topicTitle: string
  misconceptionText: string
  triggerPattern?: string
  confidence?: number
  status: MisconceptionStatus
  interventionText?: string
  generatedByModel?: string
  createdAt: string
}

// ── EG-GCKT explainability (backend/internal/modeling/dto/explain.go) ──
// Milestone 11, spec section 18: the five-part explanation for one
// (student, topic) pair.

export interface ExplanationCurrentState {
  masteryProbability?: number
  masteryStatus: string
  trend?: string
  evidenceCount: number
  forgettingRisk?: number
  lastSeen?: string
}

export interface ExplanationEvidenceItem {
  provenance: string
  estimate?: number
  uncertainty?: number
  reliability?: number
  createdAt: string
}

export interface PrerequisiteMasterySummary {
  topicId: string
  title: string
  edgeType: string
  masteryProbability?: number
}

export interface ExplanationStructuralContext {
  prerequisites: PrerequisiteMasterySummary[]
}

// ── EG-GCKT quality reports (backend/internal/curriculum/dto/quality_reports.go) ──

export interface QuestionMappingIssue {
  questionId: string
  questionText: string
  examId: string
  reason: string
  topicTitles?: string[]
}

export interface QMatrixQualityReport {
  subjectCode: string
  totalQuestions: number
  missingMappings: QuestionMappingIssue[]
  lowConfidenceMappings: QuestionMappingIssue[]
  ambiguousMappings: QuestionMappingIssue[]
}

export interface OrphanedTopic {
  topicId: string
  title: string
}

export interface LowConfidenceEdgeIssue {
  topicId: string
  topicTitle: string
  prerequisiteTopicId: string
  prerequisiteTitle: string
  edgeType: string
  confidence?: number
  isValidated: boolean
}

export interface PrerequisiteQualityReport {
  subjectCode: string
  totalTopics: number
  orphanedTopics: OrphanedTopic[]
  lowConfidenceEdges: LowConfidenceEdgeIssue[]
}

export interface SkillStateSnapshot {
  id: string
  masteryProbability?: number
  masteryStatus: string
  uncertainty?: number
  evidenceCount: number
  trend?: string
  snapshotReason: string
  sourceEventRangeStart?: string
  sourceEventRangeEnd?: string
  takenAt: string
}

export interface Explanation {
  studentId: string
  topicId: string
  topicTitle: string
  currentState: ExplanationCurrentState
  evidence: ExplanationEvidenceItem[]
  structuralContext: ExplanationStructuralContext
  confidence: string
  recommendation: string
  reason: string
}
