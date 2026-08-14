package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edugraph-ai/edugraph/internal/teacher/repository"
	"github.com/edugraph-ai/edugraph/internal/teacher/service"
	"github.com/edugraph-ai/edugraph/internal/testutil"
)

// seedTwoSchoolsWithTeachers mirrors student/service's scopeFixture at
// the scale this test actually needs -- two schools, one teacher
// account per school, one *other* teacher (the record being fetched) per
// school.
func seedTwoSchoolsWithTeachers(t *testing.T, pool *pgxpool.Pool) (callerASchoolID, teacherAID, teacherBID string) {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()[:8]
	regionID := uuid.NewString()
	schoolAID, schoolBID := uuid.NewString(), uuid.NewString()
	callerAUserID := uuid.NewString()
	teacherAUserID, teacherBUserID := uuid.NewString(), uuid.NewString()

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("seed fixture: %v", err)
		}
	}

	mustExec(`INSERT INTO regions (id, name, code) VALUES ($1, $2, $3)`, regionID, "R "+suffix, "R"+suffix)
	mustExec(`INSERT INTO schools (id, region_id, name, code) VALUES ($1, $2, $3, $4)`, schoolAID, regionID, "SA "+suffix, "SA"+suffix)
	mustExec(`INSERT INTO schools (id, region_id, name, code) VALUES ($1, $2, $3, $4)`, schoolBID, regionID, "SB "+suffix, "SB"+suffix)

	mkUser := func(id, email, schoolID string) {
		mustExec(`INSERT INTO users (id, email, password_hash, role, full_name, school_id) VALUES ($1, $2, 'x', 'teacher', 'T', $3)`, id, email, schoolID)
	}
	mkUser(callerAUserID, "callerA-"+suffix+"@edugraph.et", schoolAID)
	mkUser(teacherAUserID, "teacherA-"+suffix+"@edugraph.et", schoolAID)
	mkUser(teacherBUserID, "teacherB-"+suffix+"@edugraph.et", schoolBID)

	teacherAID, teacherBID = uuid.NewString(), uuid.NewString()
	mustExec(`INSERT INTO teachers (id, user_id, school_id) VALUES ($1, $2, $3)`, teacherAID, teacherAUserID, schoolAID)
	mustExec(`INSERT INTO teachers (id, user_id, school_id) VALUES ($1, $2, $3)`, teacherBID, teacherBUserID, schoolBID)

	return callerAUserID, teacherAID, teacherBID
}

// TestGet_ScopedToOwnSchool: checklist 10.1/11.3 -- GET /teachers/{id}
// used to have no ownership check at all, same finding as students.
func TestGet_ScopedToOwnSchool(t *testing.T) {
	pool := testutil.RequirePostgres(t)
	t.Cleanup(pool.Close)
	svc := service.New(repository.New(pool))
	ctx := context.Background()

	callerAUserID, teacherAID, teacherBID := seedTwoSchoolsWithTeachers(t, pool)

	if _, err := svc.Get(ctx, callerAUserID, "teacher", teacherAID); err != nil {
		t.Errorf("caller could not view a colleague at their own school: %v", err)
	}
	if _, err := svc.Get(ctx, callerAUserID, "teacher", teacherBID); err == nil {
		t.Error("caller viewed a teacher at a DIFFERENT school -- ownership check is not enforced")
	}
}
