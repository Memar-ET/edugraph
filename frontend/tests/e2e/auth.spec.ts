import { expect, test } from '@playwright/test'

import { DEMO_PASSWORD, DEMO_USERS } from './fixtures'

async function login(page: import('@playwright/test').Page, email: string, password: string) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password').fill(password)
  await page.getByRole('button', { name: 'Sign in' }).click()
}

test.describe('Login', () => {
  test('a teacher lands on the teacher dashboard', async ({ page }) => {
    await login(page, DEMO_USERS.teacher.email, DEMO_PASSWORD)
    // landingPathFor(role) sends every role except curriculum_officer to
    // '/', which DashboardRouter then renders per-role -- assert on
    // content, not the URL alone, so this actually proves the right
    // dashboard rendered, not just that a redirect happened.
    await expect(page).toHaveURL('/')
    // Appears in both the sidebar and header -- .first() is enough to
    // prove the dashboard rendered with the right user, this isn't
    // asserting anything about which chrome element shows it.
    await expect(page.getByText('Dawit Lemma').first()).toBeVisible() // V015's teacher@edugraph.et full name
  })

  test('a student lands on the student dashboard', async ({ page }) => {
    await login(page, DEMO_USERS.student.email, DEMO_PASSWORD)
    await expect(page).toHaveURL('/')
    await expect(page.getByText('Hanna Solomon').first()).toBeVisible() // V015's student@edugraph.et full name
  })

  test('an incorrect password is rejected with an error, not a silent failure', async ({ page }) => {
    await login(page, DEMO_USERS.teacher.email, 'not-the-real-password')
    await expect(page).toHaveURL('/login')
    await expect(page.getByText(/invalid email or password/i)).toBeVisible()
  })

  test('logging out clears the session -- a protected route redirects back to /login', async ({ page }) => {
    await login(page, DEMO_USERS.student.email, DEMO_PASSWORD)
    await expect(page).toHaveURL('/')

    // checklist 11.1: logout must revoke the HttpOnly session cookie
    // server-side, not just clear local state -- see AppHeader.tsx/
    // AppShell.tsx's logout handler and endpoints.ts's logout().
    await page.getByRole('button', { name: /logout|sign out/i }).click()
    await expect(page).toHaveURL('/login')

    await page.goto('/student/exams')
    await expect(page).toHaveURL('/login')
  })
})
