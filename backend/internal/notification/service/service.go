package service

import (
	"context"
	"errors"

	"github.com/edugraph-ai/edugraph/internal/notification/dto"
	"github.com/edugraph-ai/edugraph/internal/notification/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req dto.CreateNotificationRequest) (dto.NotificationResponse, error) {
	n, err := s.repo.Create(ctx, req.UserID, req.Title, req.Body)
	if err != nil {
		return dto.NotificationResponse{}, apperrors.Internal(err)
	}
	return toResponse(n), nil
}

func (s *Service) ListForUser(ctx context.Context, userID string, unreadOnly bool, p pagination.Params) ([]dto.NotificationResponse, int64, error) {
	items, total, err := s.repo.ListForUser(ctx, userID, unreadOnly, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	resp := make([]dto.NotificationResponse, 0, len(items))
	for _, n := range items {
		resp = append(resp, toResponse(n))
	}
	return resp, total, nil
}

func (s *Service) MarkRead(ctx context.Context, id, userID string) error {
	err := s.repo.MarkRead(ctx, id, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return apperrors.NotFound("notification not found")
	}
	if err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func toResponse(n repository.Notification) dto.NotificationResponse {
	return dto.NotificationResponse{ID: n.ID, Title: n.Title, Body: n.Body, IsRead: n.IsRead, CreatedAt: n.CreatedAt}
}
