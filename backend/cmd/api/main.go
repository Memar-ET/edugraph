package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edugraph-ai/edugraph/internal/auth"
	"github.com/edugraph-ai/edugraph/internal/curriculum"
	"github.com/edugraph-ai/edugraph/internal/assessment"
	"github.com/edugraph-ai/edugraph/internal/student"
	"github.com/edugraph-ai/edugraph/internal/school"
	"github.com/edugraph-ai/edugraph/internal/ministry"
	"github.com/edugraph-ai/edugraph/internal/sync"
	"github.com/edugraph-ai/edugraph/pkg/config"
	"github.com/edugraph-ai/edugraph/pkg/database/postgres"
	"github.com/edugraph-ai/edugraph/pkg/database/neo4j"
	"github.com/edugraph-ai/edugraph/pkg/database/redis"
	"github.com/edugraph-ai/edugraph/pkg/logger"
	"github.com/edugraph-ai/edugraph/pkg/telemetry"
)

func main() {
	cfg := config.Load()

	// Telemetry (OpenTelemetry → Jaeger)
	tp, err := telemetry.InitTracer(cfg.OTEL)
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// Logger
	zap := logger.New(cfg.LogLevel)
	defer zap.Sync() //nolint:errcheck

	// Databases
	pgPool, err := postgres.NewPool(cfg.Postgres)
	if err != nil {
		zap.Fatal("postgres connect", logger.Error(err))
	}
	defer pgPool.Close()

	neo4jDriver, err := neo4j.NewDriver(cfg.Neo4j)
	if err != nil {
		zap.Fatal("neo4j connect", logger.Error(err))
	}
	defer neo4jDriver.Close(context.Background()) //nolint:errcheck

	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		zap.Fatal("redis connect", logger.Error(err))
	}
	defer redisClient.Close()

	// Wire dependencies (manual DI — no framework)
	authRepo     := auth.NewRepository(pgPool)
	authSvc      := auth.NewService(authRepo, cfg.JWT, redisClient)
	authHandler  := auth.NewHandler(authSvc)

	currRepo     := curriculum.NewRepository(pgPool, neo4jDriver)
	currSvc      := curriculum.NewService(currRepo)
	currHandler  := curriculum.NewHandler(currSvc)

	assessRepo   := assessment.NewRepository(pgPool, neo4jDriver)
	assessSvc    := assessment.NewService(assessRepo, redisClient)
	assessHandler := assessment.NewHandler(assessSvc)

	studentRepo  := student.NewRepository(pgPool, neo4jDriver)
	studentSvc   := student.NewService(studentRepo, redisClient)
	studentHandler := student.NewHandler(studentSvc)

	schoolRepo   := school.NewRepository(pgPool)
	schoolSvc    := school.NewService(schoolRepo, redisClient)
	schoolHandler := school.NewHandler(schoolSvc)

	ministryRepo := ministry.NewRepository(pgPool)
	ministrySvc  := ministry.NewService(ministryRepo, redisClient)
	ministryHandler := ministry.NewHandler(ministrySvc)

	syncRepo     := sync.NewRepository(pgPool)
	syncSvc      := sync.NewService(syncRepo, neo4jDriver, redisClient)
	syncHandler  := sync.NewHandler(syncSvc)

	// Router
	router := newRouter(cfg, zap, authSvc,
		authHandler, currHandler, assessHandler,
		studentHandler, schoolHandler, ministryHandler, syncHandler,
	)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		zap.Info("server starting", logger.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.Fatal("server error", logger.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zap.Error("shutdown error", logger.Error(err))
	}
}
