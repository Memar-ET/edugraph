export const queryKeys = {
  curriculumJob: (jobId: string) => ['curriculum', 'job', jobId] as const,
}
