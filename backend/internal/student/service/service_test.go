package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/edugraph-ai/edugraph/internal/student/repository"
	"github.com/edugraph-ai/edugraph/internal/student/service"
	"github.com/edugraph-ai/edugraph/internal/testutil"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

// seedScopeFixture creates two regions, one school per region, one user
// per role per school, and one student per school -- everything the
// checklist 10.1/11.3 scoping tests below need. Returns the ids they
// reference by name for readability at each call site.
type scopeFixture struct {
	regionA, regionB               string
	schoolA, schoolB               string
	studentA, studentB             string // students.id
	teacherAUserID, teacherBUserID string // users.id (teacher role)
	regionalAdminAUserID           string // users.id (regional_admin, region A)
	ministryAdminUserID            string // users.id (ministry_admin, no school/region)
}

func seedScopeFixture(t *testing.T, pool *pgxpool.Pool) scopeFixture {
	t.Helper()
	ctx := context.Background()
	f := scopeFixture{
		regionA: uuid.NewString(), regionB: uuid.NewString(),
		schoolA: uuid.NewString(), schoolB: uuid.NewString(),
		teacherAUserID: uuid.NewString(), teacherBUserID: uuid.NewString(),
		regionalAdminAUserID: uuid.NewString(),
		ministryAdminUserID:  uuid.NewString(),
	}
	studentAUserID, studentBUserID := uuid.NewString(), uuid.NewString()
	suffix := uuid.NewString()[:8] // keeps unique school "code" across repeated local runs

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("seed fixture: %v (sql: %s)", err, sql)
		}
	}

	mustExec(`INSERT INTO regions (id, name, code) VALUES ($1, $2, $3)`, f.regionA, "Region A "+suffix, "RA"+suffix)
	mustExec(`INSERT INTO regions (id, name, code) VALUES ($1, $2, $3)`, f.regionB, "Region B "+suffix, "RB"+suffix)
	mustExec(`INSERT INTO schools (id, region_id, name, code) VALUES ($1, $2, $3, $4)`, f.schoolA, f.regionA, "School A "+suffix, "SA"+suffix)
	mustExec(`INSERT INTO schools (id, region_id, name, code) VALUES ($1, $2, $3, $4)`, f.schoolB, f.regionB, "School B "+suffix, "SB"+suffix)

	mkUser := func(id, email, role, schoolID, regionID string) {
		mustExec(`INSERT INTO users (id, email, password_hash, role, full_name, school_id, region_id)
			VALUES ($1, $2, 'x', $3, 'Test User', NULLIF($4, '')::uuid, NULLIF($5, '')::uuid)`,
			id, email, role, schoolID, regionID)
	}
	mkUser(f.teacherAUserID, "teacherA-"+suffix+"@edugraph.et", "teacher", f.schoolA, "")
	mkUser(f.teacherBUserID, "teacherB-"+suffix+"@edugraph.et", "teacher", f.schoolB, "")
	mkUser(f.regionalAdminAUserID, "regionalA-"+suffix+"@edugraph.et", "regional_admin", "", f.regionA)
	mkUser(f.ministryAdminUserID, "ministry-"+suffix+"@edugraph.et", "ministry_admin", "", "")
	mkUser(studentAUserID, "studentA-"+suffix+"@edugraph.et", "student", f.schoolA, "")
	mkUser(studentBUserID, "studentB-"+suffix+"@edugraph.et", "student", f.schoolB, "")

	f.studentA, f.studentB = uuid.NewString(), uuid.NewString()
	mustExec(`INSERT INTO students (id, user_id, school_id, admission_no, grade_level) VALUES ($1, $2, $3, $4, 9)`,
		f.studentA, studentAUserID, f.schoolA, "ADM-A-"+suffix)
	mustExec(`INSERT INTO students (id, user_id, school_id, admission_no, grade_level) VALUES ($1, $2, $3, $4, 9)`,
		f.studentB, studentBUserID, f.schoolB, "ADM-B-"+suffix)

	return f
}

func newTestStudentService(t *testing.T) (*service.Service, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.RequirePostgres(t)
	t.Cleanup(pool.Close)
	var neo4jDriver neo4jdriver.DriverWithContext // unused by Get/List/Delete -- the methods under test
	return service.New(repository.New(pool, neo4jDriver)), pool
}

// TestGet_ScopedToOwnSchool is the core checklist 10.1 finding: GET
// /students/{id} used to take no caller identity at all, so any
// authenticated teacher/school_admin could fetch any student's record
// nationwide. It's now scoped server-side per role.
func TestGet_ScopedToOwnSchool(t *testing.T) {
	svc, pool := newTestStudentService(t)
	f := seedScopeFixture(t, pool)
	ctx := context.Background()

	tests := []struct {
		name        string
		callerID    string
		role        string
		wantSuccess bool
	}{
		{"teacher of the student's own school can view", f.teacherAUserID, "teacher", true},
		{"teacher of a DIFFERENT school cannot view (the actual bug)", f.teacherBUserID, "teacher", false},
		{"regional_admin of the student's region can view", f.regionalAdminAUserID, "regional_admin", true},
		{"ministry_admin can view any student", f.ministryAdminUserID, "ministry_admin", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Get(ctx, tt.callerID, tt.role, f.studentA)
			if tt.wantSuccess && err != nil {
				t.Errorf("Get failed, want success: %v", err)
			}
			if !tt.wantSuccess && err == nil {
				t.Error("Get succeeded, want it rejected (cross-school access)")
			}
		})
	}
}

// TestList_ScopedToOwnSchool covers the same finding for the list
// endpoint: a teacher/school_admin's requested school_id is now forced
// to their own regardless of what they ask for.
func TestList_ScopedToOwnSchool(t *testing.T) {
	svc, pool := newTestStudentService(t)
	f := seedScopeFixture(t, pool)
	ctx := context.Background()

	// Teacher A asking for School B's roster by school_id must NOT see
	// School B's student -- their own school is forced server-side, the
	// exact IDOR this closes.
	students, _, err := svc.List(ctx, f.teacherAUserID, "teacher", f.schoolB, pagination.Params{Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, s := range students {
		if s.ID == f.studentB {
			t.Error("teacher A's List (requesting school B) returned school B's student -- school scoping is not enforced")
		}
	}

	// Sanity: teacher A's own roster does contain student A.
	students, _, err = svc.List(ctx, f.teacherAUserID, "teacher", "", pagination.Params{Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, s := range students {
		if s.ID == f.studentA {
			found = true
		}
	}
	if !found {
		t.Error("teacher A's own-school List did not include their own student")
	}
}

func TestList_RegionalAdminScopedToOwnRegion(t *testing.T) {
	svc, pool := newTestStudentService(t)
	f := seedScopeFixture(t, pool)
	ctx := context.Background()

	students, _, err := svc.List(ctx, f.regionalAdminAUserID, "regional_admin", "", pagination.Params{Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, s := range students {
		if s.ID == f.studentB {
			t.Error("regional_admin A's List included a student from region B's school -- region scoping is not enforced")
		}
	}
}

func TestDelete_IsSoftDelete(t *testing.T) {
	// checklist 10.3: Delete used to be a hard DELETE cascading through
	// a student's entire academic history -- now it must survive as a
	// row with deleted_at set, and List/Get must stop surfacing it.
	svc, pool := newTestStudentService(t)
	f := seedScopeFixture(t, pool)
	ctx := context.Background()

	if err := svc.Delete(ctx, f.studentA); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	var deletedAtSet bool
	if err := pool.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM students WHERE id = $1`, f.studentA).Scan(&deletedAtSet); err != nil {
		t.Fatalf("check row survived: %v", err)
	}
	if !deletedAtSet {
		t.Fatal("student row does not have deleted_at set -- Delete is not soft-deleting")
	}

	if _, err := svc.Get(ctx, f.ministryAdminUserID, "ministry_admin", f.studentA); err == nil {
		t.Error("Get returned a soft-deleted student, want not-found")
	}

	// Deleting again must report an error (not silently succeed), same
	// as the original hard-delete behavior for a nonexistent row.
	if err := svc.Delete(ctx, f.studentA); err == nil {
		t.Error("double-delete succeeded, want not-found")
	}
}
