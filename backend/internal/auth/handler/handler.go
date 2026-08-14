package handler

import (
	"encoding/json"
	"net/http"

	"github.com/edugraph-ai/edugraph/internal/auth/dto"
	"github.com/edugraph-ai/edugraph/internal/auth/service"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/validator"
)

type Handler struct {
	svc *service.Service
	// secureCookies gates the auth cookies' Secure attribute (checklist
	// 11.1) -- must be true wherever the app is actually served over
	// HTTPS, but a Secure cookie is silently dropped by the browser over
	// plain http://localhost, which local dev still is. See main.go for
	// how this is derived from APP_ENV.
	secureCookies bool
}

func New(svc *service.Service, secureCookies bool) *Handler {
	return &Handler{svc: svc, secureCookies: secureCookies}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	resp, err := h.svc.Register(r.Context(), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	h.setCookies(w, resp)
	middleware.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	resp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	h.setCookies(w, resp)
	middleware.WriteJSON(w, http.StatusOK, resp)
}

// Refresh reads the refresh token from its HttpOnly cookie (checklist
// 11.1) rather than a JSON body -- there's no longer a JS-accessible
// copy of it to send one with. A missing cookie means the browser was
// never logged in (or already logged out) here, same response either
// way: Unauthorized.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(middleware.RefreshTokenCookie)
	if err != nil || cookie.Value == "" {
		middleware.WriteError(w, apperrors.Unauthorized("missing refresh token"))
		return
	}

	resp, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		// The stored refresh token is invalid/revoked/expired -- clear
		// whatever's left client-side rather than leaving a dead cookie
		// around that will just fail the same way on every request.
		middleware.ClearAuthCookies(w, h.secureCookies)
		middleware.WriteError(w, err)
		return
	}
	h.setCookies(w, resp)
	middleware.WriteJSON(w, http.StatusOK, resp)
}

// Logout revokes the refresh token server-side (if present -- a missing
// cookie just means there's nothing to revoke, not an error: logging out
// twice, or logging out an already-expired session, must still succeed)
// and clears both cookies either way.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(middleware.RefreshTokenCookie); err == nil && cookie.Value != "" {
		if err := h.svc.Logout(r.Context(), cookie.Value); err != nil {
			middleware.ClearAuthCookies(w, h.secureCookies)
			middleware.WriteError(w, err)
			return
		}
	}
	middleware.ClearAuthCookies(w, h.secureCookies)
	middleware.WriteJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	resp, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) setCookies(w http.ResponseWriter, resp dto.AuthResponse) {
	middleware.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken, h.svc.AccessTTL(), h.svc.RefreshTTL(), h.secureCookies)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid request body"))
		return false
	}
	return true
}
