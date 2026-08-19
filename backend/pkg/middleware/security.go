package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/pkg/contextkeys"
)

// SecurityHeaders adds defence-in-depth HTTP security headers to every
// response. Wire this as the outermost middleware so it covers every path
// including /health and /metrics.
//
// HSTS is only set in production: the browser silently ignores Secure
// cookies on plain http://localhost, so requiring HTTPS in dev would break
// local development entirely.
func SecurityHeaders(appEnv string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-XSS-Protection", "1; mode=block")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			h.Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self' 'unsafe-inline'; "+
					"style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'")
			if appEnv == "production" {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID injects a unique request ID into each request context and echoes
// it back as X-Request-ID so callers can correlate logs with responses.
// If the incoming request already carries X-Request-ID (e.g. from a load
// balancer or upstream proxy), that value is reused rather than overwritten.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), contextkeys.RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromCtx reads the request ID injected by RequestID middleware.
func RequestIDFromCtx(ctx context.Context) string {
	id, _ := ctx.Value(contextkeys.RequestIDKey).(string)
	return id
}
