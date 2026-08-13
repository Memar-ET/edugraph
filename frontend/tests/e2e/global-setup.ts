import { randomUUID } from 'node:crypto'
import { writeFileSync } from 'node:fs'

import { Client } from 'pg'

import { DEMO_SCHOOL_ID, DEMO_USERS, SEED_OUTPUT_PATH, type SeededExam } from './fixtures'

// Seeds one real, published exam with one question directly via SQL
// rather than through the actual upload -> AI-parse -> validate ->
// publish pipeline: that pipeline depends on ai-service actually parsing
// a real file (non-deterministic timing, and not something this suite
// should also be responsible for verifying -- exam PARSING has its own
// coverage need, separate from what 12.3 asked for, which is the
// frontend flows: login, exam-TAKING, exam review). This mirrors the
// pattern this session used throughout for backend integration tests
// (internal/testutil) -- seed known-good data directly, exercise the
// real UI on top of it.
//
// Fresh examId/questionId every run (not fixed like the V015 demo
// users) so repeated local runs against a persistent dev database don't
// collide on old exam data from a previous run.
export default async function globalSetup(): Promise<void> {
  const client = new Client({
    host: process.env.POSTGRES_HOST || 'localhost',
    port: Number(process.env.POSTGRES_PORT) || 5432,
    database: process.env.POSTGRES_DB || 'edugraph',
    user: process.env.POSTGRES_USER || 'edugraph',
    password: process.env.POSTGRES_PASSWORD || 'edugraph',
    ssl: process.env.POSTGRES_SSLMODE === 'require' ? { rejectUnauthorized: false } : undefined,
  })
  await client.connect()

  try {
    const suffix = randomUUID().slice(0, 8)
    const subjectCode = `E2E${suffix}`
    const examId = randomUUID()
    const questionId = randomUUID()
    const examTitle = `E2E Test Exam ${suffix}`

    await client.query(
      `INSERT INTO curriculum.subjects (code, name_en, grade_level, academic_year) VALUES ($1, $2, 10, '2026')`,
      [subjectCode, 'E2E Test Subject'],
    )
    await client.query(
      `INSERT INTO assessment.exams
        (id, created_by, school_id, subject_code, grade_level, academic_year, exam_scope, title, total_marks, status)
       VALUES ($1, $2, $3, $4, 10, '2026', 'unit_test', $5, 5, 'published')`,
      [examId, DEMO_USERS.teacher.userId, DEMO_SCHOOL_ID, subjectCode, examTitle],
    )
    // No question_options row seeded (V021's table for rendered MCQ
    // choices) -- TakeExamPage.tsx falls back to plain lettered A/B/C/D
    // radios when q.options is empty, which is what this seed targets:
    // answer_key.correctOption must be a letter to match, not "4".
    await client.query(
      `INSERT INTO assessment.questions
        (id, exam_id, school_id, sequence_number, question_text, question_type, marks, answer_key)
       VALUES ($1, $2, $3, 1, 'Which option is correct? (select B for this test)', 'mcq', 5, $4)`,
      [questionId, examId, DEMO_SCHOOL_ID, JSON.stringify({ correctOption: 'B' })],
    )

    const seeded: SeededExam = { examId, questionId, examTitle }
    writeFileSync(SEED_OUTPUT_PATH, JSON.stringify(seeded, null, 2))
  } finally {
    await client.end()
  }
}
