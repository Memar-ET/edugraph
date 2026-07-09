package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateJob inserts a new upload job into the database.
func (r *Repository) CreateJob(ctx context.Context, uploadedBy uuid.UUID, req dto.UploadRequest, fileRef string, fileName string, fileSize int64) (uuid.UUID, error) {
	var jobID uuid.UUID
	query := `
		INSERT INTO curriculum.upload_jobs 
			(uploaded_by, subject_code, grade_level, academic_year, file_s3_key, file_name, file_size_bytes, status)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, 'pending')
		RETURNING id
	`
	// Note: We use file_s3_key to store the Postgres UUID in Dev mode.
	// In Prod, this will be the S3 Key.
	err := r.pool.QueryRow(ctx, query,
		uploadedBy, req.SubjectCode, req.GradeLevel, req.AcademicYear,
		fileRef, fileName, fileSize,
	).Scan(&jobID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("create upload job: %w", err)
	}
	return jobID, nil
}

// GetJob retrieves the full state of a job, including the AI-parsed tree
// once available -- this backs the Step 3 review screen.
func (r *Repository) GetJob(ctx context.Context, jobID uuid.UUID) (*dto.JobStatus, error) {
	var job dto.JobStatus
	var errStr *string
	var parsedRaw []byte
	query := `
		SELECT id, status, file_name, subject_code, grade_level, academic_year,
		       parsed_structure, approved_by, approved_at, parse_error
		FROM curriculum.upload_jobs 
		WHERE id = $1
	`
	err := r.pool.QueryRow(ctx, query, jobID).Scan(
		&job.JobID, &job.Status, &job.FileName, &job.SubjectCode, &job.GradeLevel, &job.AcademicYear,
		&parsedRaw, &job.ApprovedBy, &job.ApprovedAt, &errStr,
	)
	if err != nil {
		return nil, err
	}
	if len(parsedRaw) > 0 {
		job.ParsedStructure = parsedRaw
	}
	job.Error = errStr
	return &job, nil
}

// GetFileRef returns the storage reference and original filename for a job,
// used by the Step 3 "view original PDF" proxy endpoint.
func (r *Repository) GetFileRef(ctx context.Context, jobID uuid.UUID) (fileRef string, fileName string, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT file_s3_key, file_name FROM curriculum.upload_jobs WHERE id = $1`,
		jobID,
	).Scan(&fileRef, &fileName)
	return fileRef, fileName, err
}

// jobCore is the subset of a job row needed to run the approval workflow.
type jobCore struct {
	SubjectCode     string
	GradeLevel      int
	AcademicYear    string
	Status          string
	ParsedStructure []byte
}

func (r *Repository) getJobCore(ctx context.Context, tx pgx.Tx, jobID uuid.UUID, forUpdate bool) (*jobCore, error) {
	query := `
		SELECT subject_code, grade_level, academic_year, status, parsed_structure
		FROM curriculum.upload_jobs
		WHERE id = $1
	`
	if forUpdate {
		query += " FOR UPDATE"
	}
	var jc jobCore
	err := tx.QueryRow(ctx, query, jobID).Scan(
		&jc.SubjectCode, &jc.GradeLevel, &jc.AcademicYear, &jc.Status, &jc.ParsedStructure,
	)
	if err != nil {
		return nil, err
	}
	return &jc, nil
}

// GetJobForApproval fetches the fields the service layer needs before
// promoting: the job's current status (must be 'parsed' or 'review') and
// the previously-stored tree, used when the caller doesn't submit an
// edited one of their own.
func (r *Repository) GetJobForApproval(ctx context.Context, jobID uuid.UUID) (status string, parsedStructure []byte, err error) {
	query := `SELECT status, parsed_structure FROM curriculum.upload_jobs WHERE id = $1`
	err = r.pool.QueryRow(ctx, query, jobID).Scan(&status, &parsedStructure)
	return status, parsedStructure, err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ApproveAndPromote is the core of Step 3: it locks the job row, verifies
// it's actually in a state that can be approved, then promotes every
// unit/topic/CLO in `structure` into the real curriculum.* tables, and
// finally marks the job 'approved'. Everything happens in one transaction,
// so a failure partway through (e.g. a malformed CLO) leaves the database
// exactly as it was -- no half-promoted job.
//
// Units/topics/CLOs are upserted (ON CONFLICT ... DO UPDATE) keyed on their
// natural key (see migration V016), so approving the same job twice (e.g.
// after further edits) updates the existing rows rather than duplicating
// them.
func (r *Repository) ApproveAndPromote(
	ctx context.Context,
	jobID uuid.UUID,
	userID uuid.UUID,
	structure dto.ParsedStructurePayload,
	finalStructureJSON []byte,
) (*dto.ApproveResponse, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op once committed

	// Lock the row and re-check status inside the transaction -- this is
	// the authoritative check (avoids a race between two concurrent
	// approve requests for the same job).
	core, err := r.getJobCore(ctx, tx, jobID, true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("job not found")
		}
		return nil, fmt.Errorf("lock job: %w", err)
	}
	if core.Status != "parsed" && core.Status != "review" {
		return nil, apperrors.Conflict(fmt.Sprintf(
			"job status is %q; only jobs with status 'parsed' or 'review' can be approved", core.Status,
		))
	}

	subjectCode, gradeLevel, academicYear := core.SubjectCode, core.GradeLevel, core.AcademicYear

	// 1. Subject (upsert -- name_en has no better source yet than the code
	// itself; officers can rename it later via a subjects endpoint).
	_, err = tx.Exec(ctx, `
		INSERT INTO curriculum.subjects (code, name_en, grade_level, academic_year, upload_job_id)
		VALUES ($1, $1, $2, $3, $4)
		ON CONFLICT (code) DO UPDATE SET
			grade_level   = EXCLUDED.grade_level,
			academic_year = EXCLUDED.academic_year,
			upload_job_id = EXCLUDED.upload_job_id
	`, subjectCode, gradeLevel, academicYear, jobID)
	if err != nil {
		return nil, fmt.Errorf("upsert subject %q: %w", subjectCode, err)
	}

	var unitsPromoted, topicsPromoted, closPromoted int
	moeVersion := fmt.Sprintf("ai-draft-%s", academicYear)

	for _, u := range structure.Units {
		var unitID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO curriculum.units (subject_code, grade_level, number, title_en)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (subject_code, number) DO UPDATE SET title_en = EXCLUDED.title_en
			RETURNING id
		`, subjectCode, gradeLevel, u.Number, u.TitleEn).Scan(&unitID)
		if err != nil {
			return nil, fmt.Errorf("upsert unit %d (%q): %w", u.Number, u.TitleEn, err)
		}
		unitsPromoted++

		for _, t := range u.Topics {
			var topicID uuid.UUID
			err = tx.QueryRow(ctx, `
				INSERT INTO curriculum.topics
					(unit_id, subject_code, grade_level, sequence_order, title_en, description, key_concepts)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (unit_id, sequence_order) DO UPDATE SET
					title_en     = EXCLUDED.title_en,
					description  = EXCLUDED.description,
					key_concepts = EXCLUDED.key_concepts
				RETURNING id
			`, unitID, subjectCode, gradeLevel, t.SequenceOrder, t.TitleEn, nullIfEmpty(t.RawText), t.KeyConcepts).Scan(&topicID)
			if err != nil {
				return nil, fmt.Errorf("upsert topic %q (unit %d): %w", t.TitleEn, u.Number, err)
			}
			topicsPromoted++

			for _, c := range t.Clos {
				if c.Code == "" {
					continue // can't upsert a CLO without its natural key
				}
				_, err = tx.Exec(ctx, `
					INSERT INTO curriculum.clos
						(code, subject_code, grade_level, description_en, bloom_level, is_mandatory, moe_version)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
					ON CONFLICT (code) DO UPDATE SET
						description_en = EXCLUDED.description_en,
						bloom_level    = EXCLUDED.bloom_level,
						is_mandatory   = EXCLUDED.is_mandatory
				`, c.Code, subjectCode, gradeLevel, c.Description, nullIfEmpty(c.BloomLevel), c.Mandatory, moeVersion)
				if err != nil {
					return nil, fmt.Errorf("upsert clo %q: %w", c.Code, err)
				}

				_, err = tx.Exec(ctx, `
					INSERT INTO curriculum.topic_clo_mappings (topic_id, clo_code, match_method, reviewed_by, confirmed_at)
					VALUES ($1, $2, 'human_confirmed', $3, now())
					ON CONFLICT (topic_id, clo_code) DO UPDATE SET
						match_method = 'human_confirmed',
						reviewed_by  = EXCLUDED.reviewed_by,
						confirmed_at = now()
				`, topicID, c.Code, userID)
				if err != nil {
					return nil, fmt.Errorf("upsert topic_clo_mapping %q: %w", c.Code, err)
				}
				closPromoted++
			}
		}
	}

	// 2. Mark the job approved, persisting the final (possibly edited)
	// structure so the stored tree always matches what was promoted.
	tag, err := tx.Exec(ctx, `
		UPDATE curriculum.upload_jobs
		SET status = 'approved', approved_by = $2, approved_at = now(),
		    parsed_structure = $3::jsonb, updated_at = now()
		WHERE id = $1 AND status IN ('parsed', 'review')
	`, jobID, userID, finalStructureJSON)
	if err != nil {
		return nil, fmt.Errorf("mark job approved: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Shouldn't happen given the FOR UPDATE check above, but guards
		// against a status change slipping in between.
		return nil, apperrors.Conflict("job status changed during approval; please retry")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &dto.ApproveResponse{
		JobID:          jobID,
		Status:         "approved",
		SubjectCode:    subjectCode,
		UnitsPromoted:  unitsPromoted,
		TopicsPromoted: topicsPromoted,
		ClosPromoted:   closPromoted,
	}, nil
}
