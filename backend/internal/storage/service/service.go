package service

import (
	"context"

	"github.com/edugraph-ai/edugraph/internal/storage/dto"
	"github.com/edugraph-ai/edugraph/internal/storage/repository"
	"github.com/edugraph-ai/edugraph/pkg/config"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

type Service struct {
	repo    *repository.Repository
	buckets map[string]string
}

func New(repo *repository.Repository, aws config.AWSConfig) *Service {
	return &Service{
		repo: repo,
		buckets: map[string]string{
			"curriculum": aws.CurriculumBucket,
			"exam":       aws.ExamBucket,
			"reports":    aws.ReportsBucket,
			"audit":      aws.AuditBucket,
		},
	}
}

func (s *Service) PresignUpload(ctx context.Context, req dto.PresignUploadRequest) (dto.PresignResponse, error) {
	bucket, ok := s.buckets[req.Bucket]
	if !ok {
		return dto.PresignResponse{}, apperrors.BadRequest("unknown bucket alias")
	}
	url, ttl, err := s.repo.PresignUpload(ctx, bucket, req.Key, req.ContentType)
	if err != nil {
		return dto.PresignResponse{}, apperrors.Internal(err)
	}
	return dto.PresignResponse{URL: url, Key: req.Key, ExpiresIn: int(ttl.Seconds())}, nil
}

func (s *Service) PresignDownload(ctx context.Context, req dto.PresignDownloadRequest) (dto.PresignResponse, error) {
	bucket, ok := s.buckets[req.Bucket]
	if !ok {
		return dto.PresignResponse{}, apperrors.BadRequest("unknown bucket alias")
	}
	url, ttl, err := s.repo.PresignDownload(ctx, bucket, req.Key)
	if err != nil {
		return dto.PresignResponse{}, apperrors.Internal(err)
	}
	return dto.PresignResponse{URL: url, Key: req.Key, ExpiresIn: int(ttl.Seconds())}, nil
}
