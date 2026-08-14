package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	"github.com/edugraph-ai/edugraph/internal/assessment/service"
	"github.com/edugraph-ai/edugraph/internal/testutil"
)

type examFixture struct {
	teacherUserID string
	studentID     string
	examID        uuid.UUID
	questionID    uuid.UUID
}

func seedExamFixture(t *testing.T, pool *pgxpool.Pool) examFixture {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()[:8]

	regionID, schoolID := uuid.NewString(), uuid.NewString()
	teacherUserID := uuid.NewString()
	studentUserID := uuid.NewString()
	studentID := uuid.NewString()
	examID := uuid.New()
	questionID := uuid.New()

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("seed exam fixture: %v", err)
		}
	}

	mustExec(`INSERT INTO regions (id, name, code) VALUES ($1, $2, $3)`, regionID, "R "+suffix, "R"+suffix)
	mustExec(`INSERT INTO schools (id, region_id, name, code) VALUES ($1, $2, $3, $4)`, schoolID, regionID, "S "+suffix, "S"+suffix)
	mustExec(`INSERT INTO users (id, email, password_hash, role, full_name, school_id) VALUES ($1, $2, 'x', 'teacher', 'T', $3)`,
		teacherUserID, "teacher-"+suffix+"@edugraph.et", schoolID)
	mustExec(`INSERT INTO users (id, email, password_hash, role, full_name, school_id) VALUES ($1, $2, 'x', 'student', 'S', $3)`,
		studentUserID, "student-"+suffix+"@edugraph.et", schoolID)
	mustExec(`INSERT INTO students (id, user_id, school_id, admission_no, grade_level) VALUES ($1, $2, $3, $4, 9)`,
		studentID, studentUserID, schoolID, "ADM-"+suffix)
	mustExec(`INSERT INTO curriculum.subjects (code, name_en, grade_level, academic_year) VALUES ($1, 'Test Subject', 9, '2026')`, "SUBJ"+suffix)
	mustExec(`INSERT INTO assessment.exams (id, created_by, school_id, subject_code, grade_level, academic_year, exam_scope, title, total_marks, status)
		VALUES ($1, $2, $3, $4, 9, '2026', 'unit_test', 'Test Exam', 10, 'published')`,
		examID, teacherUserID, schoolID, "SUBJ"+suffix)
	mustExec(`INSERT INTO assessment.questions (id, exam_id, school_id, sequence_number, question_text, question_type, marks)
		VALUES ($1, $2, $3, 1, 'Q1?', 'short_answer', 5)`, questionID, examID, schoolID)
	mustExec(`INSERT INTO assessment.exam_attempts (id, student_id, exam_id, school_id) VALUES ($1, $2, $3, $4)`,
		uuid.NewString(), studentID, examID, schoolID)

	return examFixture{teacherUserID: teacherUserID, studentID: studentID, examID: examID, questionID: questionID}
}

func newTestExamService(t *testing.T) (*service.Service, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.RequirePostgres(t)
	t.Cleanup(pool.Close)
	var neo4jDriver neo4jdriver.DriverWithContext // unused by BulkGradeExam
	redis := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() { _ = redis.Close() })
	// storage and ai are nil: BulkGradeExam/verifyCallerOwnsExam never
	// touch either -- only the upload/answer-key paths this test
	// doesn't exercise do.
	return service.New(repository.New(pool, neo4jDriver), nil, redis, nil), pool
}

func TestBulkGradeExam_RecordsGradeHistory(t *testing.T) {
	svc, pool := newTestExamService(t)
	f := seedExamFixture(t, pool)
	ctx := context.Background()
	teacherID := uuid.MustParse(f.teacherUserID)

	_, err := svc.BulkGradeExam(ctx, f.examID, teacherID, dto.BulkGradeRequest{
		Entries: []dto.GradeEntry{{StudentID: uuid.MustParse(f.studentID), QuestionID: f.questionID, Value: "3"}},
	})
	if err != nil {
		t.Fatalf("BulkGradeExam (first grading): %v", err)
	}

	// Regrade with a different mark.
	_, err = svc.BulkGradeExam(ctx, f.examID, teacherID, dto.BulkGradeRequest{
		Entries: []dto.GradeEntry{{StudentID: uuid.MustParse(f.studentID), QuestionID: f.questionID, Value: "5"}},
	})
	if err != nil {
		t.Fatalf("BulkGradeExam (regrade): %v", err)
	}

	var finalMarks float64
	if err := pool.QueryRow(ctx, `SELECT marks_awarded FROM assessment.student_answers WHERE question_id = $1`, f.questionID).Scan(&finalMarks); err != nil {
		t.Fatalf("read final marks: %v", err)
	}
	if finalMarks != 5 {
		t.Errorf("final marks_awarded = %v, want 5 (the regrade should have overwritten the first grading)", finalMarks)
	}

	// checklist 10.3: two grading events (first grade + regrade) means
	// two history rows, not one overwritten row and not zero.
	var historyCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM assessment.answer_grade_history WHERE answer_id IN (
		SELECT id FROM assessment.student_answers WHERE question_id = $1)`, f.questionID).Scan(&historyCount); err != nil {
		t.Fatalf("count grade history: %v", err)
	}
	if historyCount != 2 {
		t.Errorf("answer_grade_history has %d rows, want 2 (one per grading event)", historyCount)
	}
}

func TestBulkGradeExam_RejectsCallerFromDifferentSchool(t *testing.T) {
	// checklist 11.3: BulkGradeExam used to have no school-ownership
	// check at all beyond RequireRole(teacher, school_admin) -- any
	// teacher nationwide could grade any exam.
	svc, pool := newTestExamService(t)
	f := seedExamFixture(t, pool)
	ctx := context.Background()

	// A second teacher at a DIFFERENT school.
	suffix := uuid.NewString()[:8]
	otherRegionID, otherSchoolID := uuid.NewString(), uuid.NewString()
	otherTeacherUserID := uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO regions (id, name, code) VALUES ($1, $2, $3)`, otherRegionID, "OR "+suffix, "OR"+suffix); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO schools (id, region_id, name, code) VALUES ($1, $2, $3, $4)`, otherSchoolID, otherRegionID, "OS "+suffix, "OS"+suffix); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, password_hash, role, full_name, school_id) VALUES ($1, $2, 'x', 'teacher', 'T', $3)`,
		otherTeacherUserID, "otherteacher-"+suffix+"@edugraph.et", otherSchoolID); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := svc.BulkGradeExam(ctx, f.examID, uuid.MustParse(otherTeacherUserID), dto.BulkGradeRequest{
		Entries: []dto.GradeEntry{{StudentID: uuid.MustParse(f.studentID), QuestionID: f.questionID, Value: "3"}},
	})
	if err == nil {
		t.Error("a teacher from a different school graded this exam -- ownership check is not enforced")
	}
}
