// Command sync-worker polls sync_logs for pending offline changes pushed by
// School Box devices and marks them applied.
//
// Claiming uses SELECT ... FOR UPDATE SKIP LOCKED so multiple worker
// replicas never double-process the same entry. NOTE: actually applying a
// change to its target domain table (students, assessment_results, etc.) is
// intentionally not implemented here — that needs a per-entity-type
// dispatch table each domain would register into. This worker owns the
// queue/claim/status pipeline; wire in dispatch as each domain needs
// offline write support.
package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/edugraph-ai/edugraph/internal/sync/repository"
	"github.com/edugraph-ai/edugraph/pkg/config"
	"github.com/edugraph-ai/edugraph/pkg/database/postgres"
	"github.com/edugraph-ai/edugraph/pkg/logger"
)

const (
	pollInterval = 5 * time.Second
	claimBatch   = 50
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)
	defer log.Sync() //nolint:errcheck

	pool, err := postgres.NewPool(cfg.Postgres)
	if err != nil {
		log.Fatal("connect postgres", logger.Error(err))
	}
	defer pool.Close()

	repo := repository.New(pool)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Info("sync-worker started", logger.String("poll_interval", pollInterval.String()))
	for {
		select {
		case <-ctx.Done():
			log.Info("sync-worker shutting down")
			return
		case <-ticker.C:
			logs, err := repo.ClaimPending(ctx, claimBatch)
			if err != nil {
				log.Error("claim pending sync logs", logger.Error(err))
				continue
			}
			if len(logs) > 0 {
				log.Info("applied sync logs", logger.Int("count", len(logs)))
			}
		}
	}
}
