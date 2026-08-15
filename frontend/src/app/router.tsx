import {
  Outlet,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
} from '@tanstack/react-router'

import { LoginPage } from '@features/auth'
import {
  CurriculumDashboardPage,
  CurriculumGraphPage,
  CurriculumVersionsPage,
  JobReviewPage,
  MinistryCurriculumDetailPage,
  MinistryCurriculumPage,
  PrerequisitesPage,
  UploadPage,
} from '@features/curriculum'
import { ExplainPage, ModelGovernancePage } from '@features/modeling'
import {
  ExamQualityPage,
  ExamReviewPage,
  ExamUploadPage,
  GradeExamPage,
  StudentExamListPage,
  TakeExamPage,
} from '@features/assessment'
import { StudentDashboardPage, TutorPage } from '@features/student'
import { MisconceptionReviewPage, TeacherDashboardPage } from '@features/teacher'
import { SchoolAdminDashboardPage } from '@features/school-admin'
import { RegionalDashboardPage } from '@features/regional'
import { MinistryDashboardPage } from '@features/ministry'
import { CareerPathsPage } from '@features/career'
import {
  canAccessCurriculumReview,
  canAccessStudentDashboard,
  canAccessTeacherDashboard,
  landingPathFor,
  useAuthStore,
} from '@stores/auth.store'

const rootRoute = createRootRoute({
  component: () => <Outlet />,
  notFoundComponent: () => (
    <div className="flex min-h-screen items-center justify-center text-gray-500">Page not found.</div>
  ),
})

// Dispatches '/' to the signed-in user's role-specific dashboard. Every
// role except curriculum_officer lands here (see landingPathFor).
function DashboardRouter() {
  const role = useAuthStore((s) => s.user?.role)
  switch (role) {
    case 'student':
      return <StudentDashboardPage />
    case 'teacher':
      return <TeacherDashboardPage />
    case 'school_admin':
      return <SchoolAdminDashboardPage />
    case 'regional_admin':
      return <RegionalDashboardPage />
    case 'ministry_admin':
      return <MinistryDashboardPage />
    default:
      return (
        <div className="flex min-h-screen items-center justify-center text-gray-500">
          Signed in. This role has no dashboard in this build yet.
        </div>
      )
  }
}

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    const { isAuthenticated, user } = useAuthStore.getState()
    if (!isAuthenticated) throw redirect({ to: '/login' })
    const landing = landingPathFor(user?.role)
    if (landing !== '/') throw redirect({ to: landing })
  },
  component: DashboardRouter,
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  beforeLoad: () => {
    const { isAuthenticated, user } = useAuthStore.getState()
    if (isAuthenticated) throw redirect({ to: landingPathFor(user?.role) })
  },
  component: LoginPage,
})

// Guards curriculum routes: Step 1/3 are restricted server-side to
// curriculum_officer/ministry_admin (see router.go RequireRole), so the
// frontend enforces the same boundary before ever calling the API.
function requireCurriculumAccess() {
  const { isAuthenticated, user } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
  if (!canAccessCurriculumReview(user?.role)) throw redirect({ to: '/' })
}

// The EG-GCKT explain endpoint is role-open (every role can reach it) --
// server-side ownership (own record for a student, same-school for a
// teacher/school_admin) is enforced in Service.authorizeExplain, not by a
// route-level role allowlist. The frontend guard here only needs to
// check authentication.
function requireAuth() {
  const { isAuthenticated } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
}

// Guards teacher exam routes (Capabilities 2A/2B/2C Flow 2, 4A, 4B) --
// matches RequireRole(roleTeacher, roleSchoolAdmin) on the Go side.
function requireTeacherAccess() {
  const { isAuthenticated, user } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
  if (!canAccessTeacherDashboard(user?.role)) throw redirect({ to: '/' })
}

// Guards student exam + tutor routes (Capabilities 2C Flow 1, 3C) --
// matches RequireRole(roleStudent) on the Go side.
function requireStudentAccess() {
  const { isAuthenticated, user } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
  if (!canAccessStudentDashboard(user?.role)) throw redirect({ to: '/' })
}

// Routes with no server-side role gate beyond "authenticated" (e.g. career
// paths -- GET is open to any role, POST is further gated inside the page).
function requireAuth() {
  const { isAuthenticated } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
}

const curriculumDashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum',
  beforeLoad: requireCurriculumAccess,
  component: CurriculumDashboardPage,
})

const uploadRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/upload',
  beforeLoad: requireCurriculumAccess,
  component: UploadPage,
})

const prerequisitesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/prerequisites',
  beforeLoad: requireCurriculumAccess,
  component: PrerequisitesPage,
})

const curriculumVersionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/versions',
  beforeLoad: requireCurriculumAccess,
  component: CurriculumVersionsPage,
})

const curriculumGraphRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/subjects/$code/graph',
  beforeLoad: requireCurriculumAccess,
  component: CurriculumGraphPage,
})

// EG-GCKT Milestone 9: model-governance review queue. Reuses
// requireCurriculumAccess (curriculum_officer/ministry_admin) for
// page-level access, matching this page's own List endpoint's RBAC;
// Promote/Reject are ministry_admin-only server-side (router.go), same
// "different actions, different RBAC within one page" precedent as
// PrerequisitesPage's ministry_admin-only Resync panel.
const modelGovernanceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/model-governance',
  beforeLoad: requireCurriculumAccess,
  component: ModelGovernancePage,
})

// EG-GCKT Milestone 11 (spec section 18): five-part explanation viewer.
const explainRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/students/$studentId/topics/$topicId/explain',
  beforeLoad: requireAuth,
  component: ExplainPage,
})

const ministryCurriculumRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ministry/curriculum',
  beforeLoad: requireCurriculumAccess,
  component: MinistryCurriculumPage,
})

const ministryCurriculumDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ministry/curriculum/$code',
  beforeLoad: requireCurriculumAccess,
  component: MinistryCurriculumDetailPage,
})

const jobReviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/jobs/$jobId',
  beforeLoad: requireCurriculumAccess,
  component: JobReviewPage,
})

const examUploadRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/exams/upload',
  beforeLoad: requireTeacherAccess,
  component: ExamUploadPage,
})

const examReviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/exams/$examId',
  beforeLoad: requireTeacherAccess,
  component: ExamReviewPage,
})

const gradeExamRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/exams/$examId/grade',
  beforeLoad: requireTeacherAccess,
  component: GradeExamPage,
})

const examQualityRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/exams/$examId/quality',
  beforeLoad: requireTeacherAccess,
  component: ExamQualityPage,
})

// EG-GCKT Milestone 11: misconception review queue.
const misconceptionReviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/misconceptions',
  beforeLoad: requireTeacherAccess,
  component: MisconceptionReviewPage,
})

const studentExamListRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/student/exams',
  beforeLoad: requireStudentAccess,
  component: StudentExamListPage,
})

const takeExamRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/student/exams/$examId',
  beforeLoad: requireStudentAccess,
  component: TakeExamPage,
})

const tutorRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/student/tutor',
  beforeLoad: requireStudentAccess,
  component: TutorPage,
})

const careerPathsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/career/paths',
  beforeLoad: requireAuth,
  component: CareerPathsPage,
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  curriculumDashboardRoute,
  uploadRoute,
  prerequisitesRoute,
  curriculumVersionsRoute,
  curriculumGraphRoute,
  modelGovernanceRoute,
  explainRoute,
  ministryCurriculumRoute,
  ministryCurriculumDetailRoute,
  jobReviewRoute,
  examUploadRoute,
  examReviewRoute,
  gradeExamRoute,
  examQualityRoute,
  misconceptionReviewRoute,
  studentExamListRoute,
  takeExamRoute,
  tutorRoute,
  careerPathsRoute,
])

// eslint-disable-next-line react-refresh/only-export-components -- router instance, not a component; main.tsx needs it.
export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

export function AppRouter() {
  return <RouterProvider router={router} />
}
