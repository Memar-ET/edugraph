// sync-agent is the School Box bidirectional sync daemon.
// It runs on a Raspberry Pi / mini-PC at each school, continuously:
//   - pushing locally-written rows from sync.outbox to the cloud API
//   - pulling cloud-authoritative changes (curriculum, model snapshots,
//     published exams) and applying them to the local School Box Postgres.
//
// Auth: X-Device-Id + X-Device-Secret headers (see
// internal/sync/handler/device_auth.go). No JWT — devices are headless.
//
// All configuration is through environment variables; see .env.school-box.example.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edugraph-ai/edugraph/internal/syncagent"
)

type agentConfig struct {
	CloudAPIURL      string
	DeviceID         string
	DeviceSecret     string
	SchoolID         string
	LocalPostgresDSN string
	SyncInterval     time.Duration
	HealthAddr       string
}

func loadConfig() agentConfig {
	interval := 60
	if v := os.Getenv("SYNC_INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			interval = n
		}
	}
	// Build DSN from components if LOCAL_POSTGRES_DSN is not set directly.
	dsn := os.Getenv("LOCAL_POSTGRES_DSN")
	if dsn == "" {
		u := &url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(
				getenv("POSTGRES_USER", "edugraph"),
				getenv("POSTGRES_PASSWORD", "edugraph"),
			),
			Host: fmt.Sprintf("%s:%s",
				getenv("POSTGRES_HOST", "postgres"),
				getenv("POSTGRES_PORT", "5432"),
			),
			Path: "/" + getenv("POSTGRES_DB", "edugraph"),
		}
		q := u.Query()
		q.Set("sslmode", getenv("POSTGRES_SSLMODE", "disable"))
		u.RawQuery = q.Encode()
		dsn = u.String()
	}
	return agentConfig{
		CloudAPIURL:      getenv("CLOUD_API_URL", ""),
		DeviceID:         getenv("DEVICE_ID", ""),
		DeviceSecret:     getenv("DEVICE_SECRET", ""),
		SchoolID:         getenv("SCHOOL_ID", ""),
		LocalPostgresDSN: dsn,
		SyncInterval:     time.Duration(interval) * time.Second,
		HealthAddr:       getenv("HEALTH_ADDR", ":8081"),
	}
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func main() {
	cfg := loadConfig()

	// Validate required config.
	if cfg.CloudAPIURL == "" || cfg.DeviceID == "" || cfg.DeviceSecret == "" || cfg.SchoolID == "" {
		slog.Error("CLOUD_API_URL, DEVICE_ID, DEVICE_SECRET, and SCHOOL_ID are required")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Connect to local Postgres.
	pgPool, err := pgxpool.New(ctx, cfg.LocalPostgresDSN)
	if err != nil {
		slog.Error("postgres connect", "err", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	if err := pgPool.Ping(ctx); err != nil {
		slog.Error("postgres ping", "err", err)
		os.Exit(1)
	}
	slog.Info("connected to local postgres")

	cloud := syncagent.NewCloudClient(cfg.CloudAPIURL, cfg.DeviceID, cfg.DeviceSecret)
	state := &syncagent.SafeState{}

	// Health server — exposes GET /health for docker healthcheck and
	// monitoring scripts.
	syncagent.StartHealthServer(ctx, cfg.HealthAddr, state)
	slog.Info("health server listening", "addr", cfg.HealthAddr)

	// Track last successful pull time — start from epoch so the first pull
	// fetches everything, then advance each cycle to avoid re-applying old
	// changes.
	lastPullAt := time.Unix(0, 0)

	ticker := time.NewTicker(cfg.SyncInterval)
	defer ticker.Stop()

	slog.Info("sync agent started",
		"school_id", cfg.SchoolID,
		"device_id", cfg.DeviceID,
		"interval", cfg.SyncInterval,
	)

	// Run one cycle immediately on startup, then on every tick.
	runCycle := func() {
		now := time.Now()

		// Push phase.
		pushed, pushErr := syncagent.PushOutbox(ctx, pgPool, cloud, cfg.DeviceID, cfg.SchoolID, state)
		errMsg := ""
		if pushErr != nil {
			slog.Error("push outbox failed", "err", pushErr)
			errMsg = pushErr.Error()
		} else if pushed > 0 {
			slog.Info("pushed outbox rows", "count", pushed)
		}
		state.SetLastPush(now, errMsg)

		// Pull phase.
		pulled, pullErr := syncagent.PullCloud(ctx, pgPool, cloud, cfg.SchoolID, lastPullAt)
		pullErrMsg := ""
		if pullErr != nil {
			slog.Error("pull cloud failed", "err", pullErr)
			pullErrMsg = pullErr.Error()
		} else {
			if pulled > 0 {
				slog.Info("applied cloud changes", "count", pulled)
			}
			lastPullAt = now
		}
		state.SetLastPull(now, pullErrMsg)
	}

	runCycle()
	for {
		select {
		case <-ctx.Done():
			slog.Info("sync agent shutting down")
			return
		case <-ticker.C:
			runCycle()
		}
	}
}
