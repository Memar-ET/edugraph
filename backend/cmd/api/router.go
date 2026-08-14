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
}

func newRouter(cfg config.Config, log *zap.Logger, verifier middleware.TokenVerifier, deviceVerifier synchandler.DeviceVerifier, h handlers) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recover(log))
	r.Use(middleware.Logging(log))
	r.Use(middleware.CORS(cfg.CORSOrigins))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	authenticated := middleware.Authenticate(verifier)

	r.Route("/api/v1", func(r chi.Router) {
		// ── Auth ──────────────────────────────────────────────
		r.Route("/auth", func(r chi.Router) {
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
				// Feature 1.5: bulk catch-up sync of the whole prerequisite graph into Neo4j.
				r.With(middleware.RequireRole(roleMinistryAdmin)).Post("/prerequisites/resync", h.curriculum.ResyncPrerequisites)
				// Curriculum Officer dashboard: the officer's own upload/approval history.
				r.With(middleware.RequireRole(roleCurriculumOfficer, roleMinistryAdmin)).Get("/jobs", h.curriculum.ListJobs)
				// Mid-year revisions (feature 1.3): link an already-approved
				// subject code as the version that supersedes another.
				r.With(middleware.RequireRole(roleCurriculumOfficer, roleMinistryAdmin)).Post("/subjects/{code}/supersede", h.curriculum.Supersede)
				r.Get("/subjects/{code}/versions", h.curriculum.ListSubjectVersions)
				// Flat topic list for the prerequisites UI's topic picker --
				// no general "browse the curriculum" endpoint exists.
				r.Get("/subjects/{code}/topics", h.curriculum.ListTopicsBySubject)
				// Ministry curriculum browser: every promoted subject system-wide.
				r.With(middleware.RequireRole(roleMinistryAdmin, roleCurriculumOfficer)).Get("/subjects", h.curriculum.ListSubjects)
				// Neo4j knowledge-graph subtree for a subject -- backs the
				// frontend graph visualization.
				r.Get("/subjects/{code}/graph", h.curriculum.GetSubjectGraph)
			})

			// ── Exams (Capability 2A: upload + AI parsing; 2B: AI validation report; 2C: submission) ──
			r.Route("/exams", func(r chi.Router) {
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Post("/upload", h.assessment.UploadExam)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/{id}", h.assessment.GetExam)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Post("/{id}/validate", h.assessment.ValidateExam)
				// Capability 2D: fix a wrong subject/grade/exam-type/unit-range
				// without re-uploading the file, then re-run /validate.
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Patch("/{id}/scope", h.assessment.UpdateExamScope)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Post("/{id}/publish", h.assessment.PublishExam)
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
				// Capability 2.2: "Mode B" print-ready exam sheet + optical
				// answer key (PDF via the browser's own print-to-PDF, see
				// exam_print_templates.go). Both teacher/school_admin only --
				// the answer key must never be reachable by students, and the
				// blank exam sheet is teacher-initiated (print then hand out).
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/{id}/print", h.assessment.PrintExam)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/{id}/print/answer-key", h.assessment.PrintAnswerKey)
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
