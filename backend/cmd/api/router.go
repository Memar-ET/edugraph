package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	assessmenthandler "github.com/edugraph-ai/edugraph/internal/assessment/handler"
	authhandler "github.com/edugraph-ai/edugraph/internal/auth/handler"
	careerhandler "github.com/edugraph-ai/edugraph/internal/career/handler"
	curriculumhandler "github.com/edugraph-ai/edugraph/internal/curriculum/handler"
	jobshandler "github.com/edugraph-ai/edugraph/internal/jobs/handler"
	ministryhandler "github.com/edugraph-ai/edugraph/internal/ministry/handler"
	reportshandler "github.com/edugraph-ai/edugraph/internal/reports/handler"
	modelinghandler "github.com/edugraph-ai/edugraph/internal/modeling/handler"
	notificationhandler "github.com/edugraph-ai/edugraph/internal/notification/handler"
	regionhandler "github.com/edugraph-ai/edugraph/internal/region/handler"
	schoolhandler "github.com/edugraph-ai/edugraph/internal/school/handler"
	storagehandler "github.com/edugraph-ai/edugraph/internal/storage/handler"
	studenthandler "github.com/edugraph-ai/edugraph/internal/student/handler"
	synchandler "github.com/edugraph-ai/edugraph/internal/sync/handler"
	teacherhandler "github.com/edugraph-ai/edugraph/internal/teacher/handler"
	"github.com/edugraph-ai/edugraph/pkg/config"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// Role constants mirror the Postgres user_role enum (db/migrations/V001).
const (
	roleStudent           = "student"
	roleTeacher           = "teacher"
	roleSchoolAdmin       = "school_admin"
	roleRegionalAdmin     = "regional_admin"
	roleMinistryAdmin     = "ministry_admin"
	roleCurriculumOfficer = "curriculum_officer"
)

type handlers struct {
	auth         *authhandler.Handler
	region       *regionhandler.Handler
	school       *schoolhandler.Handler
	student      *studenthandler.Handler
	teacher      *teacherhandler.Handler
	ministry     *ministryhandler.Handler
	curriculum   *curriculumhandler.Handler
	assessment   *assessmenthandler.Handler
	career       *careerhandler.Handler
	sync         *synchandler.Handler
	notification *notificationhandler.Handler
	jobs         *jobshandler.Handler
	storage      *storagehandler.Handler
	modeling     *modelinghandler.Handler
	reports      *reportshandler.Handler
}

func newRouter(cfg config.Config, log *zap.Logger, verifier middleware.TokenVerifier, deviceVerifier synchandler.DeviceVerifier, h handlers) http.Handler {
	middleware.SetLogger(log)
	r := chi.NewRouter()

	// Outermost middleware — applied to every path including /health.
	r.Use(middleware.SecurityHeaders(cfg.AppEnv))
	r.Use(middleware.RequestID)
	r.Use(middleware.Recover(log))
	r.Use(middleware.Logging(log))
	r.Use(middleware.CORS(cfg.CORSOrigins))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	authenticated := middleware.Authenticate(verifier)

	// Rate limiters — token bucket, stdlib only, no external deps.
	// Auth endpoints: 10 req/s burst-20 per IP (brute-force protection).
	authLimiter := middleware.NewRateLimiter(10, 20, middleware.IPKey)
	// Authenticated API: 60 req/s burst-120 per authenticated user.
	apiLimiter := middleware.NewRateLimiter(60, 120, middleware.UserKey)

	r.Route("/api/v1", func(r chi.Router) {
		// ── Auth ──────────────────────────────────────────────
		r.Route("/auth", func(r chi.Router) {
			r.Use(authLimiter)
			r.Post("/register", h.auth.Register)
			r.Post("/login", h.auth.Login)
			r.Post("/refresh", h.auth.Refresh)
			r.Post("/logout", h.auth.Logout)
			r.With(authenticated).Get("/me", h.auth.Me)
		})

		// ── Sync (School Box devices, device-secret auth) ────
		// Not middleware.Authenticate -- a School Box is headless,
		// there's no human to hold a JWT. See
		// internal/sync/handler/device_auth.go for the scheme and why
		// this used to have no auth at all (checklist 10.1).
		r.Route("/sync", func(r chi.Router) {
			r.Use(synchandler.DeviceAuth(deviceVerifier))
			r.Post("/push", h.sync.Push)
			r.Get("/pull", h.sync.Pull)
		})

		// Everything below requires a valid access token.
		r.Group(func(r chi.Router) {
			r.Use(authenticated)
			r.Use(apiLimiter)

			// ── Regions ───────────────────────────────────────
			r.Route("/regions", func(r chi.Router) {
				r.Get("/", h.region.List)
				r.Get("/{id}", h.region.Get)
				r.With(middleware.RequireRole(roleMinistryAdmin)).Post("/", h.region.Create)
				r.With(middleware.RequireRole(roleMinistryAdmin)).Patch("/{id}", h.region.Update)
				r.With(middleware.RequireRole(roleMinistryAdmin)).Delete("/{id}", h.region.Delete)
			})

			// ── Schools ───────────────────────────────────────
			r.Route("/schools", func(r chi.Router) {
				r.Get("/", h.school.List)
				r.Get("/{id}", h.school.Get)
				r.With(middleware.RequireRole(roleMinistryAdmin, roleRegionalAdmin)).Post("/", h.school.Create)
				r.With(middleware.RequireRole(roleRegionalAdmin, roleSchoolAdmin)).Patch("/{id}", h.school.Update)
				r.With(middleware.RequireRole(roleMinistryAdmin, roleRegionalAdmin)).Delete("/{id}", h.school.Delete)
				// Capability 4C: composite quality scores per subject+grade.
				r.With(middleware.RequireRole(roleSchoolAdmin, roleRegionalAdmin, roleMinistryAdmin)).Get("/{id}/quality-scores", h.assessment.GetSchoolQualityScores)
			})

			// ── Students ──────────────────────────────────────
			r.Route("/students", func(r chi.Router) {
				// List/Get are further scoped server-side to the caller's
				// own school/region (student/service.go) -- the role gate
				// here just keeps a bare "student" account off the
				// roster entirely; there's no legitimate reason for one
				// to browse other students.
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin, roleRegionalAdmin, roleMinistryAdmin)).Get("/", h.student.List)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin, roleRegionalAdmin, roleMinistryAdmin)).Get("/{id}", h.student.Get)
				r.With(middleware.RequireRole(roleSchoolAdmin, roleTeacher)).Post("/", h.student.Create)
				r.With(middleware.RequireRole(roleSchoolAdmin, roleTeacher)).Patch("/{id}", h.student.Update)
				r.With(middleware.RequireRole(roleSchoolAdmin)).Delete("/{id}", h.student.Delete)

				// Own-student only, resolved server-side from the JWT
				// (career/handler.go) -- was previously /{studentID}/
				// career/generate|matches with the student id taken
				// straight from the URL, an IDOR any authenticated
				// account could use to read or trigger generation for
				// someone else's career matches.
				r.With(middleware.RequireRole(roleStudent)).Post("/me/career/generate", h.career.Generate)
				r.With(middleware.RequireRole(roleStudent)).Get("/me/career/matches", h.career.Matches)

				// Capability 3A: Subject Health Layer for the
				// authenticated student's own dashboard. Static "me"
				// takes precedence over the {id} param route in chi.
				r.With(middleware.RequireRole(roleStudent)).Get("/me/subject-profiles", h.assessment.GetMySubjectProfiles)

				// Capability 3B: study plans generated from gap records.
				r.With(middleware.RequireRole(roleStudent)).Post("/me/study-plans", h.assessment.GenerateStudyPlan)
				r.With(middleware.RequireRole(roleStudent)).Get("/me/study-plans", h.assessment.ListMyStudyPlans)

				// EG-GCKT skill-map view: every topic the caller has any
				// fused evidence for. Resolves the caller's own students.id
				// server-side (Service.MySkillStates) -- never existed as a
				// route before, so the frontend's skill-map page 404'd.
				r.With(middleware.RequireRole(roleStudent)).Get("/me/skill-states", h.modeling.GetMySkillStates)

				// Exam-taking entry point: every published exam matching
				// the caller's own school/grade -- never existed as a
				// route before, so StudentExamListPage/available-exams
				// 404'd for every student.
				r.With(middleware.RequireRole(roleStudent)).Get("/me/available-exams", h.assessment.GetAvailableExams)

				// EG-GCKT Milestone 11: five-part explanation (spec section
				// 18). {id} is caller-supplied -- Service.authorizeExplain
				// enforces ownership server-side (own record for a student,
				// same-school for a teacher/school_admin), not the role gate
				// alone, the same lesson checklist 11.3's career-matches fix
				// already established for this codebase.
				r.Get("/{id}/topics/{topicId}/explain", h.modeling.Explain)
				// EG-GCKT checklist sections 6/18/22: historical state
				// snapshots for comparison over time. Same authorization as
				// Explain.
				r.Get("/{id}/topics/{topicId}/state-snapshots", h.modeling.ListSkillStateSnapshots)
				// Teacher-facing counterpart to /me/skill-states -- backs
				// TeacherStudentDetailPage, which previously called
				// /me/skill-states (the caller's own state, not the
				// viewed student's) and failed for every teacher. Same
				// authorization as Explain.
				r.Get("/{id}/skill-states", h.modeling.GetStudentSkillStates)
			})

			// ── Teachers ──────────────────────────────────────
			r.Route("/teachers", func(r chi.Router) {
				// Capability 4A: class-wide gap heatmap + cross-grade
				// root-cause alerts, scoped to the caller's school.
				// Static "me" takes precedence over {id} in chi.
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/me/class-heatmap", h.assessment.GetClassHeatmap)
				// List/Get scoped server-side the same way as /students
				// above -- see student/service.go's Get/List for the
				// shared rationale.
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin, roleRegionalAdmin, roleMinistryAdmin)).Get("/", h.teacher.List)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin, roleRegionalAdmin, roleMinistryAdmin)).Get("/{id}", h.teacher.Get)
				r.With(middleware.RequireRole(roleSchoolAdmin)).Post("/", h.teacher.Create)
				r.With(middleware.RequireRole(roleSchoolAdmin)).Patch("/{id}", h.teacher.Update)
				r.With(middleware.RequireRole(roleSchoolAdmin)).Delete("/{id}", h.teacher.Delete)
			})

			// ── Ministry (national/regional oversight) ─────────
			r.Route("/ministry", func(r chi.Router) {
				r.Use(middleware.RequireRole(roleMinistryAdmin, roleRegionalAdmin))
				r.Get("/overview", h.ministry.Overview)
				r.Get("/regions/{regionID}/stats", h.ministry.RegionStats)
				// 6.1: ranked underperforming schools with weak-topic breakdown
				r.Get("/regions/{regionID}/underperforming", h.ministry.UnderperformingSchools)
				// 6.2: AI-generated national curriculum insights (degrades gracefully)
				r.With(middleware.RequireRole(roleMinistryAdmin)).Post("/curriculum-insights", h.ministry.CurriculumInsights)
			})

			// ── Curriculum ──────────────────────────────────────
			r.Route("/curriculum", func(r chi.Router) {
				r.With(middleware.RequireRole(roleCurriculumOfficer, roleMinistryAdmin)).Post("/upload", h.curriculum.Upload)
				r.With(middleware.RequireRole(roleCurriculumOfficer, roleMinistryAdmin)).Get("/jobs/{id}", h.curriculum.GetJob)
				r.With(middleware.RequireRole(roleCurriculumOfficer, roleMinistryAdmin)).Post("/jobs/{id}/approve", h.curriculum.Approve)
				r.With(middleware.RequireRole(roleMinistryAdmin, roleTeacher)).Post("/units", h.curriculum.CreateUnit)
				r.With(middleware.RequireRole(roleMinistryAdmin, roleTeacher)).Patch("/units/{id}", h.curriculum.UpdateUnit)
				r.With(middleware.RequireRole(roleMinistryAdmin)).Delete("/units/{id}", h.curriculum.DeleteUnit)
				// Topic prerequisite graph (feeds 3A root causes + 3B topo sort).
				r.With(middleware.RequireRole(roleMinistryAdmin, roleTeacher, roleCurriculumOfficer)).Post("/topics/{id}/prerequisites", h.curriculum.AddTopicPrerequisite)
				r.Get("/topics/{id}/prerequisites", h.curriculum.ListTopicPrerequisites)
				// Feature 1.4: confirm a link (typically one created "ai_inferred").
				r.With(middleware.RequireRole(roleMinistryAdmin, roleTeacher, roleCurriculumOfficer)).Patch("/topics/{id}/prerequisites/{prereqId}/validate", h.curriculum.ValidatePrerequisite)
				// EG-GCKT: append-only review history for one prerequisite edge.
				r.Get("/topics/{id}/prerequisites/{prereqId}/history", h.curriculum.PrerequisiteHistory)
				// Feature 1.5: bulk catch-up sync of the whole prerequisite graph into Neo4j.
				r.With(middleware.RequireRole(roleMinistryAdmin)).Post("/prerequisites/resync", h.curriculum.ResyncPrerequisites)
				// CUR-05: bulk resync of all (:Topic)-[:HAS_CLO]->(:CLO) edges into Neo4j.
				r.With(middleware.RequireRole(roleMinistryAdmin)).Post("/clos/resync", h.curriculum.ResyncCLOs)
				// Curriculum Officer dashboard: the officer's own upload/approval history.
				r.With(middleware.RequireRole(roleCurriculumOfficer, roleMinistryAdmin)).Get("/jobs", h.curriculum.ListJobs)
				// Mid-year revisions (feature 1.3): link an already-approved
				// subject code as the version that supersedes another.
				r.With(middleware.RequireRole(roleCurriculumOfficer, roleMinistryAdmin)).Post("/subjects/{code}/supersede", h.curriculum.Supersede)
				r.Get("/subjects/{code}/versions", h.curriculum.ListSubjectVersions)
				// Flat topic list for the prerequisites UI's topic picker --
				// no general "browse the curriculum" endpoint exists.
				r.Get("/subjects/{code}/topics", h.curriculum.ListTopicsBySubject)
				// Ministry curriculum browser, but also the data source for
				// the teacher-facing CurriculumCoveragePage (frontend
				// /teacher/curriculum-coverage, reachable by teacher and
				// school_admin per canAccessTeacherDashboard) -- both need
				// the subject list to pick a subject to check coverage for.
				r.With(middleware.RequireRole(roleMinistryAdmin, roleCurriculumOfficer, roleTeacher, roleSchoolAdmin)).Get("/subjects", h.curriculum.ListSubjects)
				// Neo4j knowledge-graph subtree for a subject -- backs the
				// frontend graph visualization.
				r.Get("/subjects/{code}/graph", h.curriculum.GetSubjectGraph)
				// EG-GCKT checklist sections 4/5/16: structural quality
				// reports (missing/low-confidence/ambiguous Q-matrix
				// mappings; orphaned topics and low-confidence prerequisite
				// edges), scoped to one subject.
				r.With(middleware.RequireRole(roleCurriculumOfficer, roleMinistryAdmin)).Get("/subjects/{code}/qmatrix-quality", h.curriculum.QMatrixQuality)
				r.With(middleware.RequireRole(roleCurriculumOfficer, roleMinistryAdmin)).Get("/subjects/{code}/prerequisite-quality", h.curriculum.PrerequisiteQuality)
			})

			// ── Exams (Capability 2A: upload + AI parsing; 2B: AI validation report; 2C: submission) ──
			r.Route("/exams", func(r chi.Router) {
				// School-scoped list (Repository.ListExamsBySchool) --
				// TeacherExamListPage/ClassAnalyticsPage/QuestionBankPage/
				// TeacherDashboardPage all call this; it never existed as
				// a route before, so every one of those pages 404'd.
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/", h.assessment.ListExams)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Post("/upload", h.assessment.UploadExam)
				// Student path (PreExamPage, the exam instructions/
				// confirmation screen) needs this too -- see
				// Service.GetExam's doc comment for the two different
				// authorization rules this now enforces per caller.
				// Teacher-only until this fix, which 403'd every student.
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin, roleStudent)).Get("/{id}", h.assessment.GetExam)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Post("/{id}/validate", h.assessment.ValidateExam)
				// Capability 2D: fix a wrong subject/grade/exam-type/unit-range
				// without re-uploading the file, then re-run /validate.
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Patch("/{id}/scope", h.assessment.UpdateExamScope)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Post("/{id}/publish", h.assessment.PublishExam)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Post("/{id}/close", h.assessment.CloseExam)
				// Production-hardening pass: creates (or resumes, if one is
				// already in_progress) the student's exam session --
				// server-authoritative timer/randomization/resume all
				// anchor off the attempt this creates. Idempotent while an
				// attempt is in_progress. Must run before /questions,
				// /autosave, /submit, all of which now require it to
				// already exist.
				r.With(middleware.RequireRole(roleStudent)).Post("/{id}/start", h.assessment.StartAttempt)
				r.With(middleware.RequireRole(roleStudent)).Post("/{id}/autosave", h.assessment.AutosaveExamDraft)
				r.With(middleware.RequireRole(roleStudent)).Get("/{id}/draft", h.assessment.GetExamDraft)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Post("/{id}/answer-key", h.assessment.UploadAnswerKey)
				r.With(middleware.RequireRole(roleStudent)).Get("/{id}/questions", h.assessment.ListExamQuestions)
				r.With(middleware.RequireRole(roleStudent)).Post("/{id}/submit", h.assessment.SubmitExam)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Post("/{id}/grades/bulk", h.assessment.BulkGradeExam)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/{id}/grading-questions", h.assessment.ListQuestionsForGrading)
				// Capability 3A: gap-analysis insight reads (written
				// asynchronously by the ai-service gap worker).
				r.With(middleware.RequireRole(roleStudent)).Get("/{id}/my-insight", h.assessment.GetMyExamInsight)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/{id}/insights", h.assessment.ListExamInsights)
				// Capability 4B: retroactive exam quality report
				// (discrimination, calibration, time anomalies, CLO coverage).
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/{id}/quality", h.assessment.GetExamQuality)
				// Exam-integrity signals (tab visibility, fullscreen,
				// connection status) -- students report, teachers read an
				// aggregate summary. Never framed as proof of misconduct.
				r.With(middleware.RequireRole(roleStudent)).Post("/{id}/attempts/current/events", h.assessment.ReportIntegrityEvents)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/{id}/integrity", h.assessment.GetExamIntegritySummary)
				// Capability 2.2: "Mode B" print-ready exam sheet + optical
				// answer key (PDF via the browser's own print-to-PDF, see
				// exam_print_templates.go). Both teacher/school_admin only --
				// the answer key must never be reachable by students, and the
				// blank exam sheet is teacher-initiated (print then hand out).
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/{id}/print", h.assessment.PrintExam)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/{id}/print/answer-key", h.assessment.PrintAnswerKey)
			})

			// ── EG-GCKT: versioned Q-matrix (spec section 6.3) ──
			// Lives on the curriculum handler (dto/repository/service all in
			// the curriculum package, mirroring prerequisites.go's shape)
			// even though it's mounted under /questions -- the Q-matrix maps
			// assessment items to curriculum skills, so it's curriculum
			// domain logic operating on an assessment.questions foreign key,
			// same relationship topic_clo_mappings already has.
			r.Route("/questions", func(r chi.Router) {
				r.With(middleware.RequireRole(roleMinistryAdmin, roleTeacher, roleCurriculumOfficer)).Post("/{id}/skill-mappings", h.curriculum.AddItemSkillMapping)
				r.Get("/{id}/skill-mappings", h.curriculum.ListItemSkillMappings)
				r.With(middleware.RequireRole(roleMinistryAdmin)).Post("/skill-mappings/resync", h.curriculum.ResyncItemSkillMappings)
			})

			// ── EG-GCKT: misconception review queue (spec section 11) ──
			r.Route("/misconceptions", func(r chi.Router) {
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/", h.assessment.ListCandidateMisconceptions)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Patch("/{id}/review", h.assessment.ReviewMisconception)
			})

			// ── EG-GCKT Milestone 9: model-governance review queue ──
			// (BKT/DINA/IRT nightly refit candidates, spec section 19).
			r.Route("/model-snapshots", func(r chi.Router) {
				r.With(middleware.RequireRole(roleMinistryAdmin, roleCurriculumOfficer)).Get("/candidates", h.modeling.ListCandidateSnapshots)
				r.With(middleware.RequireRole(roleMinistryAdmin)).Post("/{id}/promote", h.modeling.PromoteSnapshot)
				r.With(middleware.RequireRole(roleMinistryAdmin)).Post("/{id}/reject", h.modeling.RejectSnapshot)
			})

			// ── AI Tutor (Capability 3C: Graph-RAG + Gemini) ──
			r.Route("/tutor", func(r chi.Router) {
				r.With(middleware.RequireRole(roleStudent)).Post("/ask", h.assessment.AskTutor)
			})

			// ── Career ────────────────────────────────────────
			r.Route("/career", func(r chi.Router) {
				r.Get("/paths", h.career.List)
				r.With(middleware.RequireRole(roleMinistryAdmin)).Post("/paths", h.career.Create)
			})

			// ── Reports (Phase 12: async report generation) ──────
			// TeacherReportsPage, SchoolReportsPage, RegionalReportsPage,
			// MinistryReportsPage, and MinistryDataExportsPage all call
			// generate/list/get -- the role gate below was originally
			// scoped to only ministry_admin/school_admin, which 403'd the
			// teacher and regional_admin pages entirely.
			r.Route("/reports", func(r chi.Router) {
				r.With(middleware.RequireRole(roleMinistryAdmin, roleSchoolAdmin, roleTeacher, roleRegionalAdmin)).Post("/generate", h.reports.Generate)
				// List is scoped server-side to the caller's own requested
				// reports (Repository.ListByRequester); role gate kept
				// aligned with generate/get for consistency.
				r.With(middleware.RequireRole(roleMinistryAdmin, roleSchoolAdmin, roleTeacher, roleRegionalAdmin)).Get("/", h.reports.List)
				r.With(middleware.RequireRole(roleMinistryAdmin, roleSchoolAdmin, roleTeacher, roleRegionalAdmin)).Get("/{id}", h.reports.Get)
			})

			// ── Notifications ─────────────────────────────────
			r.Route("/notifications", func(r chi.Router) {
				r.Get("/", h.notification.List)
				r.Patch("/{id}/read", h.notification.MarkRead)
				r.With(middleware.RequireRole(roleSchoolAdmin, roleRegionalAdmin, roleMinistryAdmin)).Post("/", h.notification.Create)
			})

			// ── Jobs ────────────────────────────────────────────
			r.Route("/jobs", func(r chi.Router) {
				r.Post("/", h.jobs.Create)
				r.Get("/", h.jobs.List)
				r.Get("/{id}", h.jobs.Get)
				r.Patch("/{id}/status", h.jobs.UpdateStatus)
			})

			// ── Storage ──────────────────────────────────────
			r.Route("/storage", func(r chi.Router) {
				r.Post("/presign-upload", h.storage.PresignUpload)
				r.Post("/presign-download", h.storage.PresignDownload)
				// Dev-mode "Fork in the Road" for viewing an uploaded curriculum
				// file: streams it from the active StorageProvider (Postgres
				// BYTEA today; would be an S3 GetObject once S3StorageProvider
				// exists -- see internal/curriculum/service.DownloadFile).
				r.With(middleware.RequireRole(roleCurriculumOfficer, roleMinistryAdmin)).Get("/files/{jobId}", h.curriculum.DownloadFile)
			})
		})
	})

	return r
}
