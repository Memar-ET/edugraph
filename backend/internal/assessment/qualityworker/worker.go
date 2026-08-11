// Package qualityworker recalculates school quality scores (Capability
// 4C) on a nightly cadence. The impl plan says "nightly Celery batch
// job", but this repo deliberately has no Celery (see
// ai-service/app/workers/curriculum_worker.py's rationale) -- a Go
// ticker goroutine in the api process is the established shape for
// background recomputation here (see assessment/syncworker).
package qualityworker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/edugraph-ai/edugraph/internal/assessment/service"
)

// initialDelay gives the stack (Postgres, migrations) time to settle
// before the first full recompute after boot -- and means a dev stack
// gets fresh scores shortly after `docker compose up`, not 24h later.
const initialDelay = 90 * time.Second

// Run executes one recompute shortly after startup, then every interval
// (24h in production wiring), until ctx is cancelled. Failures are
// logged and retried next cycle -- scores just stay stale, dashboards
// keep serving the previous row.
func Run(ctx context.Context, svc *service.Service, interval time.Duration, log *zap.Logger) {
	log.Info("quality_worker.started", zap.Duration("interval", interval))

	timer := time.NewTimer(initialDelay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("quality_worker.stopped")
			return
		case <-timer.C:
			start := time.Now()
			scored, err := svc.RecomputeAllSchoolQuality(ctx)
			if err != nil {
				log.Error("quality_worker.recompute_failed", zap.Error(err))
			} else {
				log.Info("quality_worker.recomputed",
					zap.Int("combosScored", scored),
					zap.Duration("took", time.Since(start)))
			}
			timer.Reset(interval)
		}
	}
}
