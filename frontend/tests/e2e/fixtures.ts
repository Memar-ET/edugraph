// Demo accounts seeded by backend/db/migrations/V015__seed_demo_data.sql
// -- every account uses this same password, and these ids/emails are
// fixed in that migration, not generated per-run.
export const DEMO_PASSWORD = 'password123'

export const DEMO_USERS = {
  teacher: { email: 'teacher@edugraph.et', userId: 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15' },
  student: { email: 'student@edugraph.et', userId: 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a16' },
  schoolAdmin: { email: 'school.admin@edugraph.et', userId: 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14' },
} as const

export const DEMO_SCHOOL_ID = 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01'

// Written by global-setup.ts, read by the specs -- the exam/question ids
// are generated fresh per test run (see global-setup.ts's own comment on
// why), so specs can't hardcode them the way they can the V015 accounts
// above.
export interface SeededExam {
  examId: string
  questionId: string
  examTitle: string
  /** A second, separate exam with attempt_limit=1 -- for the specific
   * test asserting that a second attempt is rejected once the limit is
   * used up. Kept separate from the primary exam (attempt_limit=2) so
   * that test doesn't collide with the other specs' own StartAttempt
   * calls against the same exam. */
  singleAttemptExamId: string
  singleAttemptQuestionId: string
}

export const SEED_OUTPUT_PATH = 'tests/e2e/.seeded-exam.json'
