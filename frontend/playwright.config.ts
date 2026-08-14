import { defineConfig, devices } from '@playwright/test'

// checklist 12.3: real E2E coverage for login, exam-taking, and exam
// review. baseURL matches vite.config.ts's dev-server port (5173) and
// the Vite proxy already routes /api/v1/* to the Go API -- these tests
// exercise the real frontend against a real running backend/Postgres/
// Neo4j/Redis, not a mocked API layer.
export default defineConfig({
  testDir: './tests/e2e',
  globalSetup: './tests/e2e/global-setup.ts',
  fullyParallel: false, // the three specs share seeded demo data (V015) -- avoid cross-test races
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:5173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
