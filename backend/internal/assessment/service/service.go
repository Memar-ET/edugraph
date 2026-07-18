package service

import (
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	"github.com/edugraph-ai/edugraph/pkg/ai"
	"github.com/edugraph-ai/edugraph/pkg/storage"
	"github.com/redis/go-redis/v9"
)

// Service is shared across every service method in this package
// (exam_upload.go, exam_validate.go, exam_submit.go, exam_answer_key.go,
// exam_insight.go, study_plan.go, tutor.go).
type Service struct {
	repo    *repository.Repository
	storage storage.StorageProvider
	redis   *redis.Client
	ai      *ai.Client
}

func New(repo *repository.Repository, storage storage.StorageProvider, redis *redis.Client, ai *ai.Client) *Service {
	return &Service{repo: repo, storage: storage, redis: redis, ai: ai}
}
