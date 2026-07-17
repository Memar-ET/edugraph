import { apiClient } from './client'
import type {
  ApproveRequest,
  ApproveResponse,
  AuthResponse,
  Envelope,
  JobStatus,
  LoginRequest,
  UploadResponse,
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
