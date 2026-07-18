import {
  Outlet,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
} from '@tanstack/react-router'

import { LoginPage } from '@features/auth'
import { JobReviewPage, UploadPage } from '@features/curriculum'
import {
  ExamReviewPage,
  ExamUploadPage,
  GradeExamPage,
  StudentExamListPage,
  TakeExamPage,
} from '@features/assessment'
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

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    const { accessToken, user } = useAuthStore.getState()
    if (!accessToken) throw redirect({ to: '/login' })
    const landing = landingPathFor(user?.role)
    if (landing !== '/') throw redirect({ to: landing })
  },
  component: () => (
    <div className="flex min-h-screen items-center justify-center text-gray-500">
      Signed in. This role has no dashboard in this build yet.
    </div>
  ),
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  beforeLoad: () => {
    const { accessToken, user } = useAuthStore.getState()
    if (accessToken) throw redirect({ to: landingPathFor(user?.role) })
  },
  component: LoginPage,
})

// Guards curriculum routes: Step 1/3 are restricted server-side to
// curriculum_officer/ministry_admin (see router.go RequireRole), so the
// frontend enforces the same boundary before ever calling the API.
function requireCurriculumAccess() {
  const { accessToken, user } = useAuthStore.getState()
  if (!accessToken) throw redirect({ to: '/login' })
  if (!canAccessCurriculumReview(user?.role)) throw redirect({ to: '/' })
}

// Guards teacher exam routes (Capabilities 2A/2B/2C Flow 2) -- matches
// RequireRole(roleTeacher, roleSchoolAdmin) on the Go side.
function requireTeacherAccess() {
  const { accessToken, user } = useAuthStore.getState()
  if (!accessToken) throw redirect({ to: '/login' })
  if (!canAccessTeacherDashboard(user?.role)) throw redirect({ to: '/' })
}

// Guards student exam routes (Capability 2C Flow 1) -- matches
// RequireRole(roleStudent) on the Go side.
function requireStudentAccess() {
  const { accessToken, user } = useAuthStore.getState()
  if (!accessToken) throw redirect({ to: '/login' })
  if (!canAccessStudentDashboard(user?.role)) throw redirect({ to: '/' })
}

const uploadRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/upload',
  beforeLoad: requireCurriculumAccess,
  component: UploadPage,
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

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  uploadRoute,
  jobReviewRoute,
  examUploadRoute,
  examReviewRoute,
  gradeExamRoute,
  studentExamListRoute,
  takeExamRoute,
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
