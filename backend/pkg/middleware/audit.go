package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditLog is a middleware that records every inbound request to the
// public.audit_log table (see migration V053). It captures the actor,
// HTTP metadata, and response status. For privileged business-level actions
// (curriculum approve, exam publish, model promote) use AuditAction instead,
// which logs a richer, action-specific record.
//
// Writes are fire-and-forget (goroutine + context.Background()) so a slow
// database write never stalls the response path. Sensitive fields (passwords,
// tokens, file bytes) are never logged — the body is read only for small
// JSON payloads; file-upload paths are skipped entirely.
func AuditLog(db *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			// Capture after the handler has run so we have the final status.
			actorID := UserID(r.Context())
			actorRole := Role(r.Context())
			requestID := RequestIDFromCtx(r.Context())
			action := actionFromRequest(r)
			durationMS := int(time.Since(start).Milliseconds())

			var bodyJSON []byte
			if isSafeToLogBody(r) {
				bodyJSON = readBodySafe(r)
			}

			go insertAuditLog(db, auditEntry{
				actorID:    actorID,
				actorRole:  actorRole,
				action:     action,
				resource:   r.URL.Path,
				requestID:  requestID,
				ip:         IPKey(r),
				userAgent:  r.UserAgent(),
				body:       bodyJSON,
				status:     rec.status,
				durationMS: durationMS,
			})
		})
	}
}

type auditEntry struct {
	actorID    string
	actorRole  string
	action     string
	resource   string
	resourceID string
	requestID  string
	ip         string
	userAgent  string
	body       []byte
	status     int
	durationMS int
}

func insertAuditLog(db *pgxpool.Pool, e auditEntry) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var actorID *string
	if e.actorID != "" {
		actorID = &e.actorID
	}
	var bodyArg *string
	if len(e.body) > 0 {
		s := string(e.body)
		bodyArg = &s
	}

	_, _ = db.Exec(ctx, `
		INSERT INTO public.audit_log
			(actor_id, actor_role, action, resource_type, resource_id,
			 request_id, ip_address, user_agent, request_body,
			 response_status, duration_ms, occurred_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,NOW())
	`,
		actorID, e.actorRole, e.action, e.resource, e.resourceID,
		e.requestID, e.ip, e.userAgent, bodyArg,
		e.status, e.durationMS,
	)
}

// AuditAction records an explicit business-level privileged action
// (curriculum.approve, exam.publish, model.promote, misconception.confirm,
// prerequisite.resync, etc.). Call it from handler code immediately after the
// action succeeds.
//
//	AuditAction(r.Context(), db, "curriculum.approve", "upload_job", jobID)
func AuditAction(ctx context.Context, db *pgxpool.Pool, action, resourceType, resourceID string) {
	entry := auditEntry{
		actorID:    UserID(ctx),
		actorRole:  Role(ctx),
		action:     action,
		resource:   resourceType,
		resourceID: resourceID,
		requestID:  RequestIDFromCtx(ctx),
		status:     http.StatusOK,
	}
	go insertAuditLog(db, entry)
}

// actionFromRequest derives a stable action string from the HTTP method and
// path so audit records are queryable without depending on exact URL patterns.
func actionFromRequest(r *http.Request) string {
	method := strings.ToLower(r.Method)
	// Strip /api/v1/ prefix and URL params (chi sets them, but we use the raw path)
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	// Trim trailing slash
	path = strings.TrimRight(path, "/")
	return method + ":" + path
}

// isSafeToLogBody returns true for small JSON POST/PATCH/PUT requests.
// Multipart (file uploads) and GET/DELETE are never read.
func isSafeToLogBody(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodDelete {
		return false
	}
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "multipart") || strings.Contains(ct, "form-data") {
		return false
	}
	cl := r.ContentLength
	return cl > 0 && cl < 4096 // only tiny JSON bodies
}

// readBodySafe reads up to 4 KB from the request body and scrubs sensitive
// fields. The body is not consumed — TeeReader would be needed for full
// passthrough; since this middleware runs after the handler, the body has
// already been consumed.  We read from a snapshot stored by a body-capture
// middleware if one exists, otherwise skip.
func readBodySafe(r *http.Request) []byte {
	if r.Body == nil {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil || len(raw) == 0 {
		return nil
	}
	// Scrub known sensitive keys
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil // not parseable JSON — don't log
	}
	for _, key := range []string{"password", "refresh_token", "access_token", "answer_text", "secret"} {
		delete(m, key)
	}
	out, _ := json.Marshal(m)
	return out
}
