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
