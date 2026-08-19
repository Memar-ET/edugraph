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

// errLog backs WriteError's 500-level logging. Previously WriteError
// silently discarded every internal-error detail with no logging at all
// (found while debugging a Supabase-migration-era regression that
// produced only an opaque client-side 500 with no server-side trace to
// diagnose it from) -- set once at startup via SetLogger, defaults to a
// no-op so tests that construct middleware without wiring a logger don't
// panic.
var errLog = zap.NewNop()

// SetLogger wires the application's real logger into WriteError's 500
// logging. Called once from router construction (see cmd/api/router.go),
// after the logger this package's other middleware (Logging, Recover)
// already receive as a constructor argument is built.
func SetLogger(log *zap.Logger) {
	errLog = log
}

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
	if appErr.Status >= 500 {
		errLog.Error("internal_error", zap.Error(err))
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
			// checklist 11.1: the browser only attaches the auth cookies
			// (see SetAuthCookies below) to a cross-origin fetch/XHR if
			// the request opts in with credentials: 'include' *and* the
			// server echoes this header -- without it, cookie-based auth
			// silently stops working for any frontend origin that isn't
			// same-origin with the API. Safe to always set: it only takes
			// effect together with a request that already opted in, and
			// wildcard-origin + credentials is rejected by browsers
			// anyway, so this can't widen the origin allowlist above.
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Authenticate accepts the access token from either the httpOnly
// AccessTokenCookie (the browser flow, see SetAuthCookies -- checklist
// 11.1, moved off localStorage specifically so JS/an XSS payload can
// never read this value at all) or a legacy "Authorization: Bearer
// <token>" header, checked in that order. The header path stays
// supported for non-browser callers (scripts, future mobile clients,
// curl/Postman during development) that have no cookie jar to rely on --
// it carries none of localStorage's XSS exposure since it's never
// persisted by this server, only whatever the caller does with it.
func Authenticate(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := tokenFromCookie(r)
			if token == "" {
				header := r.Header.Get("Authorization")
				if !strings.HasPrefix(header, "Bearer ") {
					WriteError(w, apperrors.Unauthorized("missing bearer token"))
					return
				}
				token = strings.TrimPrefix(header, "Bearer ")
			}
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

// ── Auth cookies (checklist 11.1) ───────────────────────────────
//
// Both tokens moved out of the frontend's localStorage (readable by any
// script on the page, including an XSS payload -- the exact risk
// auth.store.ts's own prior comment already flagged) into HttpOnly
// cookies the browser sends automatically and JavaScript can never read
// at all. SameSite=Lax rather than Strict: Strict drops the cookie on a
// top-level GET navigation arriving from an external link/redirect
// (e.g. opening the app from an email link while already logged in),
// which would incorrectly bounce a legitimate user to /login; Lax still
// blocks the cross-site POST/fetch forgery this is actually defending
// against. No separate CSRF token: every state-changing request here is
// JSON (Content-Type: application/json), which a cross-site <form> can't
// forge without triggering a CORS preflight the origin allowlist above
// already rejects.
const (
	AccessTokenCookie  = "edugraph_access_token"
	RefreshTokenCookie = "edugraph_refresh_token"
)

func tokenFromCookie(r *http.Request) string {
	c, err := r.Cookie(AccessTokenCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

// SetAuthCookies sets both auth cookies after a successful login/
// register/refresh. secure must be true in any environment served over
// HTTPS (the browser silently drops a Secure cookie set over plain HTTP,
// which is why this isn't just hardcoded true -- local dev over
// http://localhost needs it false to work at all).
func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string, accessTTL, refreshTTL time.Duration, secure bool) {
	http.SetCookie(w, authCookie(AccessTokenCookie, accessToken, int(accessTTL.Seconds()), secure))
	http.SetCookie(w, authCookie(RefreshTokenCookie, refreshToken, int(refreshTTL.Seconds()), secure))
}

// ClearAuthCookies expires both cookies immediately -- called on logout
// and whenever a refresh attempt fails (the refresh token was invalid/
// revoked/expired, so there's nothing left worth keeping client-side).
// MaxAge: -1 here is seconds, not a time.Duration -- Go's net/http
// treats a negative Max-Age as "delete now"; a stray time.Duration(-1)
// (one nanosecond) would round to 0, which instead means "no Max-Age
// attribute" (a session cookie), not deletion.
func ClearAuthCookies(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, authCookie(AccessTokenCookie, "", -1, secure))
	http.SetCookie(w, authCookie(RefreshTokenCookie, "", -1, secure))
}

func authCookie(name, value string, maxAgeSeconds int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAgeSeconds,
	}
}

// Role reads the authenticated user role from context.
func Role(ctx context.Context) string {
	role, _ := ctx.Value(contextkeys.UserRoleKey).(string)
	return role
}
