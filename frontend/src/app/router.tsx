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
import { canAccessCurriculumReview, useAuthStore } from '@stores/auth.store'

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
    if (canAccessCurriculumReview(user?.role)) throw redirect({ to: '/curriculum/upload' })
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
    if (accessToken) {
      throw redirect({ to: canAccessCurriculumReview(user?.role) ? '/curriculum/upload' : '/' })
    }
  },
  component: LoginPage,
})

// Guards both curriculum routes: Step 1/3 are restricted server-side to
// curriculum_officer/ministry_admin (see router.go RequireRole), so the
// frontend enforces the same boundary before ever calling the API.
function requireCurriculumAccess() {
  const { accessToken, user } = useAuthStore.getState()
  if (!accessToken) throw redirect({ to: '/login' })
  if (!canAccessCurriculumReview(user?.role)) throw redirect({ to: '/' })
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

const routeTree = rootRoute.addChildren([indexRoute, loginRoute, uploadRoute, jobReviewRoute])

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
