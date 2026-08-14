package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/service"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// Handler is shared across every handler method in this package
// (exam_upload.go, exam_validate.go, exam_submit.go, exam_answer_key.go).
type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// callerID parses the authenticated caller's id off the request context
// (set by middleware.Authenticate). Shared by every handler that needs
// to pass its own identity down for a server-side ownership check
// (checklist 11.3, see service.verifyCallerOwnsExam) rather than trusting
// a client-supplied id.
func callerID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(middleware.UserID(r.Context()))
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid request body"))
		return false
	}
	return true
}
