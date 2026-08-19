package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
)

// mcqFixture is a variant of examFixture with a real answer key (unlike
// seedExamFixture's short_answer question) and NO pre-seeded attempt --
// these tests exercise StartAttempt itself, which seedExamFixture's
// baked-in attempt row would shadow.
type mcqFixture struct {
	teacherUserID string
	studentUserID string
	studentID     string
	schoolID      string
	examID        uuid.UUID
	questionID    uuid.UUID
}

func seedMCQFixture(t *testing.T, pool *pgxpool.Pool) mcqFixture {
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
			t.Fatalf("seed mcq fixture: %v", err)
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
	mustExec(`INSERT INTO assessment.exams (id, created_by, school_id, subject_code, grade_level, academic_year, exam_scope, title, total_marks, status, attempt_limit)
		VALUES ($1, $2, $3, $4, 9, '2026', 'unit_test', 'MCQ Test Exam', 2, 'published', 1)`,
		examID, teacherUserID, schoolID, "SUBJ"+suffix)
	mustExec(`INSERT INTO assessment.questions (id, exam_id, school_id, sequence_number, question_text, question_type, marks, answer_key, options)
		VALUES ($1, $2, $3, 1, 'What is 2+2?', 'mcq', 2, '{"correctOption":"B"}'::jsonb,
		        '[{"letter":"A","text":"3"},{"letter":"B","text":"4"},{"letter":"C","text":"5"}]'::jsonb)`,
		questionID, examID, schoolID)

	return mcqFixture{
		teacherUserID: teacherUserID, studentUserID: studentUserID, studentID: studentID,
		schoolID: schoolID, examID: examID, questionID: questionID,
	}
}

func TestStartAttempt_IsIdempotentWhileInProgress(t *testing.T) {
	svc, pool := newTestExamService(t)
	f := seedMCQFixture(t, pool)
	ctx := context.Background()
	userID := uuid.MustParse(f.studentUserID)

	first, err := svc.StartAttempt(ctx, userID, f.examID)
	if err != nil {
		t.Fatalf("StartAttempt (first call): %v", err)
	}
	second, err := svc.StartAttempt(ctx, userID, f.examID)
	if err != nil {
		t.Fatalf("StartAttempt (second call): %v", err)
	}

	if first.AttemptID != second.AttemptID {
		t.Errorf("StartAttempt created a second attempt (%s) instead of resuming the first (%s) -- a page refresh would lose the student's session",
			second.AttemptID, first.AttemptID)
	}
	if len(first.Questions) != len(second.Questions) || first.Questions[0].ID != second.Questions[0].ID {
		t.Error("StartAttempt re-randomized question order on the second call -- form must be persisted, not regenerated")
	}

	var attemptCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM assessment.exam_attempts WHERE student_id = $1 AND exam_id = $2`,
		f.studentID, f.examID).Scan(&attemptCount); err != nil {
		t.Fatalf("count attempts: %v", err)
	}
	if attemptCount != 1 {
		t.Errorf("exam_attempts has %d rows for this student/exam, want exactly 1", attemptCount)
	}
}

func TestSubmitExam_DoubleSubmitIsIdempotent(t *testing.T) {
	svc, pool := newTestExamService(t)
	f := seedMCQFixture(t, pool)
	ctx := context.Background()
	userID := uuid.MustParse(f.studentUserID)

	started, err := svc.StartAttempt(ctx, userID, f.examID)
	if err != nil {
		t.Fatalf("StartAttempt: %v", err)
	}

	key := uuid.New()
	req := dto.SubmitExamRequest{
		Answers:        []dto.AnswerInput{{QuestionID: f.questionID, Response: "B"}},
		IdempotencyKey: &key,
	}

	first, err := svc.SubmitExam(ctx, userID, f.examID, req)
	if err != nil {
		t.Fatalf("SubmitExam (first call): %v", err)
	}
	if first.TotalScore == nil || *first.TotalScore != 2 {
		t.Fatalf("first submit total score = %v, want 2 (correct MCQ answer)", first.TotalScore)
	}

	second, err := svc.SubmitExam(ctx, userID, f.examID, req)
	if err != nil {
		t.Fatalf("SubmitExam (duplicate call, same idempotency key): %v", err)
	}
	if second.AttemptID != started.AttemptID {
		t.Errorf("duplicate submit resolved to a different attempt (%s vs %s)", second.AttemptID, started.AttemptID)
	}
	if second.TotalScore == nil || *second.TotalScore != *first.TotalScore {
		t.Errorf("duplicate submit total score = %v, want %v (same result, not re-graded)", second.TotalScore, first.TotalScore)
	}

	var answerRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM assessment.student_answers WHERE attempt_id = $1`,
		started.AttemptID).Scan(&answerRows); err != nil {
		t.Fatalf("count student_answers: %v", err)
	}
	if answerRows != 1 {
		t.Errorf("student_answers has %d rows after a duplicate submit, want exactly 1 (one per question, not doubled)", answerRows)
	}
}

func TestPublishExam_BlocksOnMissingAnswerKey(t *testing.T) {
	svc, pool := newTestExamService(t)
	f := seedMCQFixture(t, pool)
	ctx := context.Background()
	teacherID := uuid.MustParse(f.teacherUserID)

	// Strip the answer key -- an exam in this state must never publish:
	// every MCQ would be ungradeable and every student's score wrong.
	if _, err := pool.Exec(ctx, `UPDATE assessment.questions SET answer_key = NULL WHERE id = $1`, f.questionID); err != nil {
		t.Fatalf("strip answer key: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE assessment.exams SET status = 'validation_pending' WHERE id = $1`, f.examID); err != nil {
		t.Fatalf("set validation_pending: %v", err)
	}

	if _, err := svc.PublishExam(ctx, teacherID, f.examID); err == nil {
		t.Error("PublishExam succeeded on an exam with a missing answer key -- the structural quality gate did not block it")
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM assessment.exams WHERE id = $1`, f.examID).Scan(&status); err != nil {
		t.Fatalf("read exam status: %v", err)
	}
	if status == "published" {
		t.Error("exam status is 'published' despite PublishExam returning an error")
	}
}

func TestCloseExam_BlocksSubsequentStart(t *testing.T) {
	svc, pool := newTestExamService(t)
	f := seedMCQFixture(t, pool)
	ctx := context.Background()

	if _, err := svc.CloseExam(ctx, uuid.MustParse(f.teacherUserID), f.examID); err != nil {
		t.Fatalf("CloseExam: %v", err)
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM assessment.exams WHERE id = $1`, f.examID).Scan(&status); err != nil {
		t.Fatalf("read exam status: %v", err)
	}
	if status != "closed" {
		t.Fatalf("exam status = %q after CloseExam, want 'closed'", status)
	}

	if _, err := svc.StartAttempt(ctx, uuid.MustParse(f.studentUserID), f.examID); err == nil {
		t.Error("StartAttempt succeeded against a closed exam -- closing an exam must actually stop new attempts")
	}

	// A second close on an already-closed exam must not silently
	// succeed or panic (this was a raw 500 before the fix).
	if _, err := svc.CloseExam(ctx, uuid.MustParse(f.teacherUserID), f.examID); err == nil {
		t.Error("closing an already-closed exam succeeded -- should return a clear conflict error")
	}
}
