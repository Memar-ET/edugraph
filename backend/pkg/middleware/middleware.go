// Package middleware provides HTTP middleware shared across all domain
// routers: request logging, panic recovery, CORS, and JWT authentication.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/edugraph-ai/edugraph/pkg/contextkeys"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// Envelope is the standard JSON response shape for every endpoint.
type Envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Success: status < 400, Data: data})
}

func WriteJSONMeta(w http.ResponseWriter, status int, data any, meta any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Success: status < 400, Data: data, Meta: meta})
}

func WriteError(w http.ResponseWriter, err error) {
	appErr, ok := apperrors.As(err)
	if !ok {
		appErr = apperrors.Internal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Status)
	_ = json.NewEncoder(w).Encode(Envelope{Success: false, Error: appErr.Message})
}

// TokenVerifier verifies a bearer access token and returns the subject's
// user id and role. Implemented by internal/auth/service.
type TokenVerifier interface {
	VerifyAccessToken(ctx context.Context, token string) (userID string, role string, err error)
}

func Logging(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			log.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rec.status),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Recover(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered", zap.Any("panic", rec), zap.String("path", r.URL.Path))
					WriteError(w, apperrors.New(http.StatusInternalServerError, "internal_error", "an internal error occurred"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		origins[o] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := origins[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Authenticate requires a valid "Authorization: Bearer <token>" header and
// stores the resolved user id/role on the request context.
func Authenticate(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				WriteError(w, apperrors.Unauthorized("missing bearer token"))
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")
			userID, role, err := verifier.VerifyAccessToken(r.Context(), token)
			if err != nil {
				WriteError(w, apperrors.Unauthorized("invalid or expired token"))
				return
			}
			ctx := context.WithValue(r.Context(), contextkeys.UserIDKey, userID)
			ctx = context.WithValue(ctx, contextkeys.UserRoleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole restricts access to the given roles. Must run after Authenticate.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(contextkeys.UserRoleKey).(string)
			if _, ok := allowed[role]; !ok {
				WriteError(w, apperrors.Forbidden("insufficient permissions"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserID reads the authenticated user id from context. Only valid on
// requests that passed through Authenticate.
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(contextkeys.UserIDKey).(string)
	return id
}

// Role reads the authenticated user role from context.
func Role(ctx context.Context) string {
	role, _ := ctx.Value(contextkeys.UserRoleKey).(string)
	return role
}
