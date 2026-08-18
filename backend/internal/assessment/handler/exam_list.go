package handler

import (
	"net/http"

	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

// ListExams handles GET /api/v1/exams -- every exam at the caller's
// school, newest first (TeacherExamListPage, ClassAnalyticsPage,
// QuestionBankPage, TeacherDashboardPage).
func (h *Handler) ListExams(w http.ResponseWriter, r *http.Request) {
	userIDStr := middleware.UserID(r.Context())
	if userIDStr == "" {
		middleware.WriteError(w, apperrors.Unauthorized("user not authenticated"))
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	p := pagination.FromRequest(r)
	items, total, err := h.svc.ListExams(r.Context(), userID, p)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSONMeta(w, http.StatusOK, items, pagination.NewMeta(p, total))
}
