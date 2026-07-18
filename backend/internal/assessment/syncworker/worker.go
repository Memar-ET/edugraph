// Package syncworker mirrors graded exam attempts/answers into Neo4j
// (Capability 2C). The schema's neo4j_written flag + partial index
// (backend/db/migrations/V011__updated_curriculum.sql) were built for
// exactly this kind of poller -- curriculum's Neo4j mirror
// (internal/curriculum/repository/repository.go's syncCurriculumGraph)
// took a synchronous-mirror-after-commit shortcut instead, documented as a
// deliberate simplification, not a pattern to copy here: 2C's spec
// explicitly calls out a single "Sync Worker" both submission flows feed,
// which fits a shared poller better than duplicating the mirror call into
// two separate Go handlers.
package syncworker

import (
	"context"
	"time"

	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	"go.uber.org/zap"
)

const batchSize = 50

// Run polls Postgres for unsynced attempts/answers and mirrors them into
// Neo4j, forever, until ctx is cancelled. A write failure is logged and
// left neo4j_written=false for the next tick -- same non-blocking,
// idempotent-retry semantics as curriculum's mirror, just polling instead
// of inline.
func Run(ctx context.Context, repo *repository.Repository, interval time.Duration, log *zap.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info("sync_worker.started", zap.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			log.Info("sync_worker.stopped")
			return
		case <-ticker.C:
			syncOnce(ctx, repo, log)
		}
	}
}

func syncOnce(ctx context.Context, repo *repository.Repository, log *zap.Logger) {
	attempts, err := repo.FetchUnsyncedAttempts(ctx, batchSize)
	if err != nil {
		log.Error("sync_worker.fetch_attempts_failed", zap.Error(err))
	}
	for _, a := range attempts {
		if err := repo.SyncAttemptToNeo4j(ctx, a); err != nil {
			log.Error("sync_worker.sync_attempt_failed", zap.String("attemptId", a.ID.String()), zap.Error(err))
			continue
		}
		if err := repo.MarkAttemptSynced(ctx, a.ID); err != nil {
			log.Error("sync_worker.mark_attempt_synced_failed", zap.String("attemptId", a.ID.String()), zap.Error(err))
		}
	}

	answers, err := repo.FetchUnsyncedAnswers(ctx, batchSize)
	if err != nil {
		log.Error("sync_worker.fetch_answers_failed", zap.Error(err))
	}
	for _, a := range answers {
		if err := repo.SyncAnswerToNeo4j(ctx, a); err != nil {
			log.Error("sync_worker.sync_answer_failed", zap.String("answerId", a.ID.String()), zap.Error(err))
			continue
		}
		if err := repo.MarkAnswerSynced(ctx, a.ID); err != nil {
			log.Error("sync_worker.mark_answer_synced_failed", zap.String("answerId", a.ID.String()), zap.Error(err))
		}
	}

	if len(attempts) > 0 || len(answers) > 0 {
		log.Info("sync_worker.tick", zap.Int("attemptsSynced", len(attempts)), zap.Int("answersSynced", len(answers)))
	}
}
