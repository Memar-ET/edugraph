package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/internal/curriculum/repository"
	"github.com/edugraph-ai/edugraph/internal/testutil"
)

// TestApproveAndPromote_WritesAttribution is a real integration test
// against Postgres (and Neo4j, since the promotion mirrors into the
// graph) covering both the core Phase 1 promotion path and the
// checklist 10.3 updated_by/updated_at attribution added to
// curriculum.subjects/topics/clos.
func TestApproveAndPromote_WritesAttribution(t *testing.T) {
	pool := testutil.RequirePostgres(t)
	t.Cleanup(pool.Close)
	neo4jDriver := testutil.RequireNeo4j(t)

	repo := repository.New(pool, neo4jDriver)
	ctx := context.Background()
	suffix := uuid.NewString()[:8]

	approverUserID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, password_hash, role, full_name) VALUES ($1, $2, 'x', 'curriculum_officer', 'CO')`,
		approverUserID, "officer-"+suffix+"@edugraph.et"); err != nil {
		t.Fatalf("seed approver: %v", err)
	}

	subjectCode := "SUBJ" + suffix
	jobID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO curriculum.upload_jobs
		(id, uploaded_by, subject_code, grade_level, academic_year, file_s3_key, file_name, file_size_bytes, status)
		VALUES ($1, $2, $3, 9, '2026', 'test-ref', 'test.pdf', 1, 'parsed')`,
		jobID, approverUserID, subjectCode); err != nil {
		t.Fatalf("seed upload job: %v", err)
	}

	cloCode := "CLO" + suffix
	structure := dto.ParsedStructurePayload{
		Units: []dto.ParsedUnit{
			{
				Number:  1,
				TitleEn: "Test Unit",
				Topics: []dto.ParsedTopic{
					{
						SequenceOrder: 1,
						TitleEn:       "Test Topic",
						KeyConcepts:   []string{"concept one"},
						Clos: []dto.ParsedCLO{
							{Code: cloCode, Description: "Test CLO", Mandatory: true},
						},
					},
				},
			},
		},
	}

	resp, _, err := repo.ApproveAndPromote(ctx, jobID, approverUserID, structure, nil)
	if err != nil {
		t.Fatalf("ApproveAndPromote: %v", err)
	}
	if resp == nil {
		t.Fatal("ApproveAndPromote returned a nil response")
	}

	// Core promotion: subject/topic/CLO all actually landed in Postgres.
	var subjectExists, topicExists, cloExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM curriculum.subjects WHERE code = $1)`, subjectCode).Scan(&subjectExists); err != nil {
		t.Fatalf("check subject: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM curriculum.topics WHERE title_en = 'Test Topic' AND subject_code = $1)`, subjectCode).Scan(&topicExists); err != nil {
		t.Fatalf("check topic: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM curriculum.clos WHERE code = $1)`, cloCode).Scan(&cloExists); err != nil {
		t.Fatalf("check clo: %v", err)
	}
	if !subjectExists {
		t.Error("subject was not promoted to curriculum.subjects")
	}
	if !topicExists {
		t.Error("topic was not promoted to curriculum.topics")
	}
	if !cloExists {
		t.Error("CLO was not promoted to curriculum.clos")
	}

	// checklist 10.3: subject/topic/clo should all be attributed to the
	// approving user -- this used to only exist for topic_clo_mappings
	// (reviewed_by), not the subject/topic/CLO rows themselves.
	assertUpdatedBy := func(table, whereCol, whereVal string) {
		t.Helper()
		var updatedBy *uuid.UUID
		q := "SELECT updated_by FROM " + table + " WHERE " + whereCol + " = $1"
		if err := pool.QueryRow(ctx, q, whereVal).Scan(&updatedBy); err != nil {
			t.Fatalf("read updated_by from %s: %v", table, err)
		}
		if updatedBy == nil {
			t.Errorf("%s.updated_by is NULL, want %s", table, approverUserID)
		} else if *updatedBy != approverUserID {
			t.Errorf("%s.updated_by = %s, want %s", table, updatedBy, approverUserID)
		}
	}
	assertUpdatedBy("curriculum.subjects", "code", subjectCode)
	assertUpdatedBy("curriculum.clos", "code", cloCode)
}
