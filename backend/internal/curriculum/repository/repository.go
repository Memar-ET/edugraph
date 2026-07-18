package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

type Repository struct {
	pool  *pgxpool.Pool
	neo4j neo4jdriver.DriverWithContext
}

func New(pool *pgxpool.Pool, neo4j neo4jdriver.DriverWithContext) *Repository {
	return &Repository{pool: pool, neo4j: neo4j}
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

// promotedUnit/promotedTopic carry just enough of what got written to
// Postgres (the generated IDs) to mirror the same hierarchy into Neo4j
// afterwards, without a second round-trip to re-read it.
type promotedUnit struct {
	id      uuid.UUID
	number  int
	titleEn string
	topics  []promotedTopic
}

type promotedTopic struct {
	id            uuid.UUID
	sequenceOrder int
	titleEn       string
	keyConcepts   []string
}

// ApproveAndPromote is the core of Step 3/4: it locks the job row, verifies
// it's actually in a state that can be approved, promotes every unit/topic/
// CLO in `structure` into the real curriculum.* tables, marks the job
// 'approved' -- all in one Postgres transaction -- and then, once that's
// safely committed, mirrors the Subject/Unit/Topic hierarchy into Neo4j as
// the Knowledge Graph (Step 4's "Magic Moment").
//
// Units/topics/CLOs are upserted (ON CONFLICT ... DO UPDATE) keyed on their
// natural key (see migration V016), so approving the same job twice (e.g.
// after further edits) updates the existing rows rather than duplicating
// them, and the Neo4j MERGE calls are equally safe to repeat.
//
// The Neo4j sync happens *after* the Postgres commit, not inside the same
// transaction -- Neo4j isn't part of that transaction, so there's nothing
// to roll back there anyway. If it fails, the approval itself has still
// succeeded (Postgres is the source of truth for the review workflow);
// `curriculum.upload_jobs.neo4j_written` stays false so a failed sync is
// visible and can be retried by simply calling approve again (idempotent).
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
	defer func() { _ = tx.Rollback(ctx) }() // no-op once committed

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

	var topicsPromoted, closPromoted int
	promotedUnits := make([]promotedUnit, 0, len(structure.Units))
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

		pu := promotedUnit{id: unitID, number: u.Number, titleEn: u.TitleEn}

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

			pu.topics = append(pu.topics, promotedTopic{
				id: topicID, sequenceOrder: t.SequenceOrder, titleEn: t.TitleEn, keyConcepts: t.KeyConcepts,
			})

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

		promotedUnits = append(promotedUnits, pu)
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

	resp := &dto.ApproveResponse{
		JobID:          jobID,
		Status:         "approved",
		SubjectCode:    subjectCode,
		UnitsPromoted:  len(promotedUnits),
		TopicsPromoted: topicsPromoted,
		ClosPromoted:   closPromoted,
	}

	// Step 4: the relational data is now the source of truth (committed
	// above) -- mirror it into the Knowledge Graph. This is deliberately
	// outside the Postgres transaction: Neo4j can't participate in it, and
	// the approval itself must not be undone just because the graph sync
	// hiccuped. A failure here is recorded, not fatal to the request.
	if err := r.syncCurriculumGraph(ctx, subjectCode, gradeLevel, academicYear, promotedUnits); err != nil {
		resp.GraphSynced = false
		resp.GraphSyncError = err.Error()
		return resp, nil
	}

	if _, err := r.pool.Exec(ctx,
		`UPDATE curriculum.upload_jobs SET neo4j_written = true WHERE id = $1`, jobID,
	); err != nil {
		// The graph write itself succeeded; only the bookkeeping flag
		// failed to save. Still worth surfacing so it isn't silently lost.
		resp.GraphSynced = false
		resp.GraphSyncError = fmt.Sprintf("graph synced but failed to record neo4j_written: %v", err)
		return resp, nil
	}

	resp.GraphSynced = true
	return resp, nil
}

// syncCurriculumGraph mirrors an approved Subject -> Units -> Topics tree
// into Neo4j as :Subject/:Unit/:Topic nodes connected by :HAS_UNIT and
// :HAS_TOPIC relationships. Every write is a MERGE keyed on the same
// Postgres-generated id (or subject code), so calling this again for the
// same job -- e.g. retrying after a prior failure, simply by re-approving
// -- updates the existing nodes rather than duplicating them.
func (r *Repository) syncCurriculumGraph(
	ctx context.Context,
	subjectCode string,
	gradeLevel int,
	academicYear string,
	units []promotedUnit,
) error {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	for _, u := range units {
		_, err := session.Run(ctx, `
			MERGE (sub:Subject {code: $subjectCode})
			SET sub.gradeLevel = $gradeLevel, sub.academicYear = $academicYear
			MERGE (unit:Unit {id: $unitId})
			SET unit.number = $number, unit.titleEn = $titleEn, unit.subjectCode = $subjectCode
			MERGE (sub)-[:HAS_UNIT]->(unit)
		`, map[string]any{
			"subjectCode":  subjectCode,
			"gradeLevel":   int64(gradeLevel),
			"academicYear": academicYear,
			"unitId":       u.id.String(),
			"number":       int64(u.number),
			"titleEn":      u.titleEn,
		})
		if err != nil {
			return fmt.Errorf("sync unit %d to neo4j: %w", u.number, err)
		}

		for _, t := range u.topics {
			keyConcepts := t.keyConcepts
			if keyConcepts == nil {
				keyConcepts = []string{}
			}
			_, err := session.Run(ctx, `
				MATCH (unit:Unit {id: $unitId})
				MERGE (topic:Topic {id: $topicId})
				SET topic.titleEn = $titleEn,
				    topic.sequenceOrder = $sequenceOrder,
				    topic.keyConcepts = $keyConcepts,
				    topic.subjectCode = $subjectCode,
				    topic.gradeLevel = $gradeLevel
				MERGE (unit)-[:HAS_TOPIC]->(topic)
			`, map[string]any{
				"unitId":        u.id.String(),
				"topicId":       t.id.String(),
				"titleEn":       t.titleEn,
				"sequenceOrder": int64(t.sequenceOrder),
				"keyConcepts":   keyConcepts,
				"subjectCode":   subjectCode,
				"gradeLevel":    int64(gradeLevel),
			})
			if err != nil {
				return fmt.Errorf("sync topic %q (unit %d) to neo4j: %w", t.titleEn, u.number, err)
			}
		}
	}
	return nil
}
