package service

import (
	"context"
	"errors"
	"time"

	"github.com/edugraph-ai/edugraph/internal/sync/dto"
	"github.com/edugraph-ai/edugraph/internal/sync/repository"
	"github.com/edugraph-ai/edugraph/pkg/crypto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// VerifyDevice checks a School Box's device_id + secret against
// sync.device_credentials (see V030__sync_device_credentials.sql) and
// returns the school_id that credential is bound to. Implements
// handler.DeviceVerifier -- kept in the service layer, not the handler,
// so the bcrypt compare and repository access stay behind the same
// boundary as everything else here.
func (s *Service) VerifyDevice(ctx context.Context, deviceID, secret string) (string, error) {
	cred, err := s.repo.GetDeviceCredential(ctx, deviceID)
	if errors.Is(err, repository.ErrDeviceNotFound) {
		return "", apperrors.Unauthorized("unknown device")
	}
	if err != nil {
		return "", apperrors.Internal(err)
	}
	if cred.RevokedAt != nil {
		return "", apperrors.Unauthorized("device credential revoked")
	}
	if err := crypto.ComparePassword(cred.SecretHash, secret); err != nil {
		return "", apperrors.Unauthorized("invalid device credentials")
	}

	// Best-effort operational visibility only -- a failure here must
	// never block an otherwise-valid sync request.
	_ = s.repo.TouchDeviceLastSeen(ctx, deviceID)

	return cred.SchoolID, nil
}

// Push accepts a batch of offline changes from a School Box device and
// queues them for the sync-worker to apply. deviceSchoolID is the school
// the caller's verified device credential is bound to (from
// VerifyDevice, via the DeviceAuth middleware) -- req.SchoolID must
// match it, or a device authenticated for School A could still write
// data tagged to School B just by changing the request body.
func (s *Service) Push(ctx context.Context, deviceSchoolID string, req dto.PushRequest) (dto.PushResponse, error) {
	if req.SchoolID != deviceSchoolID {
		return dto.PushResponse{}, apperrors.Forbidden("school_id does not match this device's registered school")
	}
	for _, change := range req.Changes {
		_, err := s.repo.CreateLog(ctx, repository.CreateLogParams{
			DeviceID:   req.DeviceID,
			SchoolID:   req.SchoolID,
			EntityType: change.EntityType,
			EntityID:   change.EntityID,
			Operation:  change.Operation,
			Payload:    change.Payload,
		})
		if err != nil {
			return dto.PushResponse{}, apperrors.Internal(err)
		}
	}
	return dto.PushResponse{Accepted: len(req.Changes)}, nil
}

// Pull returns changes applied to schoolID after `since`, for a device to
// merge into its local offline store. Same deviceSchoolID cross-check as
// Push -- schoolID here is a query param, so it's just as spoofable if
// left unchecked.
func (s *Service) Pull(ctx context.Context, deviceSchoolID, schoolID string, since time.Time) (dto.PullResponse, error) {
	if schoolID != deviceSchoolID {
		return dto.PullResponse{}, apperrors.Forbidden("school_id does not match this device's registered school")
	}
	logs, err := s.repo.ListApplied(ctx, schoolID, since)
	if err != nil {
		return dto.PullResponse{}, apperrors.Internal(err)
	}

	changes := make([]dto.PulledChange, 0, len(logs))
	for _, l := range logs {
		pc := dto.PulledChange{EntityType: l.EntityType, EntityID: l.EntityID, Operation: l.Operation, Payload: l.Payload}
		if l.SyncedAt != nil {
			pc.SyncedAt = *l.SyncedAt
		}
		changes = append(changes, pc)
	}

	return dto.PullResponse{Changes: changes, ServerTime: time.Now()}, nil
}
