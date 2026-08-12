// Command sync-agent runs on a School Box and keeps its local Postgres in
// sync with the Central Cloud whenever connectivity is available: pushing
// the local sync.outbox (see V029__sync_outbox.sql) and pulling/applying
// changes back down. See internal/sync for the actual push/pull/merge
// logic — this file only handles config, wiring, and lifecycle.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edugraph-ai/edugraph-sync-agent/internal/health"
	"github.com/edugraph-ai/edugraph-sync-agent/internal/sync"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := loadConfig()
	if err != nil {
		log.Error("invalid config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, postgresDSN())
	if err != nil {
		log.Error("connect to local postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	healthState := health.NewState()
	agent := sync.NewAgent(cfg, pool, healthState, log)

	healthAddr := envOr("HEALTH_ADDR", ":9090")
	healthServer := health.NewServer(healthAddr, healthState)
	go func() {
		log.Info("health server listening", "addr", healthAddr)
		if err := healthServer.ListenAndServe(); err != nil {
			log.Error("health server stopped", "error", err)
		}
	}()

	log.Info("sync-agent started",
		"school_box_id", cfg.SchoolBoxID,
		"cloud_endpoint", cfg.CloudSyncEndpoint,
		"sync_interval", cfg.SyncInterval.String(),
	)
	agent.Run(ctx)

	log.Info("sync-agent shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = healthServer.Shutdown(shutdownCtx)
}

func loadConfig() (sync.Config, error) {
	endpoint := os.Getenv("CLOUD_SYNC_ENDPOINT")
	schoolBoxID := os.Getenv("SCHOOL_BOX_ID")
	schoolID := os.Getenv("SCHOOL_ID")
	if endpoint == "" || schoolBoxID == "" || schoolID == "" {
		return sync.Config{}, errRequired("CLOUD_SYNC_ENDPOINT, SCHOOL_BOX_ID, SCHOOL_ID are all required")
	}

	intervalMinutes, err := strconv.Atoi(envOr("SYNC_INTERVAL_MINUTES", "360"))
	if err != nil {
		return sync.Config{}, errRequired("SYNC_INTERVAL_MINUTES must be an integer")
	}
	batchSize, err := strconv.Atoi(envOr("SYNC_PUSH_BATCH_SIZE", "200"))
	if err != nil {
		return sync.Config{}, errRequired("SYNC_PUSH_BATCH_SIZE must be an integer")
	}

	return sync.Config{
		CloudSyncEndpoint: endpoint,
		SchoolBoxID:       schoolBoxID,
		SchoolID:          schoolID,
		SyncInterval:      time.Duration(intervalMinutes) * time.Minute,
		PushBatchSize:     batchSize,
		HTTPTimeout:       30 * time.Second,
	}, nil
}

func postgresDSN() string {
	host := envOr("POSTGRES_HOST", "postgres")
	port := envOr("POSTGRES_PORT", "5432")
	user := envOr("POSTGRES_USER", "edugraph")
	password := os.Getenv("POSTGRES_PASSWORD")
	db := envOr("POSTGRES_DB", "edugraph")
	return "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + db
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type configError string

func (e configError) Error() string { return string(e) }

func errRequired(msg string) error { return configError(msg) }
