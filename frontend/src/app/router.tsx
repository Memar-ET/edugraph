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
  CloManagementPage,
  ContentReviewPage,
  CurriculumDashboardPage,
  CurriculumGraphExplorerPage,
  CurriculumGraphPage,
  JobReviewPage,
  MinistryCurriculumDetailPage,
  PrerequisitesPage,
  PublishVersionsPage,
  QMatrixQualityPage,
  UploadPage,
} from '@features/curriculum'
import { ExplainPage, ModelGovernancePage } from '@features/modeling'
import {
  ExamQualityPage,
  ExamReviewPage,
  ExamUploadPage,
  GradeExamPage,
  PreExamPage,
  TakeExamPage,
} from '@features/assessment'
import {
  StudentAchievementsPage,
  StudentDashboardPage,
  StudentExamCenterPage,
  StudentMisconceptionJournalPage,
  StudentSkillMapPage,
  StudentSkillStatePage,
  StudyPlanPage,
  SubjectProfilePage,
  TutorPage,
} from '@features/student'
import {
  ClassAnalyticsPage,
  CurriculumCoveragePage,
  QuestionBankPage,
  TeacherAnnouncementsPage,
  TeacherDashboardPage,
  TeacherExamListPage,
  TeacherMisconceptionReviewPage,
  TeacherReportsPage,
  TeacherStudentDetailPage,
  TeacherStudentRosterPage,
} from '@features/teacher'
import {
  SchoolAdminDashboardPage,
  SchoolAnalyticsPage,
  SchoolAnnouncementsPage,
  SchoolApprovalsPage,
  SchoolClassesPage,
  SchoolQualityPage,
  SchoolReportsPage,
  StudentRosterPage,
  TeacherRosterPage,
} from '@features/school-admin'
import {
  RegionalAnalyticsPage,
  RegionalAnnouncementsPage,
  RegionalDashboardPage,
  RegionalQualityOversightPage,
  RegionalReportsPage,
  RegionalResourcesPage,
  RegionalSchoolsPage,
} from '@features/regional'
import {
  MinistryAnnouncementsPage,
  MinistryAuditLogPage,
  MinistryCurriculumGovernancePage,
  MinistryDataExportsPage,
  MinistryDashboardPage,
  MinistryQualityGovernancePage,
  MinistryRegionalOverviewPage,
  MinistryReportsPage,
  RegionManagementPage,
} from '@features/ministry'
import { CareerPathsPage } from '@features/career'
import { SettingsPage } from '@features/settings'


import {
  canAccessCurriculumReview,
  canAccessStudentDashboard,
  canAccessTeacherDashboard,
  landingPathFor,
  useAuthStore,
} from '@stores/auth.store'

// ── Root ─────────────────────────────────────────────────────────────

const rootRoute = createRootRoute({
  component: () => <Outlet />,
  notFoundComponent: () => (
    <div className="flex min-h-screen items-center justify-center text-sm text-gray-500">
      Page not found.
    </div>
  ),
})

// ── Role-based dashboard dispatcher ──────────────────────────────────

function DashboardRouter() {
  const role = useAuthStore((s) => s.user?.role)
  switch (role) {
    case 'student':       return <StudentDashboardPage />
    case 'teacher':       return <TeacherDashboardPage />
    case 'school_admin':  return <SchoolAdminDashboardPage />
    case 'regional_admin':return <RegionalDashboardPage />
    case 'ministry_admin':return <MinistryDashboardPage />
    default:
      return (
        <div className="flex min-h-screen items-center justify-center text-sm text-gray-500">
          Signed in. No dashboard configured for this role yet.
        </div>
      )
  }
}

// ── Auth guards ───────────────────────────────────────────────────────

function requireAuth() {
  const { isAuthenticated } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
}

function requireCurriculumAccess() {
  const { isAuthenticated, user } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
  if (!canAccessCurriculumReview(user?.role)) throw redirect({ to: '/' })
}

function requireTeacherAccess() {
  const { isAuthenticated, user } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
  if (!canAccessTeacherDashboard(user?.role)) throw redirect({ to: '/' })
}

function requireStudentAccess() {
  const { isAuthenticated, user } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
  if (!canAccessStudentDashboard(user?.role)) throw redirect({ to: '/' })
}

function requireSchoolAdmin() {
  const { isAuthenticated, user } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
  if (user?.role !== 'school_admin' && user?.role !== 'ministry_admin')
    throw redirect({ to: '/' })
}

function requireRegionalAdmin() {
  const { isAuthenticated, user } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
  if (user?.role !== 'regional_admin' && user?.role !== 'ministry_admin')
    throw redirect({ to: '/' })
}

function requireMinistryAdmin() {
  const { isAuthenticated, user } = useAuthStore.getState()
  if (!isAuthenticated) throw redirect({ to: '/login' })
  if (user?.role !== 'ministry_admin') throw redirect({ to: '/' })
}

// ── Index / login ─────────────────────────────────────────────────────

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

// ── Curriculum routes ─────────────────────────────────────────────────

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
  component: PublishVersionsPage,
})
const curriculumGraphRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/subjects/$code/graph',
  beforeLoad: requireCurriculumAccess,
  component: CurriculumGraphPage,
})
const qmatrixQualityRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/qmatrix-quality',
  beforeLoad: requireCurriculumAccess,
  component: QMatrixQualityPage,
})
const qmatrixQualityCodeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/subjects/$code/qmatrix-quality',
  beforeLoad: requireCurriculumAccess,
  component: QMatrixQualityPage,
})
const graphExplorerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/graph-explorer',
  beforeLoad: requireCurriculumAccess,
  component: CurriculumGraphExplorerPage,
})
const cloManagementRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/clo-management',
  beforeLoad: requireCurriculumAccess,
  component: CloManagementPage,
})
const contentReviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/curriculum/content-review',
  beforeLoad: requireCurriculumAccess,
  component: ContentReviewPage,
})
const modelGovernanceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/model-governance',
  beforeLoad: requireCurriculumAccess,
  component: ModelGovernancePage,
})
const explainRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/students/$studentId/topics/$topicId/explain',
  beforeLoad: requireAuth,
  component: ExplainPage,
})
const ministryCurriculumRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ministry/curriculum',
  beforeLoad: requireMinistryAdmin,
  component: MinistryCurriculumGovernancePage,
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

// ── Teacher routes ────────────────────────────────────────────────────

const teacherExamListRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/exams',
  beforeLoad: requireTeacherAccess,
  component: TeacherExamListPage,
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
const misconceptionReviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/misconceptions',
  beforeLoad: requireTeacherAccess,
  component: TeacherMisconceptionReviewPage,
})
const teacherStudentRosterRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/students',
  beforeLoad: requireTeacherAccess,
  component: TeacherStudentRosterPage,
})
const teacherStudentDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/students/$studentId',
  beforeLoad: requireTeacherAccess,
  component: TeacherStudentDetailPage,
})
const questionBankRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/question-bank',
  beforeLoad: requireTeacherAccess,
  component: QuestionBankPage,
})
const teacherAnnouncementsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/announcements',
  beforeLoad: requireTeacherAccess,
  component: TeacherAnnouncementsPage,
})
const classAnalyticsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/class-analytics',
  beforeLoad: requireTeacherAccess,
  component: ClassAnalyticsPage,
})
const curriculumCoverageRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/curriculum-coverage',
  beforeLoad: requireTeacherAccess,
  component: CurriculumCoveragePage,
})
const teacherReportsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/teacher/reports',
  beforeLoad: requireTeacherAccess,
  component: TeacherReportsPage,
})

// ── Student routes ────────────────────────────────────────────────────

const studentExamListRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/student/exams',
  beforeLoad: requireStudentAccess,
  component: StudentExamCenterPage,
})
const preExamRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/student/exams/$examId/pre',
  beforeLoad: requireStudentAccess,
  component: PreExamPage,
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
const studentSkillStateRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/students/me/skill-states',
  beforeLoad: requireStudentAccess,
  component: StudentSkillStatePage,
})
const studyPlanRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/student/study-plans',
  beforeLoad: requireStudentAccess,
  component: StudyPlanPage,
})
const subjectProfileRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/student/subject-profiles',
  beforeLoad: requireStudentAccess,
  component: SubjectProfilePage,
})
const skillMapRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/student/skill-map',
  beforeLoad: requireStudentAccess,
  component: StudentSkillMapPage,
})
const misconceptionJournalRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/student/misconception-journal',
  beforeLoad: requireStudentAccess,
  component: StudentMisconceptionJournalPage,
})
const achievementsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/student/achievements',
  beforeLoad: requireStudentAccess,
  component: StudentAchievementsPage,
})

// ── School admin routes ───────────────────────────────────────────────

const schoolQualityRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/school/quality',
  beforeLoad: requireSchoolAdmin,
  component: SchoolQualityPage,
})
const studentRosterRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/school/students',
  beforeLoad: requireSchoolAdmin,
  component: StudentRosterPage,
})
const teacherRosterRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/school/teachers',
  beforeLoad: requireSchoolAdmin,
  component: TeacherRosterPage,
})
const schoolReportsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/school/reports',
  beforeLoad: requireSchoolAdmin,
  component: SchoolReportsPage,
})
const classesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/school/classes',
  beforeLoad: requireSchoolAdmin,
  component: SchoolClassesPage,
})
const schoolAnalyticsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/school/analytics',
  beforeLoad: requireSchoolAdmin,
  component: SchoolAnalyticsPage,
})
const approvalsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/school/approvals',
  beforeLoad: requireSchoolAdmin,
  component: SchoolApprovalsPage,
})
const schoolAnnouncementsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/school/announcements',
  beforeLoad: requireSchoolAdmin,
  component: SchoolAnnouncementsPage,
})

// ── Regional admin routes ─────────────────────────────────────────────

const regionalSchoolsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/regional/schools',
  beforeLoad: requireRegionalAdmin,
  component: RegionalSchoolsPage,
})
const regionalAnalyticsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/regional/analytics',
  beforeLoad: requireRegionalAdmin,
  component: RegionalAnalyticsPage,
})
const qualityOversightRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/regional/quality-oversight',
  beforeLoad: requireRegionalAdmin,
  component: RegionalQualityOversightPage,
})
const resourceStatusRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/regional/resources',
  beforeLoad: requireRegionalAdmin,
  component: RegionalResourcesPage,
})
const regionalAnnouncementsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/regional/announcements',
  beforeLoad: requireRegionalAdmin,
  component: RegionalAnnouncementsPage,
})
const regionalReportsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/regional/reports',
  beforeLoad: requireRegionalAdmin,
  component: RegionalReportsPage,
})

// ── Ministry admin routes ─────────────────────────────────────────────

const ministryReportsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ministry/reports',
  beforeLoad: requireMinistryAdmin,
  component: MinistryReportsPage,
})
const regionManagementRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ministry/regions',
  beforeLoad: requireMinistryAdmin,
  component: RegionManagementPage,
})
const regionalOverviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ministry/regional-overview',
  beforeLoad: requireMinistryAdmin,
  component: MinistryRegionalOverviewPage,
})
const ministryAnnouncementsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ministry/announcements',
  beforeLoad: requireMinistryAdmin,
  component: MinistryAnnouncementsPage,
})
const qualityGovernanceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ministry/quality-governance',
  beforeLoad: requireMinistryAdmin,
  component: MinistryQualityGovernancePage,
})
const dataExportsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ministry/data-exports',
  beforeLoad: requireMinistryAdmin,
  component: MinistryDataExportsPage,
})

const auditLogRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ministry/audit-log',
  beforeLoad: requireMinistryAdmin,
  component: MinistryAuditLogPage,
})

// ── Shared routes ─────────────────────────────────────────────────────

const careerPathsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/career/paths',
  beforeLoad: requireAuth,
  component: CareerPathsPage,
})
const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  beforeLoad: requireAuth,
  component: SettingsPage,
})

// ── Route tree ────────────────────────────────────────────────────────

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,

  // curriculum
  curriculumDashboardRoute,
  uploadRoute,
  prerequisitesRoute,
  curriculumVersionsRoute,
  curriculumGraphRoute,
  qmatrixQualityRoute,
  qmatrixQualityCodeRoute,
  graphExplorerRoute,
  cloManagementRoute,
  contentReviewRoute,
  modelGovernanceRoute,
  explainRoute,
  ministryCurriculumRoute,
  ministryCurriculumDetailRoute,
  jobReviewRoute,

  // teacher (upload before $examId to avoid param capture)
  teacherExamListRoute,
  examUploadRoute,
  examReviewRoute,
  gradeExamRoute,
  examQualityRoute,
  misconceptionReviewRoute,
  teacherStudentRosterRoute,
  teacherStudentDetailRoute,
  questionBankRoute,
  teacherAnnouncementsRoute,
  classAnalyticsRoute,
  curriculumCoverageRoute,
  teacherReportsRoute,

  // student (pre before $examId to avoid param capture)
  studentExamListRoute,
  preExamRoute,
  takeExamRoute,
  tutorRoute,
  studentSkillStateRoute,
  studyPlanRoute,
  subjectProfileRoute,
  skillMapRoute,
  misconceptionJournalRoute,
  achievementsRoute,

  // school admin
  schoolQualityRoute,
  studentRosterRoute,
  teacherRosterRoute,
  schoolReportsRoute,
  classesRoute,
  schoolAnalyticsRoute,
  approvalsRoute,
  schoolAnnouncementsRoute,

  // regional admin
  regionalSchoolsRoute,
  regionalAnalyticsRoute,
  qualityOversightRoute,
  resourceStatusRoute,
  regionalAnnouncementsRoute,
  regionalReportsRoute,

  // ministry admin
  ministryReportsRoute,
  regionManagementRoute,
  regionalOverviewRoute,
  ministryAnnouncementsRoute,
  qualityGovernanceRoute,
  dataExportsRoute,
  auditLogRoute,

  // shared
  careerPathsRoute,
  settingsRoute,
])

// eslint-disable-next-line react-refresh/only-export-components -- router instance exported for RouterProvider in main.tsx.
export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

export function AppRouter() {
  return <RouterProvider router={router} />
}
