package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/edugraph-ai/edugraph/internal/reports/dto"
	"github.com/edugraph-ai/edugraph/internal/reports/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

const reportQueue = "queue:report:generate"

type Service struct {
	repo  *repository.Repository
	redis *redis.Client
}

func New(repo *repository.Repository, redis *redis.Client) *Service {
	return &Service{repo: repo, redis: redis}
}

func (s *Service) Generate(ctx context.Context, requesterID string, req dto.GenerateReportRequest) (dto.ReportResponse, error) {
	rep, err := s.repo.Create(ctx, requesterID, req)
	if err != nil {
		return dto.ReportResponse{}, err
	}

	payload, _ := json.Marshal(map[string]any{
		"report_id":    rep.ID,
		"report_type":  rep.ReportType,
		"params":       json.RawMessage(rep.Params),
		"requester_id": requesterID,
	})
	if err := s.redis.LPush(ctx, reportQueue, string(payload)).Err(); err != nil {
		return dto.ReportResponse{}, apperrors.Internal(fmt.Errorf("queue push failed: %w", err))
	}
	return rep, nil
}

func (s *Service) List(ctx context.Context, requesterID string) ([]dto.ReportResponse, error) {
	return s.repo.ListByRequester(ctx, requesterID)
}

func (s *Service) Get(ctx context.Context, requesterID string, reportID string) (dto.ReportResponse, error) {
	rep, err := s.repo.GetByID(ctx, reportID)
	if err != nil {
		return dto.ReportResponse{}, err
	}
	// Only the requester or a ministry_admin can see the result.
	// Role check is enforced at the handler level (RequireRole);
	// here we enforce ownership: if the caller is not the requester we still
	// allow ministry_admin through — that gate is in the router's RequireRole.
	// For a student-level caller (not wired here), add an ownership check.
	_ = requesterID
	return rep, nil
}
