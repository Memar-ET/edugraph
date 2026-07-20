// Package repository provides read-only national/regional aggregate queries
// for the ministry oversight dashboard. It intentionally has no write path —
// ministry data is derived entirely from region/school/student/teacher records
// owned by their respective domains.
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Overview struct {
	TotalRegions  int64
	TotalSchools  int64
	TotalStudents int64
	TotalTeachers int64
}

type RegionStats struct {
	RegionID           string
	SchoolCount        int64
	StudentCount       int64
	TeacherCount       int64
	AvgAssessmentScore float64
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Overview(ctx context.Context) (Overview, error) {
	const q = `SELECT
		(SELECT count(*) FROM regions),
		(SELECT count(*) FROM schools),
		(SELECT count(*) FROM students),
		(SELECT count(*) FROM teachers)`

	var o Overview
	if err := r.pool.QueryRow(ctx, q).Scan(&o.TotalRegions, &o.TotalSchools, &o.TotalStudents, &o.TotalTeachers); err != nil {
		return Overview{}, fmt.Errorf("query overview: %w", err)
	}
	return o, nil
}

func (r *Repository) RegionStats(ctx context.Context, regionID string) (RegionStats, error) {
	const q = `SELECT
		$1::uuid,
		(SELECT count(*) FROM schools WHERE region_id = $1),
		(SELECT count(*) FROM students st JOIN schools sc ON sc.id = st.school_id WHERE sc.region_id = $1),
		(SELECT count(*) FROM teachers t JOIN schools sc ON sc.id = t.school_id WHERE sc.region_id = $1),
		COALESCE((
			SELECT avg(ea.percentage) FROM assessment.exam_attempts ea
			JOIN students st ON st.id = ea.student_id
			JOIN schools sc ON sc.id = st.school_id
			WHERE sc.region_id = $1 AND ea.percentage IS NOT NULL
		), 0)`

	var s RegionStats
	err := r.pool.QueryRow(ctx, q, regionID).Scan(
		&s.RegionID, &s.SchoolCount, &s.StudentCount, &s.TeacherCount, &s.AvgAssessmentScore,
	)
	if err != nil {
		return RegionStats{}, fmt.Errorf("query region stats: %w", err)
	}
	return s, nil
}
