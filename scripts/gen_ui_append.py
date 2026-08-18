"""Append sections 3-7 to gen_ui_spec.py, then import and run it to produce UI.docx."""
import importlib.util, sys, pathlib, textwrap

SCRIPT = pathlib.Path(r"D:\EDUGRAPH PROJECT\edugraph\scripts\gen_ui_spec.py")

APPEND = textwrap.dedent(r'''

# ══════════════════════════════════════════════════════════════════════════
# SECTION 3 — AUTH & ROUTING
# ══════════════════════════════════════════════════════════════════════════
h1("3.  Authentication & Routing")

h2("3.1  Auth Flow")
bullet("JWT RS256 — ", "15-min access token + 7-day HttpOnly refresh cookie. Auth interceptor silently retries a 401 once before surfacing an error.")
bullet("Login landing — ", "curriculum_officer goes to /curriculum (CurriculumDashboardPage); every other role goes to / (DashboardRouter).")
bullet("Logout — ", "POST /auth/logout revokes the server-side session cookie, then clearAuth() wipes Zustand state, then navigate to /login.")

h2("3.2  Route Guards")
tbl(
    ["Guard function", "Roles allowed", "Applied to"],
    [
        ["requireAuth()", "Any authenticated user", "/career/paths, /students/:id/topics/:id/explain, /state-snapshots"],
        ["requireStudentAccess()", "student", "All /student/* and /students/me/* routes"],
        ["requireTeacherAccess()", "teacher, school_admin", "All /teacher/* routes"],
        ["requireCurriculumAccess()", "curriculum_officer, ministry_admin", "All /curriculum/*, /model-governance, /ministry/curriculum"],
    ],
    widths=[2.2, 1.8, 3.0],
)

h2("3.3  Full Route Table")
tbl(
    ["Path", "Component", "Status"],
    [
        ["/login", "LoginPage", "Built"],
        ["/", "DashboardRouter (role dispatch)", "Built"],
        ["/student/exams", "StudentExamListPage", "Built"],
        ["/student/exams/available", "ExamAvailabilityPage", "Built"],
        ["/student/exams/:id/pre", "PreExamPage", "Built"],
        ["/student/exams/:id", "TakeExamPage (autosave + draft restore)", "Built"],
        ["/student/tutor", "TutorPage", "Built"],
        ["/students/me/skill-states", "StudentSkillStatePage", "Built"],
        ["/students/:id/topics/:topicId/explain", "ExplainPage", "Built"],
        ["/student/study-plans", "StudyPlanPage", "Planned"],
        ["/student/subject-profiles", "SubjectProfilePage", "Planned"],
        ["/student/settings", "StudentSettingsPage", "Planned"],
        ["/teacher/exams/upload", "ExamUploadPage", "Built"],
        ["/teacher/exams/:id", "ExamReviewPage", "Built"],
        ["/teacher/exams/:id/grade", "GradeExamPage", "Built"],
        ["/teacher/exams/:id/quality", "ExamQualityPage", "Built"],
        ["/teacher/misconceptions", "MisconceptionReviewPage", "Built"],
        ["/teacher/exams", "TeacherExamListPage", "Planned"],
        ["/teacher/students", "TeacherStudentRosterPage", "Planned"],
        ["/teacher/students/:id", "TeacherStudentDetailPage", "Planned"],
        ["/teacher/class-analytics", "ClassAnalyticsPage", "Planned"],
        ["/school/quality", "SchoolQualityPage", "Planned"],
        ["/school/teachers", "TeacherRosterPage", "Planned"],
        ["/school/students", "StudentRosterPage", "Planned"],
        ["/school/reports", "SchoolReportsPage", "Planned"],
        ["/regional/schools", "RegionalSchoolsPage", "Planned"],
        ["/regional/analytics", "RegionalAnalyticsPage", "Planned"],
        ["/ministry/curriculum", "MinistryCurriculumPage", "Built"],
        ["/ministry/curriculum/:code", "MinistryCurriculumDetailPage", "Built"],
        ["/ministry/reports", "MinistryReportsPage", "Planned"],
        ["/ministry/regions", "RegionManagementPage", "Planned"],
        ["/curriculum", "CurriculumDashboardPage", "Built"],
        ["/curriculum/upload", "UploadPage", "Built"],
        ["/curriculum/jobs/:jobId", "JobReviewPage", "Built"],
        ["/curriculum/prerequisites", "PrerequisitesPage", "Built"],
        ["/curriculum/versions", "CurriculumVersionsPage", "Built"],
        ["/curriculum/subjects/:code/graph", "CurriculumGraphPage", "Built"],
        ["/model-governance", "ModelGovernancePage", "Built"],
        ["/curriculum/subjects/:code/qmatrix-quality", "QMatrixQualityPage", "Planned"],
        ["/career/paths", "CareerPathsPage", "Built"],
        ["/settings", "SettingsPage", "Planned"],
    ],
    widths=[2.6, 2.2, 0.9],
    status_col=2,
)
doc.add_page_break()

# ══════════════════════════════════════════════════════════════════════════
# SECTION 4 — ROLE DEEP-DIVES
# ══════════════════════════════════════════════════════════════════════════
h1("4.  Role-Based Access — Feature Deep-Dives")

h2("4.1  Student")
para("Primary persona: K-12 learner preparing for, taking, and reviewing exams; "
     "receiving AI-generated personalised study support.", bold=True)
tbl(
    ["Feature", "Page", "Status", "API"],
    [
        ["Dashboard (subject health + study plan summary)", "StudentDashboardPage", "Built", "GET /students/me/subject-profiles"],
        ["Browse published exams", "ExamAvailabilityPage", "Built", "GET /exams"],
        ["Pre-exam briefing", "PreExamPage", "Built", "GET /exams/:id"],
        ["Take exam (autosave + draft restore + idempotency)", "TakeExamPage", "Built", "GET /exams/:id/questions, POST autosave/draft/submit"],
        ["Post-exam gap-analysis insight polling", "TakeExamPage (inline)", "Built", "GET /exams/:id/my-insight"],
        ["EG-GCKT knowledge map (skill states)", "StudentSkillStatePage", "Built", "GET /students/me/skill-states"],
        ["5-part topic explanation (EG-GCKT)", "ExplainPage", "Built", "GET /students/:id/topics/:topicId/explain"],
        ["Ask AI Tutor (Graph-RAG)", "TutorPage", "Built", "POST /tutor/ask"],
        ["Career path matches + generate", "CareerPathsPage", "Built", "GET + POST /students/me/career/*"],
        ["Active study plan (day-by-day blocks)", "StudyPlanPage", "Planned", "GET + POST /students/me/study-plans"],
        ["Subject health profiles (deep)", "SubjectProfilePage", "Planned", "GET /students/me/subject-profiles"],
        ["Account settings (language, password)", "SettingsPage", "Planned", "PATCH /auth/me"],
    ],
    widths=[2.1, 1.6, 0.9, 2.4],
    status_col=2,
)
bullet("Top missing: ", "StudyPlanPage, SubjectProfilePage, ExplainPage link-through from skill-state cards, SettingsPage.")

h2("4.2  Teacher")
para("Primary persona: classroom teacher managing exams, grading, reviewing AI quality "
     "and misconception alerts, monitoring class gaps.", bold=True)
tbl(
    ["Feature", "Page", "Status", "API"],
    [
        ["Class dashboard (heatmap + KPIs)", "TeacherDashboardPage", "Built", "GET /teachers/me/class-heatmap"],
        ["Heatmap topic click -> ExplainPage", "HeatmapGrid in dashboard", "Built", "GET /students/:id/topics/:topicId/explain"],
        ["Upload exam (PDF/DOCX)", "ExamUploadPage", "Built", "POST /exams/upload"],
        ["Review AI-parsed exam, edit, validate, publish, close", "ExamReviewPage", "Built", "Full exam lifecycle API"],
        ["Upload answer key", "ExamReviewPage", "Built", "POST /exams/:id/answer-key"],
        ["Bulk grade exam", "GradeExamPage", "Built", "POST /exams/:id/grades/bulk"],
        ["Per-student gap insights for an exam", "ExamReviewPage (insights tab)", "Built", "GET /exams/:id/insights"],
        ["Exam quality report", "ExamQualityPage", "Built", "GET /exams/:id/quality"],
        ["Print exam sheet + answer key", "ExamReviewPage (print actions)", "Built", "GET /exams/:id/print*"],
        ["Misconception review queue", "MisconceptionReviewPage", "Built", "GET/PATCH /misconceptions"],
        ["My exam list", "TeacherExamListPage", "Planned", "GET /exams"],
        ["Student roster for my class", "TeacherStudentRosterPage", "Planned", "GET /students"],
        ["Individual student detail + skill states", "TeacherStudentDetailPage", "Planned", "GET /students/:id + explain"],
        ["Class analytics (performance over time)", "ClassAnalyticsPage", "Planned", "GET /teachers/me/class-heatmap + GET /exams"],
    ],
    widths=[2.1, 1.7, 0.9, 2.3],
    status_col=2,
)
bullet("Top missing: ", "TeacherExamListPage (teachers have no list view of their own exams), student roster + detail, class analytics over time.")

doc.add_page_break()
h2("4.3  School Admin")
para("Primary persona: school principal with full teacher exam access plus "
     "school-wide quality, staff management, and student roster.", bold=True)
tbl(
    ["Feature", "Page", "Status", "API"],
    [
        ["School dashboard (quality score + heatmap)", "SchoolAdminDashboardPage", "Built", "GET /schools/:id/quality-scores"],
        ["All teacher exam capabilities (shared role)", "Same exam pages", "Built", "Same as teacher"],
        ["School quality score detail + breakdown", "SchoolQualityPage", "Planned", "GET /schools/:id/quality-scores"],
        ["Teacher roster (create, edit, deactivate)", "TeacherRosterPage", "Planned", "GET + POST + PATCH /teachers"],
        ["Student roster (create, edit, delete)", "StudentRosterPage", "Planned", "GET + POST + PATCH + DELETE /students"],
        ["Async reports (school_monthly)", "SchoolReportsPage", "Planned", "POST + GET /reports"],
        ["Notifications (flagged-for-review alerts)", "NotificationBell", "Built", "GET /notifications"],
    ],
    widths=[2.3, 1.7, 0.9, 2.1],
    status_col=2,
)
bullet("Top missing: ", "All management pages (staff roster, student roster, quality detail, reports page). Backend fully wired for all of them.")

h2("4.4  Regional Admin")
para("Primary persona: regional education bureau comparing school performance, "
     "flagging underperformers, creating schools and accounts.", bold=True)
tbl(
    ["Feature", "Page", "Status", "API"],
    [
        ["Regional dashboard (KPIs, underperforming schools)", "RegionalDashboardPage", "Built", "GET /ministry/regions/:id/stats + underperforming"],
        ["School comparison drill-down", "RegionalSchoolsPage", "Planned", "GET /schools + GET /schools/:id/quality-scores"],
        ["Regional analytics over time", "RegionalAnalyticsPage", "Planned", "GET /ministry/regions/:id/stats"],
        ["Create and manage schools", "RegionalSchoolsPage", "Planned", "POST + PATCH /schools"],
        ["Create staff accounts", "Within school pages", "Planned", "POST /teachers + /students"],
    ],
    widths=[2.3, 1.7, 0.9, 2.1],
    status_col=2,
)
bullet("Top missing: ", "RegionalSchoolsPage and RegionalAnalyticsPage. The entire management surface beyond the dashboard is absent.")

doc.add_page_break()
h2("4.5  Ministry Admin")
para("Primary persona: MoE official with full national authority — curriculum approval, "
     "model governance, national reports, region management.", bold=True)
tbl(
    ["Feature", "Page", "Status", "API"],
    [
        ["National overview dashboard", "MinistryDashboardPage", "Built", "GET /ministry/overview"],
        ["Curriculum browser (all subjects)", "MinistryCurriculumPage", "Built", "GET /curriculum/subjects"],
        ["Subject detail + version lineage", "MinistryCurriculumDetailPage", "Built", "GET /curriculum/subjects/:code/versions"],
        ["Upload + review + approve curriculum", "UploadPage, JobReviewPage", "Built", "Full curriculum pipeline"],
        ["Prerequisite graph + CLO resync", "PrerequisitesPage", "Built", "POST /curriculum/prerequisites/resync + /clos/resync"],
        ["Knowledge graph visualisation", "CurriculumGraphPage", "Built", "GET /curriculum/subjects/:code/graph"],
        ["Model governance (promote/reject snapshots)", "ModelGovernancePage", "Built", "GET/POST /model-snapshots"],
        ["Career paths management", "CareerPathsPage", "Built", "GET + POST /career/paths"],
        ["Q-matrix quality report", "QMatrixQualityPage", "Planned", "GET /curriculum/subjects/:code/qmatrix-quality"],
        ["Generate national reports (async)", "MinistryReportsPage", "Planned", "POST + GET /reports"],
        ["Region management (CRUD)", "RegionManagementPage", "Planned", "GET/POST/PATCH/DELETE /regions"],
        ["Underperforming schools + AI curriculum insights", "MinistryDashboardPage (partial)", "Partial", "GET /ministry/regions/:id/underperforming + POST /ministry/curriculum-insights"],
    ],
    widths=[2.3, 1.7, 0.9, 2.1],
    status_col=2,
)
bullet("Top missing: ", "MinistryReportsPage, RegionManagementPage, QMatrixQualityPage. Nav items for reports and regions still point to '/'.")

h2("4.6  Curriculum Officer")
para("Primary persona: MoE curriculum specialist uploading raw documents, reviewing "
     "AI-extracted structure, and maintaining the prerequisite/Q-matrix graph.", bold=True)
tbl(
    ["Feature", "Page", "Status", "API"],
    [
        ["Job list dashboard (own upload history)", "CurriculumDashboardPage", "Built", "GET /curriculum/jobs"],
        ["Upload curriculum PDF/DOCX", "UploadPage", "Built", "POST /curriculum/upload"],
        ["Review + edit + approve parsed structure", "JobReviewPage", "Built", "GET + POST /curriculum/jobs/:id"],
        ["View original uploaded file (proxied)", "JobReviewPage file viewer", "Built", "GET /storage/files/:jobId"],
        ["Knowledge graph visualisation per subject", "CurriculumGraphPage", "Built", "GET /curriculum/subjects/:code/graph"],
        ["Prerequisite editor (add, type, validate, history)", "PrerequisitesPage", "Built", "Full prerequisite API"],
        ["Version lineage + supersede", "CurriculumVersionsPage", "Built", "GET + POST version endpoints"],
        ["Q-matrix quality report", "QMatrixQualityPage", "Planned", "GET /curriculum/subjects/:code/qmatrix-quality"],
        ["Prerequisite quality report (orphaned topics)", "QMatrixQualityPage or PrerequisitesPage", "Planned", "GET /curriculum/subjects/:code/prerequisite-quality"],
        ["Model governance (candidate review)", "ModelGovernancePage", "Built", "GET /model-snapshots/candidates"],
        ["Parsing analytics (job success rate, timing)", "Planned page", "Planned", "GET /curriculum/jobs (aggregate)"],
    ],
    widths=[2.3, 1.7, 0.9, 2.1],
    status_col=2,
)
bullet("Top missing: ", "QMatrixQualityPage (two backend endpoints already exist), parsing analytics page, ExplainPage link-through from graph view.")

doc.add_page_break()

# ══════════════════════════════════════════════════════════════════════════
# SECTION 5 — NAV GAP AUDIT
# ══════════════════════════════════════════════════════════════════════════
h1("5.  Nav-Config Gap Audit")
para("Every nav item that currently points to '/' is a stub that needs wiring. "
     "This table is the complete punch-list extracted from nav-config.ts.")
tbl(
    ["Role", "Nav label", "Current 'to'", "Target route", "Priority"],
    [
        ["student", "Analytics", "/", "/student/subject-profiles", "High"],
        ["student", "Help & Support", "/", "/help", "Low"],
        ["student", "Settings", "/", "/settings", "Medium"],
        ["teacher", "Class Analytics", "/", "/teacher/class-analytics", "High"],
        ["teacher", "Help & Support", "/", "/help", "Low"],
        ["teacher", "Settings", "/", "/settings", "Medium"],
        ["school_admin", "Quality Score", "/", "/school/quality", "High"],
        ["school_admin", "Help & Support", "/", "/help", "Low"],
        ["school_admin", "Settings", "/", "/settings", "Medium"],
        ["regional_admin", "Regional Analytics", "/", "/regional/analytics", "High"],
        ["regional_admin", "Help & Support", "/", "/help", "Low"],
        ["regional_admin", "Settings", "/", "/settings", "Medium"],
        ["ministry_admin", "National Reports", "/", "/ministry/reports", "High"],
        ["ministry_admin", "Help & Support", "/", "/help", "Low"],
        ["ministry_admin", "Settings", "/", "/settings", "Medium"],
        ["curriculum_officer", "Parsing Analytics", "/", "/curriculum/analytics", "Medium"],
        ["curriculum_officer", "Help & Support", "/", "/help", "Low"],
        ["curriculum_officer", "Settings", "/", "/settings", "Medium"],
    ],
    widths=[1.4, 1.6, 0.9, 1.9, 0.9],
    status_col=4,
)
doc.add_page_break()

# ══════════════════════════════════════════════════════════════════════════
# SECTION 6 — I18N & ACCESSIBILITY
# ══════════════════════════════════════════════════════════════════════════
h1("6.  Internationalisation & Accessibility")

h2("6.1  Language Support")
para("The platform targets Ethiopian schools where the primary instruction language "
     "is Amharic (or regional Ethiosemitic/Cushitic languages), but all current UI "
     "strings are in English. The design system is ready for Amharic:")
bullet("Font: ", "IBM Plex Sans Ethiopic is loaded alongside IBM Plex Sans in the same font stack. Mixed EN/AM paragraphs render on the same baseline.")
bullet("Direction: ", "Amharic is left-to-right; no RTL layout work needed.")
bullet("i18n library needed: ", "react-i18next is the standard choice for React. Add an i18n/ directory with en.json and am.json message catalogs, wrap the app in I18nextProvider, and replace string literals with t('key') calls.")
bullet("Gap analysis / study plan text: ", "The ai-service already generates bilingual EN/AM narrative in gap_analysis/llm.py. The frontend just needs to display it; no translation layer required for AI-generated content.")
bullet("Exam content: ", "Question text and CLO descriptions can be in Amharic. The font renders them correctly; no other change needed.")

h2("6.2  Accessibility Checklist")
tbl(
    ["Requirement", "Current state", "Action needed"],
    [
        ["Colour contrast (4.5:1 AA minimum)", "primary-700 on gray-50 = 10.8:1. Alert colours designed for readability.", "Verify status-pill and badge colours with axe or contrast checker."],
        ["Keyboard navigation", "Tab order follows DOM order. Most interactive elements are native button/input.", "Add visible focus ring (ring-2 ring-primary-500) to any custom div-based interactive elements."],
        ["Screen reader labels", "Most pages use semantic HTML.", "Add aria-label to icon-only buttons (close X, notification bell). Add aria-live='polite' to async result areas (gap insight polling, autosave status)."],
        ["Form labels", "Input/Label components have htmlFor wired.", "Audit new planned forms (roster create, report generate) on build."],
        ["Motion", "prefers-reduced-motion applied globally in index.css.", "No action needed."],
        ["Language declaration", "html lang not yet set dynamically.", "Set lang='am' or lang='en' on <html> based on user language preference."],
        ["Error states", "Banner component exists.", "All async error paths should render a Banner with a descriptive message, not just a spinner that never resolves."],
    ],
    widths=[1.8, 2.2, 3.0],
    status_col=1,
)
doc.add_page_break()

# ══════════════════════════════════════════════════════════════════════════
# SECTION 7 — IMPLEMENTATION PRIORITIES
# ══════════════════════════════════════════════════════════════════════════
h1("7.  Implementation Priorities")
para("Ordered by user impact vs. implementation effort. "
     "All items marked 'Planned' have fully wired backend APIs — these are frontend-only builds.", bold=True)

h2("P0 — Blocking gaps (high impact, backend already done)")
tbl(
    ["Item", "Effort", "Unblocks"],
    [
        ["TeacherExamListPage (/teacher/exams) — teachers have no list of their own exams", "S", "Core teacher workflow"],
        ["Wire all stub nav items to their target routes (see Section 5)", "S", "All roles — nav items that go to '/' look broken"],
        ["StudyPlanPage — student day-by-day study schedule", "M", "Student retention loop; backend GET+POST fully wired"],
        ["ExplainPage link-through — button from StudentSkillStatePage cards", "S", "Makes EG-GCKT explain discoverable"],
        ["SubjectProfilePage — student subject health deep-dive", "S", "Backend GET /students/me/subject-profiles exists"],
        ["TeacherStudentRosterPage + TeacherStudentDetailPage", "M", "Teacher can drill from heatmap to individual student"],
    ],
    widths=[3.8, 0.6, 2.6],
)

h2("P1 — High value, straightforward builds")
tbl(
    ["Item", "Effort", "Notes"],
    [
        ["SchoolQualityPage — quality score breakdown for school_admin", "M", "Rich data already in /schools/:id/quality-scores"],
        ["StudentRosterPage + TeacherRosterPage for school_admin", "M", "Full CRUD API wired"],
        ["MinistryReportsPage — trigger + poll async reports", "M", "POST /reports/generate + poll GET /reports/:id (same pattern as gap insight polling)"],
        ["RegionManagementPage for ministry_admin", "M", "GET/POST/PATCH/DELETE /regions all wired"],
        ["QMatrixQualityPage — two tabs: Q-matrix quality + prerequisite quality", "M", "Two backend endpoints already exist"],
        ["ClassAnalyticsPage — teacher performance trend over time", "L", "Needs time-series from multiple exam results"],
        ["Modal / Dialog primitives", "S", "Needed for confirm-close exam, reject snapshot, misconception review forms"],
        ["Toast / Snackbar system (Sonner)", "S", "Autosave feedback exists in TakeExamPage; rest of app needs the same pattern"],
    ],
    widths=[2.8, 0.6, 3.6],
)

h2("P2 — Medium term")
tbl(
    ["Item", "Effort", "Notes"],
    [
        ["RegionalSchoolsPage + RegionalAnalyticsPage", "L", "Regional admin has essentially one page today"],
        ["SchoolReportsPage (school_admin async report trigger)", "S", "Same pattern as MinistryReportsPage"],
        ["SettingsPage (language toggle EN/AM, password change)", "S", "All roles share this page"],
        ["react-i18next setup + am.json catalog", "L", "Foundation for full Amharic UI"],
        ["FileDropzone component for upload pages", "S", "UX improvement on UploadPage and ExamUploadPage"],
        ["Combobox/autocomplete for topic picker", "M", "PrerequisitesPage topic picker breaks at 500+ topics with a plain Select"],
        ["DatePicker for exam scheduling", "S", "Needed when scheduled exam lifecycle state is surfaced"],
        ["Pagination component", "S", "Curriculum dashboard, student exam list, staff roster all need it"],
        ["SparklineMini chart for student subject cards", "S", "7-day mastery trend on StudentDashboardPage"],
        ["Tooltip component", "S", "CLO codes, edge types, quality score factors all need inline help text"],
        ["CurriculumGraphPage legend + filter controls", "M", "Graph is implemented but needs UX polish for large subjects"],
    ],
    widths=[2.8, 0.6, 3.6],
)

h2("P3 — Nice-to-have / post-launch")
tbl(
    ["Item", "Effort", "Notes"],
    [
        ["Dark mode", "XL", "Non-trivial given warm ledger palette — separate design track"],
        ["Offline mode UI (School Box)", "XL", "sync-agent is built; frontend needs an offline indicator + queue status"],
        ["Help & Support page", "M", "Static FAQ/documentation; low product urgency"],
        ["Print stylesheet for reports", "M", "National/school reports that print cleanly"],
        ["PWA / installable app", "L", "Useful for schools with limited connectivity; needs service worker"],
        ["RadarChart for CLO coverage on ExamQualityPage", "M", "Visual polish on an already-working page"],
    ],
    widths=[2.8, 0.6, 3.6],
)

h2("7.1  Missing Component Summary")
tbl(
    ["Component", "Needed by", "Priority"],
    [
        ["Modal / Dialog (Radix Dialog)", "Exam close confirm, reject snapshot, misconception forms", "P1"],
        ["Toast / Snackbar (Sonner)", "Autosave feedback, plan generation, career generate", "P1"],
        ["Tabs (Radix Tabs)", "ExamReviewPage (Questions | Quality | Insights), QMatrixQualityPage", "P1"],
        ["Combobox / Autocomplete", "PrerequisitesPage topic picker (500+ topics)", "P1"],
        ["FileDropzone", "UploadPage, ExamUploadPage", "P2"],
        ["DatePicker", "Exam scheduling", "P2"],
        ["Pagination", "Job list, exam list, rosters", "P2"],
        ["Tooltip", "CLO codes, edge types, quality factors", "P2"],
        ["SparklineMini", "Student dashboard subject cards", "P2"],
    ],
    widths=[1.8, 3.0, 0.8],
    status_col=2,
)
doc.add_page_break()

# ══════════════════════════════════════════════════════════════════════════
# SECTION 8 — PAGE SPECS FOR TOP-PRIORITY PLANNED PAGES
# ══════════════════════════════════════════════════════════════════════════
h1("8.  Page Specs — Top-Priority Planned Pages")
para("Each spec covers the route, guard, layout, key data sources, and main UI elements. "
     "These are build-ready; the APIs are confirmed live.", italic=True, color=LEDGER)

h2("8.1  StudyPlanPage  (/student/study-plans)")
bullet("Guard: ", "requireStudentAccess")
bullet("Data: ", "GET /students/me/study-plans (active plan), POST /students/me/study-plans (generate new)")
bullet("Layout: ", "AppShell. If no active plan: EmptyState with a 'Generate my study plan' button (calls POST, then polls until status=ready). If plan exists: accordion of days (Day 1, Day 2, ...) each containing topic cards.")
bullet("Topic card: ", "Topic title, estimated hours, action type badge (practice/diagnostic/prereq-review from EG-GCKT action ranking), mastery status indicator (at_risk/learning/mastered), 'Explain this topic' link to ExplainPage.")
bullet("Header actions: ", "Regenerate plan button (confirm dialog: this will replace the current plan).")
bullet("Empty prereq state: ", "If the student has no gap records yet, show a prompt to take an exam first.")

h2("8.2  TeacherExamListPage  (/teacher/exams)")
bullet("Guard: ", "requireTeacherAccess")
bullet("Data: ", "GET /exams (filtered to caller's school/grade)")
bullet("Layout: ", "AppShell. Stat bar (total exams, drafts, published, closed). Table with columns: Exam name, Subject/Grade, Status (StatusPill), Questions, Submissions, Created date, Actions.")
bullet("Actions per row: ", "View (->ExamReviewPage), Grade (->GradeExamPage), Quality (->ExamQualityPage), Close (confirm dialog).")
bullet("Empty state: ", "EmptyState with an 'Upload exam' CTA.")
bullet("Upload button: ", "Top-right, navigates to /teacher/exams/upload.")

h2("8.3  TeacherStudentRosterPage  (/teacher/students)")
bullet("Guard: ", "requireTeacherAccess")
bullet("Data: ", "GET /students (server-side scoped to caller's school)")
bullet("Layout: ", "AppShell. Search input + grade filter. Table: name, grade, last active, mastery trend (SparklineMini), actions.")
bullet("Row click: ", "Navigates to /teacher/students/:id (TeacherStudentDetailPage).")

h2("8.4  TeacherStudentDetailPage  (/teacher/students/:id)")
bullet("Guard: ", "requireTeacherAccess")
bullet("Data: ", "GET /students/:id, GET /students/:id/topics/:topicId/explain (lazy per-topic)")
bullet("Layout: ", "AppShell. Student profile header (name, grade, school). Tab strip: Skill States | Exam History | Misconceptions.")
bullet("Skill States tab: ", "Same card grid as StudentSkillStatePage but for the selected student. 'Explain' button on each card.")
bullet("Exam History tab: ", "List of this student's submissions with scores.")
bullet("Misconceptions tab: ", "Any confirmed misconception_hypotheses for this student.")

h2("8.5  MinistryReportsPage  (/ministry/reports)")
bullet("Guard: ", "requireCurriculumAccess (ministry_admin)")
bullet("Data: ", "POST /reports/generate (trigger), GET /reports/:id (poll status)")
bullet("Layout: ", "AppShell. Two-column: left sidebar selects report type (school_monthly / national_heatmap / clo_coverage), right panel shows params form + history list.")
bullet("Params form: ", "Varies by report type. school_monthly: school picker + month selector. national_heatmap: no params. clo_coverage: subject code.")
bullet("Status poll: ", "Same pattern as gap-analysis insight polling in TakeExamPage — poll every 5s until status=ready, then show a download/view button.")
bullet("History list: ", "Table of past reports (type, params, status, generated_at, requester).")

h2("8.6  QMatrixQualityPage  (/curriculum/subjects/:code/qmatrix-quality)")
bullet("Guard: ", "requireCurriculumAccess")
bullet("Data: ", "GET /curriculum/subjects/:code/qmatrix-quality, GET /curriculum/subjects/:code/prerequisite-quality")
bullet("Layout: ", "AppShell. Subject code in page header. Two tabs: Q-Matrix Issues | Prerequisite Issues.")
bullet("Q-Matrix Issues tab: ", "Table of questions with missing / low-confidence / ambiguous skill mappings. Each row has an 'Edit mappings' link -> /questions/:id/skill-mappings (a future page).")
bullet("Prerequisite Issues tab: ", "Table of orphaned topics (no edges) and low-confidence edges. 'View in prerequisites editor' link per row.")

doc.add_page_break()

# ══════════════════════════════════════════════════════════════════════════
# SECTION 9 — STATE MANAGEMENT
# ══════════════════════════════════════════════════════════════════════════
h1("9.  State Management")
tbl(
    ["Store", "File", "What it holds"],
    [
        ["auth.store", "src/stores/auth.store.ts", "user, isAuthenticated, tokens. clearAuth(), landingPathFor(), role guards. JWT interceptor reads from here."],
        ["ui.store", "src/stores/ui.store.ts", "Global UI flags (sidebar open, active theme, language preference)."],
        ["offline.store", "src/stores/offline.store.ts", "Offline/online status, pending sync queue length (School Box indicator)."],
        ["TanStack Query", "src/lib/query/", "All server state (exams, curriculum, skill states, notifications). Cached, deduplicated, background-refreshed. Do not duplicate in Zustand."],
    ],
    widths=[1.4, 2.2, 3.4],
)
bullet("Rule: ", "Server state lives in TanStack Query. Only global client-only state lives in Zustand. No component-local state for data that other components need.")
bullet("Query keys: ", "Defined in src/lib/query/keys.ts. Always use the factory pattern: examKeys.detail(id), not hardcoded strings.")
bullet("Mutations: ", "Use useMutation from TanStack Query. On success, invalidate the relevant query keys so the UI stays in sync without manual refetch calls.")

h1("10.  Summary")
para("The EduGraph AI frontend is substantially built — not a scaffold. "
     "The design system is complete (The Register palette, IBM Plex Sans + Ethiopic, warm ledger neutrals), "
     "the component library is solid, the AppShell is production-quality, and 29 pages "
     "across all 6 roles are implemented and type-checked.", bold=False)
doc.add_paragraph()
para("The real gap is that roughly 20 additional pages are needed to make every nav item "
     "functional and to serve the management workflows that school admins, regional admins, "
     "and ministry admins need. All of these pages have fully wired backend APIs. "
     "They are frontend-only builds — no new backend work is required.", italic=True, color=HEALTH)
doc.add_paragraph()
tbl(
    ["Category", "Built", "Planned"],
    [
        ["Pages (feature components)", "29", "~20"],
        ["UI primitive components", "11", "9"],
        ["Chart components", "6", "3"],
        ["Routes in router.tsx", "24", "~18"],
        ["Nav items wired to real routes", "~20", "~17 point to '/'"],
    ],
    widths=[2.5, 1.5, 1.5],
)
doc.add_paragraph()
para("End of UI specification.", italic=True, color=LEDGER)

doc.save(r"D:\EDUGRAPH PROJECT\edugraph\UI.docx")
print("UI.docx written OK")
''')

with open(SCRIPT, "a", encoding="utf-8") as f:
    f.write(APPEND)
print("Appended OK, running full script...")

# Now exec the whole combined script to produce UI.docx
exec(compile(SCRIPT.read_text(encoding="utf-8"), str(SCRIPT), "exec"))
