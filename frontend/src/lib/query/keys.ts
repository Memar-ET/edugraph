export const queryKeys = {
  curriculumJob: (jobId: string) => ['curriculum', 'job', jobId] as const,
  exam: (examId: string) => ['assessment', 'exam', examId] as const,
  examQuestions: (examId: string) => ['assessment', 'exam', examId, 'questions'] as const,
  gradingQuestions: (examId: string) => ['assessment', 'exam', examId, 'grading-questions'] as const,
  students: (schoolId: string) => ['students', schoolId] as const,
}
