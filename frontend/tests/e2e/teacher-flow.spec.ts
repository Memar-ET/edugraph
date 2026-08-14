import { readFileSync } from 'node:fs'

import { expect, test } from '@playwright/test'

import { DEMO_PASSWORD, DEMO_USERS, SEED_OUTPUT_PATH, type SeededExam } from './fixtures'

const seeded: SeededExam = JSON.parse(readFileSync(SEED_OUTPUT_PATH, 'utf-8'))

// Runs after student-flow.spec.ts (alphabetical file order, workers: 1 in
// playwright.config.ts -- see its own comment on why these specs aren't
// parallelized): reviews/grades the exam the student flow just
// submitted. Reusing the same seeded exam, not a fresh one, is
// deliberate -- exam review is meaningless without a real submission
// under it, and seeding a second exam+submission here would just
// duplicate global-setup.ts's job.
test.describe('Teacher: reviewing/grading a submitted exam', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login')
    await page.getByLabel('Email').fill(DEMO_USERS.teacher.email)
    await page.getByLabel('Password').fill(DEMO_PASSWORD)
    await page.getByRole('button', { name: 'Sign in' }).click()
    await expect(page).toHaveURL('/')
  })

  test('the grading spreadsheet shows the student who took the exam', async ({ page }) => {
    await page.goto(`/teacher/exams/${seeded.examId}/grade`)

    await expect(page.getByText(seeded.examTitle)).toBeVisible()
    // V015's student@edugraph.et admission number -- the row the
    // grading spreadsheet renders per student.
    await expect(page.getByText('STU-2026-001')).toBeVisible()
  })

  test('a teacher can enter/adjust a mark and save it', async ({ page }) => {
    await page.goto(`/teacher/exams/${seeded.examId}/grade`)

    const row = page.locator('tr', { hasText: 'STU-2026-001' })
    await expect(row).toBeVisible()

    // The single seeded question's grade cell -- adjust it to a
    // different value than the student's auto-graded submission, the
    // real-world "teacher overrides an auto-grade" case checklist
    // 10.3's answer_grade_history exists to track.
    const cell = row.locator('input')
    await cell.fill('C')

    await page.getByRole('button', { name: 'Save All' }).click()
    await expect(page.getByText(/Saved \d+ answer\(s\)/)).toBeVisible()
  })
})
