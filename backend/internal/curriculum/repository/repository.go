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
	parentID      *uuid.UUID // set for a promoted subtopic, nil for a top-level topic
	externalCode  string     // caller-supplied dto.ParsedTopic.ExternalCode, "" if unset
	sequenceOrder int
	titleEn       string
	keyConcepts   []string
	clos          []promotedCLO // this topic's own CLOs plus every unit-level CLO (mapped to every topic in the unit)
	subtopics     []promotedTopic
}

type promotedCLO struct {
	code        string
	description string
}

// PromotionResult carries the parts of an ApproveAndPromote call a caller
// might need beyond the HTTP-facing dto.ApproveResponse: the embedding
// jobs to queue (feature 1.1), and -- for callers that supplied
// dto.ParsedTopic.ExternalCode, e.g. a bulk source-document importer --
// a way to resolve those external codes back to the generated topic
// UUIDs (e.g. to bulk-insert prerequisite edges keyed on the source
// document's own topic IDs). Topics promoted without an ExternalCode
// (the normal AI-parsed upload flow) simply aren't in the map.
type PromotionResult struct {
	EmbeddingTargets []EmbeddingTarget
	TopicIDByCode    map[string]uuid.UUID
}

// EmbeddingTarget identifies one CLO or topic that was just promoted (or
// re-promoted) and needs a vector embedding generated/refreshed -- see
// ApproveAndPromote and Service.Approve, which turns these into
// queue:embedding:generate jobs for the ai-service embed worker (feature
// 1.1). Only Kind is exported; only one of TopicID/CloCode is set,
// matching Kind.
type EmbeddingTarget struct {
	Kind    string // "topic" or "clo"
	TopicID uuid.UUID
	CloCode string
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
) (*dto.ApproveResponse, *PromotionResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op once committed

	// Lock the row and re-check status inside the transaction -- this is
	// the authoritative check (avoids a race between two concurrent
	// approve requests for the same job).
	core, err := r.getJobCore(ctx, tx, jobID, true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, apperrors.NotFound("job not found")
		}
		return nil, nil, fmt.Errorf("lock job: %w", err)
	}
	if core.Status != "parsed" && core.Status != "review" {
		return nil, nil, apperrors.Conflict(fmt.Sprintf(
			"job status is %q; only jobs with status 'parsed' or 'review' can be approved", core.Status,
		))
	}

	subjectCode, gradeLevel, academicYear := core.SubjectCode, core.GradeLevel, core.AcademicYear

	// 1. Subject (upsert -- name_en has no better source yet than the code
	// itself; officers can rename it later via a subjects endpoint).
	// updated_by/updated_at (V031, checklist 10.3): re-approving the same
	// code is a real edit (ON CONFLICT DO UPDATE), so it needs the same
	// attribution as topics/clos below, not just created_at.
	_, err = tx.Exec(ctx, `
		INSERT INTO curriculum.subjects (code, name_en, grade_level, academic_year, upload_job_id, updated_by)
		VALUES ($1, $1, $2, $3, $4, $5)
		ON CONFLICT (code) DO UPDATE SET
			grade_level   = EXCLUDED.grade_level,
			academic_year = EXCLUDED.academic_year,
			upload_job_id = EXCLUDED.upload_job_id,
			updated_by    = EXCLUDED.updated_by,
			updated_at    = now()
	`, subjectCode, gradeLevel, academicYear, jobID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("upsert subject %q: %w", subjectCode, err)
	}

	var topicsPromoted, closPromoted int
	promotedUnits := make([]promotedUnit, 0, len(structure.Units))
	moeVersion := fmt.Sprintf("ai-draft-%s", academicYear)
	embeddingTargets := make([]EmbeddingTarget, 0)
	closEmbedded := make(map[string]bool)

	for _, u := range structure.Units {
		var unitID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO curriculum.units (subject_code, grade_level, number, title_en)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (subject_code, number) DO UPDATE SET title_en = EXCLUDED.title_en
			RETURNING id
		`, subjectCode, gradeLevel, u.Number, u.TitleEn).Scan(&unitID)
		if err != nil {
			return nil, nil, fmt.Errorf("upsert unit %d (%q): %w", u.Number, u.TitleEn, err)
		}

		pu := promotedUnit{id: unitID, number: u.Number, titleEn: u.TitleEn}

		for _, t := range u.Topics {
			pt, tp, cp, err := r.promoteOneTopic(
				ctx, tx, unitID, subjectCode, gradeLevel, nil, t, userID, moeVersion,
				closEmbedded, &embeddingTargets,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("promote topic %q (unit %d): %w", t.TitleEn, u.Number, err)
			}
			pu.topics = append(pu.topics, pt)
			topicsPromoted += tp
			closPromoted += cp
		}

		// Unit-level CLOs ("Unit Outcomes" in the source, not tied to one
		// topic) are mapped to every topic and subtopic promoted under this
		// unit, since curriculum.clos has no unit_id column of its own.
		for _, c := range u.UnitClos {
			if c.Code == "" {
				continue
			}
			if err := r.upsertCLO(ctx, tx, subjectCode, gradeLevel, moeVersion, c, userID, closEmbedded, &embeddingTargets); err != nil {
				return nil, nil, err
			}
			n, err := r.mapCLOToAllTopics(ctx, tx, pu.topics, c, userID)
			if err != nil {
				return nil, nil, err
			}
			closPromoted += n
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
		return nil, nil, fmt.Errorf("mark job approved: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Shouldn't happen given the FOR UPDATE check above, but guards
		// against a status change slipping in between.
		return nil, nil, apperrors.Conflict("job status changed during approval; please retry")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit tx: %w", err)
	}

	resp := &dto.ApproveResponse{
		JobID:          jobID,
		Status:         "approved",
		SubjectCode:    subjectCode,
		UnitsPromoted:  len(promotedUnits),
		TopicsPromoted: topicsPromoted,
		ClosPromoted:   closPromoted,
	}
	result := &PromotionResult{
		EmbeddingTargets: embeddingTargets,
		TopicIDByCode:    topicIDByCode(promotedUnits),
	}

	// Step 4: the relational data is now the source of truth (committed
	// above) -- mirror it into the Knowledge Graph. This is deliberately
	// outside the Postgres transaction: Neo4j can't participate in it, and
	// the approval itself must not be undone just because the graph sync
	// hiccuped. A failure here is recorded, not fatal to the request.
	if err := r.syncCurriculumGraph(ctx, subjectCode, gradeLevel, academicYear, promotedUnits); err != nil {
		resp.GraphSynced = false
		resp.GraphSyncError = err.Error()
		return resp, result, nil
	}

	if _, err := r.pool.Exec(ctx,
		`UPDATE curriculum.upload_jobs SET neo4j_written = true WHERE id = $1`, jobID,
	); err != nil {
		// The graph write itself succeeded; only the bookkeeping flag
		// failed to save. Still worth surfacing so it isn't silently lost.
		resp.GraphSynced = false
		resp.GraphSyncError = fmt.Sprintf("graph synced but failed to record neo4j_written: %v", err)
		return resp, result, nil
	}

	resp.GraphSynced = true
	return resp, result, nil
}

// topicIDByCode walks every promoted topic (and its subtopics) across all
// units and collects a code -> id map for whichever ones the caller
// tagged with ExternalCode -- see PromotionResult.
func topicIDByCode(units []promotedUnit) map[string]uuid.UUID {
	out := make(map[string]uuid.UUID)
	var walk func(topics []promotedTopic)
	walk = func(topics []promotedTopic) {
		for _, t := range topics {
			if t.externalCode != "" {
				out[t.externalCode] = t.id
			}
			walk(t.subtopics)
		}
	}
	for _, u := range units {
		walk(u.topics)
	}
	return out
}

// promoteOneTopic upserts a single topic (or subtopic, when parentID is
// non-nil) plus its own CLOs, returning enough (id, its CLOs, its
// promoted subtopics) to mirror it into Neo4j and to map unit-level CLOs
// onto it afterwards. Recurses one level for t.Subtopics -- see
// dto.ParsedTopic's doc comment on why the recursive shape exists despite
// nothing going deeper than one level in practice.
func (r *Repository) promoteOneTopic(
	ctx context.Context, tx pgx.Tx,
	unitID uuid.UUID, subjectCode string, gradeLevel int, parentID *uuid.UUID,
	t dto.ParsedTopic, userID uuid.UUID, moeVersion string,
	closEmbedded map[string]bool, embeddingTargets *[]EmbeddingTarget,
) (promotedTopic, int, int, error) {
	var topicID uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO curriculum.topics
			(unit_id, subject_code, grade_level, sequence_order, title_en, description, key_concepts, parent_topic_id, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (unit_id, sequence_order) DO UPDATE SET
			title_en        = EXCLUDED.title_en,
			description     = EXCLUDED.description,
			key_concepts    = EXCLUDED.key_concepts,
			parent_topic_id = EXCLUDED.parent_topic_id,
			updated_by      = EXCLUDED.updated_by,
			updated_at      = now()
		RETURNING id
	`, unitID, subjectCode, gradeLevel, t.SequenceOrder, t.TitleEn, nullIfEmpty(t.RawText), t.KeyConcepts, parentID, userID).Scan(&topicID)
	if err != nil {
		return promotedTopic{}, 0, 0, fmt.Errorf("upsert topic %q: %w", t.TitleEn, err)
	}

	*embeddingTargets = append(*embeddingTargets, EmbeddingTarget{Kind: "topic", TopicID: topicID})
	pt := promotedTopic{
		id: topicID, parentID: parentID, externalCode: t.ExternalCode,
		sequenceOrder: t.SequenceOrder, titleEn: t.TitleEn, keyConcepts: t.KeyConcepts,
	}
	topicsPromoted, closPromoted := 1, 0

	for _, c := range t.Clos {
		if c.Code == "" {
			continue // can't upsert a CLO without its natural key
		}
		if err := r.upsertCLO(ctx, tx, subjectCode, gradeLevel, moeVersion, c, userID, closEmbedded, embeddingTargets); err != nil {
			return promotedTopic{}, 0, 0, err
		}
		if err := r.upsertTopicCLOMapping(ctx, tx, topicID, c.Code, userID, "human_confirmed"); err != nil {
			return promotedTopic{}, 0, 0, err
		}
		pt.clos = append(pt.clos, promotedCLO{code: c.Code, description: c.Description})
		closPromoted++
	}

	for _, st := range t.Subtopics {
		spt, stp, scp, err := r.promoteOneTopic(ctx, tx, unitID, subjectCode, gradeLevel, &topicID, st, userID, moeVersion, closEmbedded, embeddingTargets)
		if err != nil {
			return promotedTopic{}, 0, 0, err
		}
		pt.subtopics = append(pt.subtopics, spt)
		topicsPromoted += stp
		closPromoted += scp
	}

	return pt, topicsPromoted, closPromoted, nil
}

// upsertCLO upserts one curriculum.clos row and registers it for
// embedding generation at most once per ApproveAndPromote call (dedup via
// closEmbedded) -- it does NOT touch topic_clo_mappings, since a CLO can
// be mapped to several topics (a unit-level CLO is mapped to every topic
// in the unit) via separate calls to upsertTopicCLOMapping.
func (r *Repository) upsertCLO(
	ctx context.Context, tx pgx.Tx, subjectCode string, gradeLevel int, moeVersion string,
	c dto.ParsedCLO, userID uuid.UUID, closEmbedded map[string]bool, embeddingTargets *[]EmbeddingTarget,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO curriculum.clos
			(code, subject_code, grade_level, description_en, bloom_level, is_mandatory, moe_version, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (code) DO UPDATE SET
			description_en = EXCLUDED.description_en,
			bloom_level    = EXCLUDED.bloom_level,
			is_mandatory   = EXCLUDED.is_mandatory,
			updated_by     = EXCLUDED.updated_by,
			updated_at     = now()
	`, c.Code, subjectCode, gradeLevel, c.Description, nullIfEmpty(c.BloomLevel), c.Mandatory, moeVersion, userID)
	if err != nil {
		return fmt.Errorf("upsert clo %q: %w", c.Code, err)
	}
	if !closEmbedded[c.Code] {
		closEmbedded[c.Code] = true
		*embeddingTargets = append(*embeddingTargets, EmbeddingTarget{Kind: "clo", CloCode: c.Code})
	}
	return nil
}

func (r *Repository) upsertTopicCLOMapping(ctx context.Context, tx pgx.Tx, topicID uuid.UUID, cloCode string, userID uuid.UUID, matchMethod string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO curriculum.topic_clo_mappings (topic_id, clo_code, match_method, reviewed_by, confirmed_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (topic_id, clo_code) DO UPDATE SET
			match_method = EXCLUDED.match_method,
			reviewed_by  = EXCLUDED.reviewed_by,
			confirmed_at = now()
	`, topicID, cloCode, matchMethod, userID)
	if err != nil {
		return fmt.Errorf("upsert topic_clo_mapping %q: %w", cloCode, err)
	}
	return nil
}

// mapCLOToAllTopics maps a unit-level CLO to every topic and subtopic in
// `topics` (match_method "manual", distinguishing it from a direct
// topic-CLO pairing) and mirrors it into each one's .clos so the Neo4j
// sync sees it too. Mutates `topics` in place through the shared backing
// array -- callers don't need to reassign the slice they passed in.
func (r *Repository) mapCLOToAllTopics(ctx context.Context, tx pgx.Tx, topics []promotedTopic, c dto.ParsedCLO, userID uuid.UUID) (int, error) {
	count := 0
	for i := range topics {
		if err := r.upsertTopicCLOMapping(ctx, tx, topics[i].id, c.Code, userID, "manual"); err != nil {
			return 0, err
		}
		topics[i].clos = append(topics[i].clos, promotedCLO{code: c.Code, description: c.Description})
		count++

		if len(topics[i].subtopics) > 0 {
			n, err := r.mapCLOToAllTopics(ctx, tx, topics[i].subtopics, c, userID)
			if err != nil {
				return 0, err
			}
			count += n
		}
	}
	return count, nil
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
			if err := r.syncTopicToNeo4j(ctx, session, u.id, subjectCode, gradeLevel, t); err != nil {
				return fmt.Errorf("sync topic %q (unit %d): %w", t.titleEn, u.number, err)
			}
		}
	}
	return nil
}

// syncTopicToNeo4j mirrors one topic (or subtopic) plus its CLOs, then
// recurses into its own subtopics. Every promoted topic -- top-level or
// subtopic -- gets a direct (:Unit)-[:HAS_TOPIC]->(:Topic) edge (so
// existing unit->topic traversals, e.g. the class heatmap, see subtopics
// too without a schema-aware rewrite); a subtopic additionally gets
// (:Topic)-[:HAS_SUBTOPIC]->(:Topic) from its parent, mirroring
// parent_topic_id (V027). CLOs -- topic-level and unit-level alike, both
// already flattened into promotedTopic.clos by ApproveAndPromote -- are
// mirrored as (:Topic)-[:HAS_CLO]->(:CLO), closing the gap noted in
// CLAUDE.md ("CLO Neo4j sync deferred to Phase 2") for the topics/CLOs
// this function actually touches.
func (r *Repository) syncTopicToNeo4j(
	ctx context.Context, session neo4jdriver.SessionWithContext,
	unitID uuid.UUID, subjectCode string, gradeLevel int, t promotedTopic,
) error {
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
		"unitId":        unitID.String(),
		"topicId":       t.id.String(),
		"titleEn":       t.titleEn,
		"sequenceOrder": int64(t.sequenceOrder),
		"keyConcepts":   keyConcepts,
		"subjectCode":   subjectCode,
		"gradeLevel":    int64(gradeLevel),
	})
	if err != nil {
		return fmt.Errorf("sync topic node: %w", err)
	}

	if t.parentID != nil {
		if _, err := session.Run(ctx, `
			MATCH (parent:Topic {id: $parentId})
			MATCH (topic:Topic {id: $topicId})
			MERGE (parent)-[:HAS_SUBTOPIC]->(topic)
		`, map[string]any{"parentId": t.parentID.String(), "topicId": t.id.String()}); err != nil {
			return fmt.Errorf("sync subtopic relationship: %w", err)
		}
	}

	for _, c := range t.clos {
		if _, err := session.Run(ctx, `
			MATCH (topic:Topic {id: $topicId})
			MERGE (clo:CLO {code: $code})
			SET clo.description = $description
			MERGE (topic)-[:HAS_CLO]->(clo)
		`, map[string]any{
			"topicId":     t.id.String(),
			"code":        c.code,
			"description": c.description,
		}); err != nil {
			return fmt.Errorf("sync clo %q: %w", c.code, err)
		}
	}

	for _, st := range t.subtopics {
		if err := r.syncTopicToNeo4j(ctx, session, unitID, subjectCode, gradeLevel, st); err != nil {
			return err
		}
	}
	return nil
}
