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

// bucketRoles: checklist 11.3 finding -- these two endpoints had no role
// check at all beyond being logged in, so any authenticated account
// (including a plain student) could presign a GET or PUT for any key in
// any of the four S3 buckets, including "audit" (audit logs -- the most
// sensitive bucket here) and "reports". This dormant in local dev today
// (repository.New loads AWS credentials lazily and nothing here is
// wired into the active curriculum/exam upload flow -- see CLAUDE.md's
// "No S3StorageProvider exists yet"), but the presign code itself is
// real, functional AWS SDK usage, not a stub, so it's a real
// vulnerability wherever AWS credentials for these buckets ARE
// configured. Least-privilege per bucket, not just "any authenticated
// user": curriculum uploads are curriculum_officer/ministry_admin's own
// domain (matches /curriculum/upload's own role gate), exam files are
// teacher/school_admin's, reports/audit are ministry-level oversight
// data nobody else has a reason to read or write.
var bucketRoles = map[string][]string{
	"curriculum": {"curriculum_officer", "ministry_admin"},
	"exam":       {"teacher", "school_admin"},
	"reports":    {"ministry_admin", "regional_admin", "school_admin"},
	"audit":      {"ministry_admin"},
}

func (s *Service) resolveBucket(bucketAlias, role string) (string, error) {
	bucket, ok := s.buckets[bucketAlias]
	if !ok {
		return "", apperrors.BadRequest("unknown bucket alias")
	}
	allowed := bucketRoles[bucketAlias]
	for _, r := range allowed {
		if r == role {
			return bucket, nil
		}
	}
	return "", apperrors.Forbidden("not permitted to access this bucket")
}

func (s *Service) PresignUpload(ctx context.Context, role string, req dto.PresignUploadRequest) (dto.PresignResponse, error) {
	bucket, err := s.resolveBucket(req.Bucket, role)
	if err != nil {
		return dto.PresignResponse{}, err
	}
	url, ttl, err := s.repo.PresignUpload(ctx, bucket, req.Key, req.ContentType)
	if err != nil {
		return dto.PresignResponse{}, apperrors.Internal(err)
	}
	return dto.PresignResponse{URL: url, Key: req.Key, ExpiresIn: int(ttl.Seconds())}, nil
}

func (s *Service) PresignDownload(ctx context.Context, role string, req dto.PresignDownloadRequest) (dto.PresignResponse, error) {
	bucket, err := s.resolveBucket(req.Bucket, role)
	if err != nil {
		return dto.PresignResponse{}, err
	}
	url, ttl, err := s.repo.PresignDownload(ctx, bucket, req.Key)
	if err != nil {
		return dto.PresignResponse{}, apperrors.Internal(err)
	}
	return dto.PresignResponse{URL: url, Key: req.Key, ExpiresIn: int(ttl.Seconds())}, nil
}
