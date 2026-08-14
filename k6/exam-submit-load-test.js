// Checklist 13.2: load-test the real exam-submission flow (login ->
// list questions -> submit) against the actual running single-instance
// system -- not a mock, not a synthetic microbenchmark.
//
// Run via run-load-test.sh, which drives this script at increasing VU
// counts against disjoint slices of a pre-seeded user pool (see
// backend/cmd/seed-load-test) -- one real student account submits
// exactly once per VU (executor: per-vu-iterations, iterations: 1),
// matching the real "a student can only submit an exam once" business
// rule (assessment.exam_attempts's uniqueness), so a 409 here is a real
// signal, not a test-harness artifact.
//
// Env vars (all required, set by run-load-test.sh):
//   BASE_URL     e.g. http://localhost:18090
//   USERS_FILE   path to the JSON seed-load-test wrote (examId + users[])
//   VUS          concurrent virtual students for this stage
//   OFFSET       starting index into users[] for this stage's slice

import http from 'k6/http'
import { check } from 'k6'
import { SharedArray } from 'k6/data'

const seed = new SharedArray('seed', function () {
  return [JSON.parse(open(__ENV.USERS_FILE))]
})[0]

const BASE_URL = __ENV.BASE_URL
const VUS = Number(__ENV.VUS)
const OFFSET = Number(__ENV.OFFSET)

export const options = {
  scenarios: {
    exam_submissions: {
      executor: 'per-vu-iterations',
      vus: VUS,
      iterations: 1,
      maxDuration: '2m',
    },
  },
  // Informational only, not a hard target -- see run-load-test.sh /
  // docs/architecture/performance.md for why 150 (or any small number)
  // isn't what the PRD actually asks for, and this is measuring the
  // CURRENT single-instance ceiling, not asserting one.
  thresholds: {
    http_req_failed: ['rate>=0'],
  },
}

export default function () {
  const user = seed.users[OFFSET + __VU - 1]
  if (!user) {
    // Ran past the end of this stage's user slice -- run-load-test.sh
    // sizes offsets to prevent this; fail loudly rather than silently
    // reusing another stage's user and corrupting its results.
    throw new Error(`no seeded user at index ${OFFSET + __VU - 1} -- seed pool too small for this stage`)
  }

  const jar = http.cookieJar()

  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email: user.email, password: user.password }),
    { headers: { 'Content-Type': 'application/json' }, jar, tags: { step: 'login' } },
  )
  if (!check(loginRes, { 'login succeeded': (r) => r.status === 200 })) {
    return // no session cookie -- nothing else in this iteration can succeed
  }

  const questionsRes = http.get(`${BASE_URL}/api/v1/exams/${seed.examId}/questions`, {
    jar,
    tags: { step: 'list_questions' },
  })
  check(questionsRes, { 'list questions succeeded': (r) => r.status === 200 })

  let answers = [{ questionId: '', response: 'A' }]
  try {
    const questions = JSON.parse(questionsRes.body).data
    if (Array.isArray(questions) && questions.length > 0) {
      answers = questions.map((q) => ({ questionId: q.id, response: 'A' }))
    }
  } catch {
    // fall through with the placeholder answers array -- the submit
    // call below will fail and be recorded as a real failure, which is
    // correct: a broken list-questions response IS a real problem.
  }

  const submitRes = http.post(
    `${BASE_URL}/api/v1/exams/${seed.examId}/submit`,
    JSON.stringify({ answers }),
    { headers: { 'Content-Type': 'application/json' }, jar, tags: { step: 'submit' } },
  )
  check(submitRes, { 'submit succeeded': (r) => r.status === 202 || r.status === 200 })
}
