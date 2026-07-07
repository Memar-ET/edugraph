package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	storagepkg "github.com/edugraph-ai/edugraph/pkg/storage"

	assessmenthandler "github.com/edugraph-ai/edugraph/internal/assessment/handler"
	assessmentrepo "github.com/edugraph-ai/edugraph/internal/assessment/repository"
	assessmentsvc "github.com/edugraph-ai/edugraph/internal/assessment/service"

	authhandler "github.com/edugraph-ai/edugraph/internal/auth/handler"
	authrepo "github.com/edugraph-ai/edugraph/internal/auth/repository"
	authsvc "github.com/edugraph-ai/edugraph/internal/auth/service"

	careerhandler "github.com/edugraph-ai/edugraph/internal/career/handler"
	careerrepo "github.com/edugraph-ai/edugraph/internal/career/repository"
	careersvc "github.com/edugraph-ai/edugraph/internal/career/service"

	curriculumhandler "github.com/edugraph-ai/edugraph/internal/curriculum/handler"
	curriculumrepo "github.com/edugraph-ai/edugraph/internal/curriculum/repository"
	curriculumsvc "github.com/edugraph-ai/edugraph/internal/curriculum/service"

	jobshandler "github.com/edugraph-ai/edugraph/internal/jobs/handler"
	jobsrepo "github.com/edugraph-ai/edugraph/internal/jobs/repository"
	jobssvc "github.com/edugraph-ai/edugraph/internal/jobs/service"

	ministryhandler "github.com/edugraph-ai/edugraph/internal/ministry/handler"
	ministryrepo "github.com/edugraph-ai/edugraph/internal/ministry/repository"
	ministrysvc "github.com/edugraph-ai/edugraph/internal/ministry/service"

	notificationhandler "github.com/edugraph-ai/edugraph/internal/notification/handler"
	notificationrepo "github.com/edugraph-ai/edugraph/internal/notification/repository"
	notificationsvc "github.com/edugraph-ai/edugraph/internal/notification/service"

	regionhandler "github.com/edugraph-ai/edugraph/internal/region/handler"
	regionrepo "github.com/edugraph-ai/edugraph/internal/region/repository"
	regionsvc "github.com/edugraph-ai/edugraph/internal/region/service"

	schoolhandler "github.com/edugraph-ai/edugraph/internal/school/handler"
	schoolrepo "github.com/edugraph-ai/edugraph/internal/school/repository"
	schoolsvc "github.com/edugraph-ai/edugraph/internal/school/service"

	storagehandler "github.com/edugraph-ai/edugraph/internal/storage/handler"
	storagerepo "github.com/edugraph-ai/edugraph/internal/storage/repository"
	storagesvc "github.com/edugraph-ai/edugraph/internal/storage/service"

	studenthandler "github.com/edugraph-ai/edugraph/internal/student/handler"
	studentrepo "github.com/edugraph-ai/edugraph/internal/student/repository"
	studentsvc "github.com/edugraph-ai/edugraph/internal/student/service"

	synchandler "github.com/edugraph-ai/edugraph/internal/sync/handler"
	syncrepo "github.com/edugraph-ai/edugraph/internal/sync/repository"
	syncsvc "github.com/edugraph-ai/edugraph/internal/sync/service"

	teacherhandler "github.com/edugraph-ai/edugraph/internal/teacher/handler"
	teacherrepo "github.com/edugraph-ai/edugraph/internal/teacher/repository"
	teachersvc "github.com/edugraph-ai/edugraph/internal/teacher/service"

	"github.com/edugraph-ai/edugraph/pkg/ai"
	"github.com/edugraph-ai/edugraph/pkg/config"
	"github.com/edugraph-ai/edugraph/pkg/crypto"
	neo4jdb "github.com/edugraph-ai/edugraph/pkg/database/neo4j"
	"github.com/edugraph-ai/edugraph/pkg/database/postgres"
	"github.com/edugraph-ai/edugraph/pkg/database/redis"
	"github.com/edugraph-ai/edugraph/pkg/logger"
	"github.com/edugraph-ai/edugraph/pkg/telemetry"
)

func main() {
	cfg := config.Load()

	// Telemetry (OpenTelemetry → Jaeger)
	tp, err := telemetry.InitTracer(cfg.OTEL)
	if err != nil {
		os.Stderr.WriteString("init tracer: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// Logger
	log := logger.New(cfg.LogLevel)
	defer log.Sync() //nolint:errcheck

	// Databases
	pgPool, err := postgres.NewPool(cfg.Postgres)
	if err != nil {
		log.Fatal("postgres connect", logger.Error(err))
	}
	defer pgPool.Close()

	neo4jDriver, err := neo4jdb.NewDriver(cfg.Neo4j)
	if err != nil {
		log.Fatal("neo4j connect", logger.Error(err))
	}
	defer neo4jDriver.Close(context.Background()) //nolint:errcheck

	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		log.Fatal("redis connect", logger.Error(err))
	}
	defer redisClient.Close()

	// JWT signing keys — auto-generate a local dev keypair if missing so a
	// fresh checkout runs with zero manual setup. Production must mount real
	// keys at JWT_PRIVATE_KEY_PATH / JWT_PUBLIC_KEY_PATH.
	if cfg.AppEnv == "development" {
		if err := crypto.EnsureDevKeyPair(cfg.JWT.PrivateKeyPath, cfg.JWT.PublicKeyPath); err != nil {
			log.Fatal("generate dev jwt keypair", logger.Error(err))
		}
	}
	jwtSigner, err := crypto.NewJWTSigner(cfg.JWT.PrivateKeyPath, cfg.JWT.PublicKeyPath, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	if err != nil {
		log.Fatal("load jwt signer", logger.Error(err))
	}

	aiClient := ai.NewClient(cfg.AIServiceURL)

	storageRepo, err := storagerepo.New(context.Background(), cfg.AWS.Region)
	if err != nil {
		log.Fatal("init storage client", logger.Error(err))
	}

	// ── Wire dependencies (manual DI — no framework) ──────────
	authRepository := authrepo.New(pgPool)
	authService := authsvc.New(authRepository, jwtSigner, redisClient)
	authHandler := authhandler.New(authService)

	regionRepository := regionrepo.New(pgPool)
	regionService := regionsvc.New(regionRepository)
	regionHandler := regionhandler.New(regionService)

	schoolRepository := schoolrepo.New(pgPool)
	schoolService := schoolsvc.New(schoolRepository)
	schoolHandler := schoolhandler.New(schoolService)

	studentRepository := studentrepo.New(pgPool, neo4jDriver)
	studentService := studentsvc.New(studentRepository)
	studentHandler := studenthandler.New(studentService)

	teacherRepository := teacherrepo.New(pgPool)
	teacherService := teachersvc.New(teacherRepository)
	teacherHandler := teacherhandler.New(teacherService)

	ministryRepository := ministryrepo.New(pgPool)
	ministryService := ministrysvc.New(ministryRepository)
	ministryHandler := ministryhandler.New(ministryService)

	// ... existing code ...

	// 1. Initialize Storage Provider (Dev Mode: Postgres)
	// In Prod, you would swap this for storage.NewS3Storage(cfg.AWS)
	storageProvider := storagepkg.NewPostgresStorage(pgPool)

	// 2. Initialize Curriculum Domain
	curriculumRepo := curriculumrepo.New(pgPool)
	curriculumService := curriculumsvc.New(curriculumRepo, storageProvider, redisClient)
	curriculumHandler := curriculumhandler.New(curriculumService)

	// ... pass curriculumHandler to router ...

	assessmentRepository := assessmentrepo.New(pgPool, neo4jDriver)
	assessmentService := assessmentsvc.New(assessmentRepository)
	assessmentHandler := assessmenthandler.New(assessmentService)

	careerRepository := careerrepo.New(pgPool, neo4jDriver)
	careerService := careersvc.New(careerRepository, aiClient)
	careerHandler := careerhandler.New(careerService)

	syncRepository := syncrepo.New(pgPool)
	syncService := syncsvc.New(syncRepository)
	syncHandler := synchandler.New(syncService)

	notificationRepository := notificationrepo.New(pgPool)
	notificationService := notificationsvc.New(notificationRepository)
	notificationHandler := notificationhandler.New(notificationService)

	jobsRepository := jobsrepo.New(pgPool)
	jobsService := jobssvc.New(jobsRepository)
	jobsHandler := jobshandler.New(jobsService)

	storageService := storagesvc.New(storageRepo, cfg.AWS)
	storageHandler := storagehandler.New(storageService)

	// Router
	router := newRouter(cfg, log, authService, handlers{
		auth:         authHandler,
		region:       regionHandler,
		school:       schoolHandler,
		student:      studentHandler,
		teacher:      teacherHandler,
		ministry:     ministryHandler,
		curriculum:   curriculumHandler,
		assessment:   assessmentHandler,
		career:       careerHandler,
		sync:         syncHandler,
		notification: notificationHandler,
		jobs:         jobsHandler,
		storage:      storageHandler,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Info("server starting", logger.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", logger.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", logger.Error(err))
	}
}
