package repository

import (
	"context"

	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// RecordAuditAction wraps middleware.AuditAction so the service layer
// doesn't need its own *pgxpool.Pool dependency. AuditAction/public.audit_log
// (V053) already existed and worked -- it was simply never called
// anywhere in this codebase (confirmed by grep during the exam
// production-hardening audit), so administrative actions produced zero
// audit rows despite the infra being fully built. Fire-and-forget by
// design (see middleware.AuditAction's own doc comment) -- an audit-log
// write must never be able to fail or slow down the actual action.
func (r *Repository) RecordAuditAction(ctx context.Context, action, resourceType, resourceID string) {
	middleware.AuditAction(ctx, r.pool, action, resourceType, resourceID)
}
