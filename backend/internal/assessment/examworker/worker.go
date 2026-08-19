// Package examworker auto-submits exam attempts whose server-set
// expires_at has passed. There is no per-request trigger for "time just
// ran out" the way there is for a student action (autosave/submit), so a
// short-interval ticker goroutine is the mechanism -- same established
// shape as qualityworker/syncworker in this same domain (a Go ticker in
// the api process, no Celery/separate deployable).
package examworker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/edugraph-ai/edugraph/internal/assessment/service"
)

// initialDelay mirrors qualityworker's rationale: let the stack settle
// after boot before the first sweep.
const initialDelay = 15 * time.Second

// Run sweeps for expired in_progress attempts shortly after startup,
// then every interval, until ctx is cancelled. A short interval (~30s in
// production wiring) keeps the gap between "time technically expired"
// and "actually auto-submitted" small without hammering the database --
// this table is small (only ever contains attempts, not every request).
func Run(ctx context.Context, svc *service.Service, interval time.Duration, log *zap.Logger) {
	log.Info("exam_worker.started", zap.Duration("interval", interval))

	timer := time.NewTimer(initialDelay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("exam_worker.stopped")
			return
		case <-timer.C:
			start := time.Now()
			succeeded, failed := svc.AutoSubmitExpiredAttempts(ctx)
			if succeeded > 0 || failed > 0 {
				log.Info("exam_worker.auto_submitted",
					zap.Int("succeeded", succeeded),
					zap.Int("failed", failed),
					zap.Duration("took", time.Since(start)))
			}
			timer.Reset(interval)
		}
	}
}
