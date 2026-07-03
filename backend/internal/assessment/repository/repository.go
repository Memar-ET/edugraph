package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var ErrNotFound = errors.New("not found")

type Assessment struct {
	ID         string
	SubjectID  string
	SchoolID   *string
	CreatedBy  string
	Title      string
	Type       string
	TotalMarks int
	CreatedAt  time.Time
}

type CreateAssessmentParams struct {
	SubjectID  string
	SchoolID   *string
	CreatedBy  string
	Title      string
	Type       string
	TotalMarks int
}

type Question struct {
	ID            string
	AssessmentID  string
	QuestionText  string
	Options       []string
	CorrectAnswer string
	Marks         int
	OrderIndex    int
}

type CreateQuestionParams struct {
	AssessmentID  string
	QuestionText  string
	Options       []string
	CorrectAnswer string
	Marks         int
	OrderIndex    int
}

type Result struct {
	ID           string
	AssessmentID string
	StudentID    string
	Score        float64
	SubmittedAt  time.Time
	GradedAt     *time.Time
}

type Repository struct {
	pool  *pgxpool.Pool
	neo4j neo4jdriver.DriverWithContext
}

func New(pool *pgxpool.Pool, neo4j neo4jdriver.DriverWithContext) *Repository {
	return &Repository{pool: pool, neo4j: neo4j}
}

func (r *Repository) CreateAssessment(ctx context.Context, p CreateAssessmentParams) (Assessment, error) {
	const q = `INSERT INTO assessments (subject_id, school_id, created_by, title, type, total_marks)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, subject_id, school_id, created_by, title, type, total_marks, created_at`
	var a Assessment
	err := r.pool.QueryRow(ctx, q, p.SubjectID, p.SchoolID, p.CreatedBy, p.Title, p.Type, p.TotalMarks).
		Scan(&a.ID, &a.SubjectID, &a.SchoolID, &a.CreatedBy, &a.Title, &a.Type, &a.TotalMarks, &a.CreatedAt)
	if err != nil {
		return Assessment{}, fmt.Errorf("create assessment: %w", err)
	}
	return a, nil
}

func (r *Repository) GetAssessment(ctx context.Context, id string) (Assessment, error) {
	const q = `SELECT id, subject_id, school_id, created_by, title, type, total_marks, created_at
		FROM assessments WHERE id = $1`
	var a Assessment
	err := r.pool.QueryRow(ctx, q, id).
		Scan(&a.ID, &a.SubjectID, &a.SchoolID, &a.CreatedBy, &a.Title, &a.Type, &a.TotalMarks, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Assessment{}, ErrNotFound
	}
	if err != nil {
		return Assessment{}, fmt.Errorf("get assessment: %w", err)
	}
	return a, nil
}

func (r *Repository) ListAssessments(ctx context.Context, subjectID, schoolID string, limit, offset int) ([]Assessment, int64, error) {
	q := `SELECT id, subject_id, school_id, created_by, title, type, total_marks, created_at FROM assessments WHERE true`
	countQ := `SELECT count(*) FROM assessments WHERE true`
	args := []any{}
	if subjectID != "" {
		args = append(args, subjectID)
		q += fmt.Sprintf(" AND subject_id = $%d", len(args))
		countQ += fmt.Sprintf(" AND subject_id = $%d", len(args))
	}
	if schoolID != "" {
		args = append(args, schoolID)
		q += fmt.Sprintf(" AND school_id = $%d", len(args))
		countQ += fmt.Sprintf(" AND school_id = $%d", len(args))
	}
	q += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d OFFSET %d", limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list assessments: %w", err)
	}
	defer rows.Close()

	var items []Assessment
	for rows.Next() {
		var a Assessment
		if err := rows.Scan(&a.ID, &a.SubjectID, &a.SchoolID, &a.CreatedBy, &a.Title, &a.Type, &a.TotalMarks, &a.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan assessment: %w", err)
		}
		items = append(items, a)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count assessments: %w", err)
	}

	return items, total, nil
}

func (r *Repository) AddQuestion(ctx context.Context, p CreateQuestionParams) (Question, error) {
	optionsJSON, err := json.Marshal(p.Options)
	if err != nil {
		return Question{}, fmt.Errorf("marshal options: %w", err)
	}
	const q = `INSERT INTO assessment_questions (assessment_id, question_text, options, correct_answer, marks, order_index)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, assessment_id, question_text, options, correct_answer, marks, order_index`
	var question Question
	var rawOptions []byte
	err = r.pool.QueryRow(ctx, q, p.AssessmentID, p.QuestionText, optionsJSON, p.CorrectAnswer, p.Marks, p.OrderIndex).
		Scan(&question.ID, &question.AssessmentID, &question.QuestionText, &rawOptions, &question.CorrectAnswer, &question.Marks, &question.OrderIndex)
	if err != nil {
		return Question{}, fmt.Errorf("add question: %w", err)
	}
	_ = json.Unmarshal(rawOptions, &question.Options)
	return question, nil
}

func (r *Repository) ListQuestions(ctx context.Context, assessmentID string) ([]Question, error) {
	const q = `SELECT id, assessment_id, question_text, options, correct_answer, marks, order_index
		FROM assessment_questions WHERE assessment_id = $1 ORDER BY order_index`
	rows, err := r.pool.Query(ctx, q, assessmentID)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var question Question
		var rawOptions []byte
		if err := rows.Scan(&question.ID, &question.AssessmentID, &question.QuestionText, &rawOptions, &question.CorrectAnswer, &question.Marks, &question.OrderIndex); err != nil {
			return nil, fmt.Errorf("scan question: %w", err)
		}
		_ = json.Unmarshal(rawOptions, &question.Options)
		questions = append(questions, question)
	}
	return questions, nil
}

func (r *Repository) SubmitResult(ctx context.Context, assessmentID, studentID string, answers map[string]string, score float64) (Result, error) {
	answersJSON, err := json.Marshal(answers)
	if err != nil {
		return Result{}, fmt.Errorf("marshal answers: %w", err)
	}
	const q = `INSERT INTO assessment_results (assessment_id, student_id, answers, score, graded_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (assessment_id, student_id)
		DO UPDATE SET answers = $3, score = $4, graded_at = now()
		RETURNING id, assessment_id, student_id, score, submitted_at, graded_at`
	var result Result
	err = r.pool.QueryRow(ctx, q, assessmentID, studentID, answersJSON, score).
		Scan(&result.ID, &result.AssessmentID, &result.StudentID, &result.Score, &result.SubmittedAt, &result.GradedAt)
	if err != nil {
		return Result{}, fmt.Errorf("submit result: %w", err)
	}
	return result, nil
}

func (r *Repository) ListResultsByStudent(ctx context.Context, studentID string) ([]Result, error) {
	const q = `SELECT id, assessment_id, student_id, score, submitted_at, graded_at
		FROM assessment_results WHERE student_id = $1 ORDER BY submitted_at DESC`
	return r.queryResults(ctx, q, studentID)
}

func (r *Repository) ListResultsByAssessment(ctx context.Context, assessmentID string) ([]Result, error) {
	const q = `SELECT id, assessment_id, student_id, score, submitted_at, graded_at
		FROM assessment_results WHERE assessment_id = $1 ORDER BY score DESC`
	return r.queryResults(ctx, q, assessmentID)
}

func (r *Repository) queryResults(ctx context.Context, q, arg string) ([]Result, error) {
	rows, err := r.pool.Query(ctx, q, arg)
	if err != nil {
		return nil, fmt.Errorf("query results: %w", err)
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var res Result
		if err := rows.Scan(&res.ID, &res.AssessmentID, &res.StudentID, &res.Score, &res.SubmittedAt, &res.GradedAt); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		results = append(results, res)
	}
	return results, nil
}
