package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edugraph-ai/edugraph/internal/reports/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, requesterID string, req dto.GenerateReportRequest) (dto.ReportResponse, error) {
	var id string
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx,
		`INSERT INTO public.report_results (report_type, params, requester_id, status)
		 VALUES ($1, $2::jsonb, $3::uuid, 'pending')
		 RETURNING id, created_at, updated_at`,
		req.ReportType,
		string(req.Params),
		requesterID,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return dto.ReportResponse{}, apperrors.Internal(err)
	}
	return dto.ReportResponse{
		ID:          id,
		ReportType:  req.ReportType,
		Params:      req.Params,
		RequesterID: requesterID,
		Status:      "pending",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

// ListByRequester returns every report the given user requested, newest
// first. Scoped to the caller's own reports -- report_results has no
// school/region column to scope by otherwise, and "my generated reports"
// is the correct default for a personal report queue regardless of role.
func (r *Repository) ListByRequester(ctx context.Context, requesterID string) ([]dto.ReportResponse, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, report_type, params, result, requester_id::text, status,
		        error_text, generated_at, created_at, updated_at
		 FROM public.report_results
		 WHERE requester_id = $1::uuid
		 ORDER BY created_at DESC`,
		requesterID,
	)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	defer rows.Close()

	items := make([]dto.ReportResponse, 0)
	for rows.Next() {
		var rep dto.ReportResponse
		var params, result []byte
		var errorText *string
		var generatedAt *time.Time
		if err := rows.Scan(
			&rep.ID, &rep.ReportType, &params, &result,
			&rep.RequesterID, &rep.Status,
			&errorText, &generatedAt, &rep.CreatedAt, &rep.UpdatedAt,
		); err != nil {
			return nil, apperrors.Internal(err)
		}
		if len(params) > 0 {
			rep.Params = json.RawMessage(params)
		}
		if len(result) > 0 {
			rep.Result = json.RawMessage(result)
		}
		rep.ErrorText = errorText
		rep.GeneratedAt = generatedAt
		items = append(items, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal(err)
	}
	return items, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (dto.ReportResponse, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, report_type, params, result, requester_id::text, status,
		        error_text, generated_at, created_at, updated_at
		 FROM public.report_results WHERE id = $1::uuid`,
		id,
	)
	var rep dto.ReportResponse
	var params, result []byte
	var errorText *string
	var generatedAt *time.Time
	err := row.Scan(
		&rep.ID, &rep.ReportType, &params, &result,
		&rep.RequesterID, &rep.Status,
		&errorText, &generatedAt, &rep.CreatedAt, &rep.UpdatedAt,
	)
	if err != nil {
		return dto.ReportResponse{}, apperrors.NotFound("report not found")
	}
	if len(params) > 0 {
		rep.Params = json.RawMessage(params)
	}
	if len(result) > 0 {
		rep.Result = json.RawMessage(result)
	}
	rep.ErrorText = errorText
	rep.GeneratedAt = generatedAt
	return rep, nil
}
