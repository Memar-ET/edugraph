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
// promoting: the job's current status and the previously-stored tree.
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

// PromotionResult carries the parts of an ApproveAndPromote call a caller
// might need beyond the HTTP-facing dto.ApproveResponse: the embedding
// jobs to queue, and a way to resolve ExternalCode back to generated topic UUIDs.
type PromotionResult struct {
	EmbeddingTargets []EmbeddingTarget
	TopicIDByCode    map[string]uuid.UUID
}

// EmbeddingTarget identifies one CLO or topic that needs a vector embedding refreshed.
type EmbeddingTarget struct {
	Kind    string // "topic" or "clo"
	TopicID uuid.UUID
	CloCode string
}

// --- Internal flat types for batch processing ---

// flatUnit is a unit extracted from the parsed structure for batch upsert.
type flatUnit struct {
	number  int
	titleEn string
}

// flatTopic represents one topic or subtopic for batch upsert.
// parentTopicIdx is -1 for top-level topics, or an index into the same slice for subtopics.
// DFS order guarantees parent index < child index.
type flatTopic struct {
	unitIdx        int
	parentTopicIdx int // -1 = top-level
	seqOrder       int
	titleEn        string
	rawText        string
	keyConcepts    []string
	externalCode   string
}

// flatCLO is a deduplicated CLO for batch upsert.
type flatCLO struct {
	code        string
	description string
	bloomLevel  string
	mandatory   bool
}

// flatMapping is a topic→CLO mapping for batch upsert.
type flatMapping struct {
	topicIdx    int
	cloCode     string
	matchMethod string // "human_confirmed" or "manual"
}

// flattenStructure walks the parsed structure and produces flat, indexed lists
// suitable for batched DB writes with no per-row round-trips during traversal.
// CLOs are deduplicated by code; unit-level CLOs are mapped to every topic and
// subtopic in their unit (matchMethod "manual") just as the old per-row path did.
func flattenStructure(structure dto.ParsedStructurePayload) (
	units []flatUnit,
	topics []flatTopic,
	clos []flatCLO,
	mappings []flatMapping,
) {
	cloByCode := make(map[string]int) // code → index in clos

	ensureCLO := func(c dto.ParsedCLO) {
		if c.Code == "" {
			return
		}
		if _, ok := cloByCode[c.Code]; !ok {
			cloByCode[c.Code] = len(clos)
			clos = append(clos, flatCLO{
				code:        c.Code,
				description: c.Description,
				bloomLevel:  c.BloomLevel,
				mandatory:   c.Mandatory,
			})
		}
	}

	addMapping := func(topicIdx int, cloCode, matchMethod string) {
		if cloCode == "" {
			return
		}
		mappings = append(mappings, flatMapping{topicIdx: topicIdx, cloCode: cloCode, matchMethod: matchMethod})
	}

	// addTopic appends t (and all its subtopics recursively) to topics and
	// records mappings for each CLO. Returns the index of t in topics.
	var addTopic func(unitIdx, parentTopicIdx int, t dto.ParsedTopic) int
	addTopic = func(unitIdx, parentTopicIdx int, t dto.ParsedTopic) int {
		myIdx := len(topics)
		topics = append(topics, flatTopic{
			unitIdx:        unitIdx,
			parentTopicIdx: parentTopicIdx,
			seqOrder:       t.SequenceOrder,
			titleEn:        t.TitleEn,
			rawText:        t.RawText,
			keyConcepts:    t.KeyConcepts,
			externalCode:   t.ExternalCode,
		})
		for _, c := range t.Clos {
			ensureCLO(c)
			addMapping(myIdx, c.Code, "human_confirmed")
		}
		for _, st := range t.Subtopics {
			addTopic(unitIdx, myIdx, st)
		}
		return myIdx
	}

	for _, u := range structure.Units {
		unitIdx := len(units)
		units = append(units, flatUnit{number: u.Number, titleEn: u.TitleEn})

		// Track which topic indices belong to this unit for unit-level CLO mapping.
		startTopicIdx := len(topics)
		for _, t := range u.Topics {
			addTopic(unitIdx, -1, t)
		}
		endTopicIdx := len(topics)

		// Unit-level CLOs mapped to every topic and subtopic in this unit.
		for _, c := range u.UnitClos {
			ensureCLO(c)
			for i := startTopicIdx; i < endTopicIdx; i++ {
				addMapping(i, c.Code, "manual")
			}
		}
	}

	return units, topics, clos, mappings
}

// ApproveAndPromote is the core of Steps 3/4: it locks the job row, verifies
// it's in an approvable state, promotes every unit/topic/CLO in `structure`
// into the real curriculum.* tables, marks the job 'approved' -- all in one
// Postgres transaction -- and then mirrors the hierarchy into Neo4j.
//
// This implementation uses pgx.Batch to group all Postgres writes of the same
// type into single round-trips (units batch, level-0 topics batch, subtopics
// batch, CLOs batch, mappings batch), reducing network round-trips from
// O(topics × CLOs) to a fixed ~7 regardless of curriculum size. Neo4j writes
// use UNWIND-based Cypher (one query per node/edge type).
//
// The Neo4j sync happens after the Postgres commit. If it fails, the approval
// itself has still succeeded; `neo4j_written` stays false so the failure is
// visible and the sync can be retried by re-approving.
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
	defer func() { _ = tx.Rollback(ctx) }()

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
	moeVersion := fmt.Sprintf("ai-draft-%s", academicYear)

	// 1. Subject upsert (single row, needed before unit FK inserts).
	if _, err := tx.Exec(ctx, `
		INSERT INTO curriculum.subjects (code, name_en, grade_level, academic_year, upload_job_id, updated_by)
		VALUES ($1, $1, $2, $3, $4, $5)
		ON CONFLICT (code) DO UPDATE SET
			grade_level   = EXCLUDED.grade_level,
			academic_year = EXCLUDED.academic_year,
			upload_job_id = EXCLUDED.upload_job_id,
			updated_by    = EXCLUDED.updated_by,
			updated_at    = now()
	`, subjectCode, gradeLevel, academicYear, jobID, userID); err != nil {
		return nil, nil, fmt.Errorf("upsert subject %q: %w", subjectCode, err)
	}

	// 2. Flatten structure -- pure, no DB calls.
	flatUnits, flatTopics, flatCLOs, flatMappings := flattenStructure(structure)

	// 3. Batch-upsert units (1 round-trip, returns IDs).
	unitIDs, err := r.batchUpsertUnits(ctx, tx, subjectCode, gradeLevel, flatUnits)
	if err != nil {
		return nil, nil, err
	}

	// 4+5. Batch-upsert topics level-by-level (2 round-trips for 2-level hierarchy).
	topicIDs, err := r.batchUpsertTopics(ctx, tx, subjectCode, gradeLevel, userID, flatTopics, unitIDs)
	if err != nil {
		return nil, nil, err
	}

	// 6. Batch-upsert CLOs (1 round-trip, no IDs needed).
	if err := r.batchUpsertCLOs(ctx, tx, subjectCode, gradeLevel, moeVersion, userID, flatCLOs); err != nil {
		return nil, nil, err
	}

	// 7. Batch-upsert topic_clo_mappings (1 round-trip, no IDs needed).
	if err := r.batchUpsertMappings(ctx, tx, userID, flatMappings, topicIDs); err != nil {
		return nil, nil, err
	}

	// 8. Mark job approved.
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
		return nil, nil, apperrors.Conflict("job status changed during approval; please retry")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit tx: %w", err)
	}

	// Build result structs.
	embeddingTargets := make([]EmbeddingTarget, 0, len(flatTopics)+len(flatCLOs))
	for _, id := range topicIDs {
		embeddingTargets = append(embeddingTargets, EmbeddingTarget{Kind: "topic", TopicID: id})
	}
	for _, c := range flatCLOs {
		embeddingTargets = append(embeddingTargets, EmbeddingTarget{Kind: "clo", CloCode: c.code})
	}

	topicByCode := make(map[string]uuid.UUID)
	for i, ft := range flatTopics {
		if ft.externalCode != "" {
			topicByCode[ft.externalCode] = topicIDs[i]
		}
	}

	resp := &dto.ApproveResponse{
		JobID:          jobID,
		Status:         "approved",
		SubjectCode:    subjectCode,
		UnitsPromoted:  len(flatUnits),
		TopicsPromoted: len(flatTopics),
		ClosPromoted:   len(flatMappings),
	}
	result := &PromotionResult{
		EmbeddingTargets: embeddingTargets,
		TopicIDByCode:    topicByCode,
	}

	// 9. Neo4j UNWIND sync (post-commit, best-effort).
	if err := r.syncCurriculumGraphUnwind(ctx, subjectCode, gradeLevel, academicYear, flatUnits, flatTopics, flatCLOs, flatMappings, unitIDs, topicIDs); err != nil {
		resp.GraphSynced = false
		resp.GraphSyncError = err.Error()
		return resp, result, nil
	}

	if _, err := r.pool.Exec(ctx,
		`UPDATE curriculum.upload_jobs SET neo4j_written = true WHERE id = $1`, jobID,
	); err != nil {
		resp.GraphSynced = false
		resp.GraphSyncError = fmt.Sprintf("graph synced but failed to record neo4j_written: %v", err)
		return resp, result, nil
	}

	resp.GraphSynced = true
	return resp, result, nil
}

// batchUpsertUnits sends all unit upserts in a single round-trip and returns
// the generated IDs in the same order as the input slice.
func (r *Repository) batchUpsertUnits(ctx context.Context, tx pgx.Tx, subjectCode string, gradeLevel int, units []flatUnit) ([]uuid.UUID, error) {
	if len(units) == 0 {
		return nil, nil
	}
	batch := &pgx.Batch{}
	for _, u := range units {
		batch.Queue(`
			INSERT INTO curriculum.units (subject_code, grade_level, number, title_en)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (subject_code, number) DO UPDATE SET title_en = EXCLUDED.title_en
			RETURNING id
		`, subjectCode, gradeLevel, u.number, u.titleEn)
	}
	br := tx.SendBatch(ctx, batch)
	defer br.Close()
	ids := make([]uuid.UUID, len(units))
	for i := range units {
		if err := br.QueryRow().Scan(&ids[i]); err != nil {
			return nil, fmt.Errorf("upsert unit %d %q: %w", units[i].number, units[i].titleEn, err)
		}
	}
	return ids, nil
}

// batchUpsertTopics upserts topics level by level (parents before children)
// using pgx.Batch per level, returning IDs aligned with the input slice.
// Topics are processed in waves: those with no parent, then those whose parent
// is already inserted, and so on. In practice this is always 2 waves (top-level
// topics + subtopics), but the loop handles arbitrary depth correctly.
func (r *Repository) batchUpsertTopics(
	ctx context.Context, tx pgx.Tx,
	subjectCode string, gradeLevel int, userID uuid.UUID,
	topics []flatTopic, unitIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	if len(topics) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, len(topics))
	inserted := make([]bool, len(topics))
	remaining := len(topics)

	for remaining > 0 {
		// Collect all topics ready to insert this wave.
		ready := make([]int, 0, remaining)
		for i, ft := range topics {
			if !inserted[i] && (ft.parentTopicIdx < 0 || inserted[ft.parentTopicIdx]) {
				ready = append(ready, i)
			}
		}
		if len(ready) == 0 {
			return nil, fmt.Errorf("cycle or orphan in topic structure (%d topics remain)", remaining)
		}

		batch := &pgx.Batch{}
		for _, i := range ready {
			ft := topics[i]
			var parentID *uuid.UUID
			if ft.parentTopicIdx >= 0 {
				p := ids[ft.parentTopicIdx]
				parentID = &p
			}
			kc := ft.keyConcepts
			if kc == nil {
				kc = []string{}
			}
			batch.Queue(`
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
			`, unitIDs[ft.unitIdx], subjectCode, gradeLevel, ft.seqOrder, ft.titleEn, nullIfEmpty(ft.rawText), kc, parentID, userID)
		}

		br := tx.SendBatch(ctx, batch)
		for _, i := range ready {
			if err := br.QueryRow().Scan(&ids[i]); err != nil {
				_ = br.Close()
				return nil, fmt.Errorf("upsert topic %q: %w", topics[i].titleEn, err)
			}
			inserted[i] = true
			remaining--
		}
		if err := br.Close(); err != nil {
			return nil, fmt.Errorf("topic batch close: %w", err)
		}
	}

	return ids, nil
}

// batchUpsertCLOs sends all CLO upserts in a single round-trip.
func (r *Repository) batchUpsertCLOs(
	ctx context.Context, tx pgx.Tx,
	subjectCode string, gradeLevel int, moeVersion string, userID uuid.UUID,
	clos []flatCLO,
) error {
	if len(clos) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, c := range clos {
		batch.Queue(`
			INSERT INTO curriculum.clos
				(code, subject_code, grade_level, description_en, bloom_level, is_mandatory, moe_version, updated_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (code) DO UPDATE SET
				description_en = EXCLUDED.description_en,
				bloom_level    = EXCLUDED.bloom_level,
				is_mandatory   = EXCLUDED.is_mandatory,
				updated_by     = EXCLUDED.updated_by,
				updated_at     = now()
		`, c.code, subjectCode, gradeLevel, c.description, nullIfEmpty(c.bloomLevel), c.mandatory, moeVersion, userID)
	}
	br := tx.SendBatch(ctx, batch)
	defer br.Close()
	for _, c := range clos {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("upsert clo %q: %w", c.code, err)
		}
	}
	return nil
}

// batchUpsertMappings sends all topic_clo_mapping upserts in a single round-trip.
func (r *Repository) batchUpsertMappings(
	ctx context.Context, tx pgx.Tx,
	userID uuid.UUID,
	mappings []flatMapping,
	topicIDs []uuid.UUID,
) error {
	if len(mappings) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, m := range mappings {
		batch.Queue(`
			INSERT INTO curriculum.topic_clo_mappings (topic_id, clo_code, match_method, reviewed_by, confirmed_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (topic_id, clo_code) DO UPDATE SET
				match_method = EXCLUDED.match_method,
				reviewed_by  = EXCLUDED.reviewed_by,
				confirmed_at = now()
		`, topicIDs[m.topicIdx], m.cloCode, m.matchMethod, userID)
	}
	br := tx.SendBatch(ctx, batch)
	defer br.Close()
	for _, m := range mappings {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("upsert mapping topic[%d]→%q: %w", m.topicIdx, m.cloCode, err)
		}
	}
	return nil
}

// syncCurriculumGraphUnwind mirrors an approved subject's hierarchy into Neo4j
// using UNWIND-based Cypher (one query per node/edge type) rather than per-row
// session.Run calls, reducing Neo4j round-trips from O(N) to 4.
func (r *Repository) syncCurriculumGraphUnwind(
	ctx context.Context,
	subjectCode string, gradeLevel int, academicYear string,
	flatUnits []flatUnit, flatTopics []flatTopic,
	flatCLOs []flatCLO, flatMappings []flatMapping,
	unitIDs []uuid.UUID, topicIDs []uuid.UUID,
) error {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	// 1. Subject + all units (single UNWIND query).
	neoUnits := make([]map[string]any, len(flatUnits))
	for i, u := range flatUnits {
		neoUnits[i] = map[string]any{
			"id":      unitIDs[i].String(),
			"number":  int64(u.number),
			"titleEn": u.titleEn,
		}
	}
	if _, err := session.Run(ctx, `
		MERGE (sub:Subject {code: $subjectCode})
		SET sub.gradeLevel = $gradeLevel, sub.academicYear = $academicYear
		WITH sub
		UNWIND $units AS u
		MERGE (unit:Unit {id: u.id})
		SET unit.number = u.number, unit.titleEn = u.titleEn, unit.subjectCode = $subjectCode
		MERGE (sub)-[:HAS_UNIT]->(unit)
	`, map[string]any{
		"subjectCode":  subjectCode,
		"gradeLevel":   int64(gradeLevel),
		"academicYear": academicYear,
		"units":        neoUnits,
	}); err != nil {
		return fmt.Errorf("sync units to neo4j: %w", err)
	}

	// 2. All topics -- unit→topic edges (single UNWIND query).
	if len(flatTopics) > 0 {
		neoTopics := make([]map[string]any, len(flatTopics))
		for i, ft := range flatTopics {
			kc := ft.keyConcepts
			if kc == nil {
				kc = []string{}
			}
			neoTopics[i] = map[string]any{
				"id":          topicIDs[i].String(),
				"unitId":      unitIDs[ft.unitIdx].String(),
				"titleEn":     ft.titleEn,
				"seqOrder":    int64(ft.seqOrder),
				"keyConcepts": kc,
			}
		}
		if _, err := session.Run(ctx, `
			UNWIND $topics AS t
			MATCH (unit:Unit {id: t.unitId})
			MERGE (topic:Topic {id: t.id})
			SET topic.titleEn = t.titleEn,
			    topic.sequenceOrder = t.seqOrder,
			    topic.keyConcepts = t.keyConcepts,
			    topic.subjectCode = $subjectCode,
			    topic.gradeLevel = $gradeLevel
			MERGE (unit)-[:HAS_TOPIC]->(topic)
		`, map[string]any{
			"topics":      neoTopics,
			"subjectCode": subjectCode,
			"gradeLevel":  int64(gradeLevel),
		}); err != nil {
			return fmt.Errorf("sync topics to neo4j: %w", err)
		}
	}

	// 3. Parent→subtopic edges (single UNWIND query, skipped if no subtopics).
	var subtopicEdges []map[string]any
	for i, ft := range flatTopics {
		if ft.parentTopicIdx >= 0 {
			subtopicEdges = append(subtopicEdges, map[string]any{
				"parentId": topicIDs[ft.parentTopicIdx].String(),
				"childId":  topicIDs[i].String(),
			})
		}
	}
	if len(subtopicEdges) > 0 {
		if _, err := session.Run(ctx, `
			UNWIND $edges AS e
			MATCH (parent:Topic {id: e.parentId})
			MATCH (child:Topic {id: e.childId})
			MERGE (parent)-[:HAS_SUBTOPIC]->(child)
		`, map[string]any{"edges": subtopicEdges}); err != nil {
			return fmt.Errorf("sync subtopic edges to neo4j: %w", err)
		}
	}

	// 4. CLO nodes + topic→CLO edges (single UNWIND query).
	if len(flatMappings) > 0 {
		cloDescByCode := make(map[string]string, len(flatCLOs))
		for _, c := range flatCLOs {
			cloDescByCode[c.code] = c.description
		}
		cloEdges := make([]map[string]any, 0, len(flatMappings))
		for _, m := range flatMappings {
			if m.cloCode == "" {
				continue
			}
			cloEdges = append(cloEdges, map[string]any{
				"topicId":     topicIDs[m.topicIdx].String(),
				"code":        m.cloCode,
				"description": cloDescByCode[m.cloCode],
			})
		}
		if len(cloEdges) > 0 {
			if _, err := session.Run(ctx, `
				UNWIND $edges AS e
				MATCH (topic:Topic {id: e.topicId})
				MERGE (clo:CLO {code: e.code})
				SET clo.description = e.description
				MERGE (topic)-[:HAS_CLO]->(clo)
			`, map[string]any{"edges": cloEdges}); err != nil {
				return fmt.Errorf("sync clo edges to neo4j: %w", err)
			}
		}
	}

	return nil
}
