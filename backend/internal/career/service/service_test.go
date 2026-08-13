package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/edugraph-ai/edugraph/internal/career/repository"
	"github.com/edugraph-ai/edugraph/internal/career/service"
	"github.com/edugraph-ai/edugraph/internal/testutil"
)

// TestMatches_ScopedToCallerOwnStudentRecord is the checklist 10.1/11.3
// finding: GenerateMatches/Matches used to take a studentID straight
// from the URL -- any authenticated account could read or trigger
// generation for someone else's career matches. Now the student is
// always resolved from the caller's own JWT userID
// (repo.StudentIDByUserID), never trusted from a request.
func TestMatches_ScopedToCallerOwnStudentRecord(t *testing.T) {
	pool := testutil.RequirePostgres(t)
	t.Cleanup(pool.Close)
	// aiClient is nil deliberately: Matches (unlike GenerateMatches)
	// never calls it, and this test only exercises Matches/the
	// studentIDForUser resolution the IDOR fix depends on.
	var neo4jDriver neo4jdriver.DriverWithContext
	svc := service.New(repository.New(pool, neo4jDriver), nil)
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	regionID, schoolID := uuid.NewString(), uuid.NewString()
	studentUserID := uuid.NewString()
	nonStudentUserID := uuid.NewString() // a real account, but with no students row at all

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("seed fixture: %v", err)
		}
	}
	mustExec(`INSERT INTO regions (id, name, code) VALUES ($1, $2, $3)`, regionID, "R "+suffix, "R"+suffix)
	mustExec(`INSERT INTO schools (id, region_id, name, code) VALUES ($1, $2, $3, $4)`, schoolID, regionID, "S "+suffix, "S"+suffix)
	mustExec(`INSERT INTO users (id, email, password_hash, role, full_name, school_id) VALUES ($1, $2, 'x', 'student', 'S', $3)`,
		studentUserID, "student-"+suffix+"@edugraph.et", schoolID)
	mustExec(`INSERT INTO users (id, email, password_hash, role, full_name) VALUES ($1, $2, 'x', 'teacher', 'T')`,
		nonStudentUserID, "teacher-"+suffix+"@edugraph.et")
	mustExec(`INSERT INTO students (id, user_id, school_id, admission_no, grade_level) VALUES ($1, $2, $3, $4, 9)`,
		uuid.NewString(), studentUserID, schoolID, "ADM-"+suffix)

	// A real student account with no matches yet -- should succeed with
	// an empty list, not error (there's nothing IDOR-able here, but this
	// confirms the resolution path itself works for a legitimate caller).
	matches, err := svc.Matches(ctx, studentUserID)
	if err != nil {
		t.Errorf("Matches failed for a real student account: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected no matches for a freshly seeded student, got %d", len(matches))
	}

	// An authenticated account with no student profile (e.g. this
	// endpoint being hit by a teacher/school_admin/anyone else) must be
	// rejected, not silently resolve to nothing or someone else's data.
	if _, err := svc.Matches(ctx, nonStudentUserID); err == nil {
		t.Error("Matches succeeded for a caller with no student profile, want an error")
	}
}
