import { readFileSync } from 'node:fs'

import { expect, test } from '@playwright/test'

import { DEMO_PASSWORD, DEMO_USERS, SEED_OUTPUT_PATH, type SeededExam } from './fixtures'

const seeded: SeededExam = JSON.parse(readFileSync(SEED_OUTPUT_PATH, 'utf-8'))

test.describe('Student: taking an exam', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login')
    await page.getByLabel('Email').fill(DEMO_USERS.student.email)
    await page.getByLabel('Password').fill(DEMO_PASSWORD)
    await page.getByRole('button', { name: 'Sign in' }).click()
    await expect(page).toHaveURL('/')
  })

  test('a student can find, start, take, and submit the seeded published exam', async ({ page }) => {
    // Real UI flow: exam center list -> "Start Exam" -> instructions
    // page (confirmation gate) -> the exam itself. Previously this spec
    // referenced an "Exam link or ID" input that doesn't exist anywhere
    // in the current StudentExamCenterPage -- fixed to match the real
    // flow, and extended to cover the new server-authoritative session
    // (StartAttempt) this exam system now goes through.
    await page.goto('/student/exams')
    await expect(page.getByText(seeded.examTitle)).toBeVisible()
    await page
      .locator('div.divide-y > div', { hasText: seeded.examTitle })
      .getByRole('button', { name: 'Start Exam' })
      .click()

    await expect(page).toHaveURL(`/student/exams/${seeded.examId}/pre`)
    await expect(page.getByText('No Time Limit').or(page.getByText('Time Limit'))).toBeVisible()
    // The explicit confirmation gate -- StartAttempt must not fire from
    // merely viewing this page.
    const startButton = page.getByRole('button', { name: 'Start Exam' })
    await expect(startButton).toBeDisabled()
    await page.getByRole('checkbox').check()
    await expect(startButton).toBeEnabled()
    await startButton.click()

    await expect(page).toHaveURL(`/student/exams/${seeded.examId}`)
    // "No time limit" pill confirms the /start response's timer state
    // rendered, not a client-side fabrication.
    await expect(page.getByText('No time limit')).toBeVisible()
    await expect(page.getByText(seeded.examTitle).or(page.getByText('Which option is correct?'))).toBeVisible()

    // global-setup.ts's seeded question has no question_options row, so
    // TakeExamPage falls back to plain lettered A/B/C/D radios -- the
    // seeded answer_key.correctOption is "B".
    await page.getByRole('radio', { name: 'B', exact: true }).check()
    // Autosave status should reflect the answer being persisted before
    // submission, not just at the end.
    await expect(page.getByText(/Saving…|Saved/)).toBeVisible({ timeout: 5000 })

    await page.getByRole('button', { name: 'Submit exam' }).click()

    await expect(page.getByText('Submitted')).toBeVisible()
    // A single auto-graded MCQ finishes grading immediately (no
    // pending-grading banner) and the correct answer means a pass.
    await expect(page.getByText(/1 of 1 question\(s\) graded instantly/)).toBeVisible()
    await expect(page.getByText(/Passed/)).toBeVisible()
  })

  test('resuming (refreshing) an in-progress attempt preserves question order, not a new randomization', async ({
    page,
  }) => {
    await page.goto(`/student/exams/${seeded.examId}/pre`)
    await page.getByRole('checkbox').check()
    await page.getByRole('button', { name: 'Start Exam' }).click()
    await expect(page).toHaveURL(`/student/exams/${seeded.examId}`)
    await expect(page.getByRole('radio', { name: 'B', exact: true })).toBeVisible()

    const firstQuestionText = await page.getByText(/^1\./).first().textContent()

    await page.reload()

    await expect(page.getByRole('radio', { name: 'B', exact: true })).toBeVisible()
    const secondQuestionText = await page.getByText(/^1\./).first().textContent()
    expect(secondQuestionText).toBe(firstQuestionText)
  })

  test('submitting the same exam twice is idempotent, not double-recorded', async ({ page }) => {
    // A dedicated exam (attempt_limit=1, the production default) --
    // separate from the other specs' shared exam so this test's
    // attempt-limit assertion isn't affected by their own StartAttempt
    // calls.
    const examId = seeded.singleAttemptExamId
    await page.goto(`/student/exams/${examId}/pre`)
    await page.getByRole('checkbox').check()
    await page.getByRole('button', { name: 'Start Exam' }).click()
    await expect(page).toHaveURL(`/student/exams/${examId}`)

    await page.getByRole('radio', { name: 'B', exact: true }).check()
    await page.getByRole('button', { name: 'Submit exam' }).click()
    await expect(page.getByText('Submitted')).toBeVisible()

    // A second attempt to start/take the same exam must not be able to
    // submit again -- attempt_limit is 1 for this exam, and the attempt
    // is already frozen either way.
    await page.goto(`/student/exams/${examId}/pre`)
    await page.getByRole('checkbox').check()
    await page.getByRole('button', { name: 'Start Exam' }).click()
    await expect(page.getByText(/used all allowed attempts|Could not start this exam/i)).toBeVisible()
  })
})
