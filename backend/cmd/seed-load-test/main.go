// Command seed-load-test creates N synthetic student accounts sharing
// one published exam, all in the same school/grade, for checklist
// 13.2's k6 exam-submission load test. Not for any other use -- this is
// throwaway load-test fixture data, meant to be created against a
// disposable test database and torn down after.
//
// One bcrypt hash computed once and reused across every seeded user
// (crypto.HashPassword is deliberately slow, cost 12 -- hashing it N
// times for a synthetic load-test fixture would make seeding itself the
// bottleneck, not the thing actually being measured) -- real accounts
// never share a password, this is a test-fixture-only shortcut.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/pkg/config"
	"github.com/edugraph-ai/edugraph/pkg/crypto"
	"github.com/edugraph-ai/edugraph/pkg/database/postgres"
)

const loadTestPassword = "loadtest123"

type seededUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type seedOutput struct {
	ExamID string       `json:"examId"`
	Users  []seededUser `json:"users"`
}

func main() {
	count := flag.Int("count", 500, "number of synthetic student accounts to create")
	outPath := flag.String("out", "/tmp/loadtest-users.json", "where to write the seeded credentials + exam id")
	flag.Parse()

	ctx := context.Background()
	cfg := config.Load()
	pool, err := postgres.NewPool(cfg.Postgres)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	hash, err := crypto.HashPassword(loadTestPassword)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	suffix := uuid.NewString()[:8]
	regionID, schoolID := uuid.New(), uuid.New()
	teacherUserID := uuid.New()
	subjectCode := "LT" + suffix
	examID := uuid.New()
	questionID := uuid.New()

	mustExec := func(sql string, args ...any) {
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			log.Fatalf("seed: %v (sql: %s)", err, sql)
		}
	}

	mustExec(`INSERT INTO regions (id, name, code) VALUES ($1, $2, $3)`, regionID, "LoadTest Region "+suffix, "LTR"+suffix)
	mustExec(`INSERT INTO schools (id, region_id, name, code) VALUES ($1, $2, $3, $4)`, schoolID, regionID, "LoadTest School "+suffix, "LTS"+suffix)
	mustExec(`INSERT INTO users (id, email, password_hash, role, full_name, school_id) VALUES ($1, $2, $3, 'teacher', 'Load Test Teacher', $4)`,
		teacherUserID, "loadtest-teacher-"+suffix+"@edugraph.et", hash, schoolID)
	mustExec(`INSERT INTO curriculum.subjects (code, name_en, grade_level, academic_year) VALUES ($1, 'Load Test Subject', 9, '2026')`, subjectCode)
	mustExec(`INSERT INTO assessment.exams
		(id, created_by, school_id, subject_code, grade_level, academic_year, exam_scope, title, total_marks, status)
		VALUES ($1, $2, $3, $4, 9, '2026', 'unit_test', 'Load Test Exam', 5, 'published')`,
		examID, teacherUserID, schoolID, subjectCode)
	mustExec(`INSERT INTO assessment.questions
		(id, exam_id, school_id, sequence_number, question_text, question_type, marks, answer_key)
		VALUES ($1, $2, $3, 1, 'Load test question', 'mcq', 5, $4)`,
		questionID, examID, schoolID, `{"correctOption":"A"}`)

	out := seedOutput{ExamID: examID.String()}
	for i := 0; i < *count; i++ {
		userID := uuid.New()
		email := fmt.Sprintf("loadtest-student-%s-%d@edugraph.et", suffix, i)
		mustExec(`INSERT INTO users (id, email, password_hash, role, full_name, school_id) VALUES ($1, $2, $3, 'student', $4, $5)`,
			userID, email, hash, fmt.Sprintf("Load Test Student %d", i), schoolID)
		mustExec(`INSERT INTO students (id, user_id, school_id, admission_no, grade_level) VALUES ($1, $2, $3, $4, 9)`,
			uuid.New(), userID, schoolID, fmt.Sprintf("LT-%s-%04d", suffix, i))
		out.Users = append(out.Users, seededUser{Email: email, Password: loadTestPassword})
	}

	f, err := os.Create(*outPath)
	if err != nil {
		log.Fatalf("create output file: %v", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(out); err != nil {
		log.Fatalf("write output: %v", err)
	}

	fmt.Printf("Seeded %d student accounts + 1 exam (id=%s). Credentials written to %s\n", *count, examID, *outPath)
}
