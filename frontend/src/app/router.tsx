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
import {
  ExamQualityPage,
  ExamReviewPage,
  ExamUploadPage,
  GradeExamPage,
  StudentExamListPage,
  TakeExamPage,
} from '@features/assessment'
import { StudentDashboardPage, TutorPage } from '@features/student'
import { TeacherDashboardPage } from '@features/teacher'
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
  ministryCurriculumRoute,
  ministryCurriculumDetailRoute,
  jobReviewRoute,
  examUploadRoute,
  examReviewRoute,
  gradeExamRoute,
  examQualityRoute,
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
