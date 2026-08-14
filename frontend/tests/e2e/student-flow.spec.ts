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

  test('a student can find, take, and submit the seeded published exam', async ({ page }) => {
    await page.goto('/student/exams')
    await page.getByLabel('Exam link or ID').fill(seeded.examId)
    await page.getByRole('button', { name: 'Continue' }).click()

    await expect(page).toHaveURL(`/student/exams/${seeded.examId}`)
    await expect(page.getByText(seeded.examTitle).or(page.getByText('Which option is correct?'))).toBeVisible()

    // global-setup.ts's seeded question has no question_options row, so
    // TakeExamPage falls back to plain lettered A/B/C/D radios -- the
    // seeded answer_key.correctOption is "B".
    await page.getByRole('radio', { name: 'B', exact: true }).check()
    await page.getByRole('button', { name: 'Submit exam' }).click()

    await expect(page.getByText('Submitted')).toBeVisible()
    // A single auto-graded MCQ finishes grading immediately (no
    // pending-grading banner) and the correct answer means a pass.
    await expect(page.getByText(/1 of 1 question\(s\) graded instantly/)).toBeVisible()
    await expect(page.getByText(/Passed/)).toBeVisible()
  })

  test('submitting the same exam twice is rejected, not silently double-recorded', async ({ page }) => {
    await page.goto(`/student/exams/${seeded.examId}`)
    await page.getByRole('radio', { name: 'B', exact: true }).check()
    await page.getByRole('button', { name: 'Submit exam' }).click()
    await expect(page.getByText(/Submission failed|already submitted/i)).toBeVisible()
  })
})
