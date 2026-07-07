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

func newRouter(cfg config.Config, log *zap.Logger, verifier middleware.TokenVerifier, h handlers) http.Handler {
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

		// ── Sync (School Box devices, no per-user auth) ──────
		r.Route("/sync", func(r chi.Router) {
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
			})

			// ── Students ──────────────────────────────────────
			r.Route("/students", func(r chi.Router) {
				r.Get("/", h.student.List)
				r.Get("/{id}", h.student.Get)
				r.With(middleware.RequireRole(roleSchoolAdmin, roleTeacher)).Post("/", h.student.Create)
				r.With(middleware.RequireRole(roleSchoolAdmin, roleTeacher)).Patch("/{id}", h.student.Update)
				r.With(middleware.RequireRole(roleSchoolAdmin)).Delete("/{id}", h.student.Delete)

				r.Get("/{studentID}/results", h.assessment.ResultsByStudent)
				r.Post("/{studentID}/career/generate", h.career.Generate)
				r.Get("/{studentID}/career/matches", h.career.Matches)
			})

			// ── Teachers ──────────────────────────────────────
			r.Route("/teachers", func(r chi.Router) {
				r.Get("/", h.teacher.List)
				r.Get("/{id}", h.teacher.Get)
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
				r.Get("/jobs/{id}", h.curriculum.GetJob)
				r.With(middleware.RequireRole(roleMinistryAdmin, roleTeacher)).Post("/units", h.curriculum.CreateUnit)
				r.With(middleware.RequireRole(roleMinistryAdmin, roleTeacher)).Patch("/units/{id}", h.curriculum.UpdateUnit)
				r.With(middleware.RequireRole(roleMinistryAdmin)).Delete("/units/{id}", h.curriculum.DeleteUnit)
				r.With(middleware.RequireRole(roleMinistryAdmin, roleTeacher)).Post("/units/{id}/prerequisites", h.curriculum.AddPrerequisite)
			})

			// ── Assessments ─────────────────────────────────────
			r.Route("/assessments", func(r chi.Router) {
				r.Get("/", h.assessment.List)
				r.Get("/{id}", h.assessment.Get)
				r.Get("/{id}/questions", h.assessment.ListQuestions)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Post("/", h.assessment.Create)
				r.With(middleware.RequireRole(roleTeacher)).Post("/{id}/questions", h.assessment.AddQuestion)
				r.With(middleware.RequireRole(roleStudent)).Post("/{id}/submit", h.assessment.Submit)
				r.With(middleware.RequireRole(roleTeacher, roleSchoolAdmin)).Get("/{id}/results", h.assessment.ResultsByAssessment)
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
			})
		})
	})

	return r
}
