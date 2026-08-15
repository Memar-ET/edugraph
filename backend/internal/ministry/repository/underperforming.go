package repository

import (
	"context"
	"fmt"
)

// WeakTopicRow is the raw DB row for a topic weakness.
type WeakTopicRow struct {
	TopicID          string
	TopicTitle       string
	AvgMastery       float64
	AffectedStudents int64
}

// UnderperformingSchoolRow is the raw DB row per school.
type UnderperformingSchoolRow struct {
	SchoolID           string
	SchoolName         string
	SchoolCode         string
	QualityScore       float64
	MasteryRate        float64
	FlaggedTopicsCount int64
}

// UnderperformingSchools returns the worst-performing schools in a region
// ordered by ascending average mastery, capped at limit. Each school also
// carries its top 3 weakest topics.
func (r *Repository) UnderperformingSchools(ctx context.Context, regionID string, limit int) ([]UnderperformingSchoolRow, map[string][]WeakTopicRow, error) {
	// Step 1: worst schools by average mastery across their students' gap_records.
	const schoolQ = `
		SELECT
			sc.id,
			sc.name,
			sc.code,
			COALESCE(sq.composite_score, 0)              AS quality_score,
			COALESCE(AVG(gr.mastery_score), 0)           AS mastery_rate,
			COUNT(DISTINCT CASE WHEN gr.mastery_score < 0.5 THEN gr.topic_id END) AS flagged_topics_count
		FROM schools sc
		LEFT JOIN students.gap_records gr ON gr.school_id = sc.id
		LEFT JOIN schools.quality_scores sq ON sq.school_id = sc.id
		WHERE sc.region_id = $1
		GROUP BY sc.id, sc.name, sc.code, sq.composite_score
		ORDER BY mastery_rate ASC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, schoolQ, regionID, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("query underperforming schools: %w", err)
	}
	defer rows.Close()

	var schools []UnderperformingSchoolRow
	for rows.Next() {
		var s UnderperformingSchoolRow
		if err := rows.Scan(&s.SchoolID, &s.SchoolName, &s.SchoolCode, &s.QualityScore, &s.MasteryRate, &s.FlaggedTopicsCount); err != nil {
			return nil, nil, fmt.Errorf("scan underperforming school: %w", err)
		}
		schools = append(schools, s)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate underperforming schools: %w", err)
	}

	if len(schools) == 0 {
		return nil, map[string][]WeakTopicRow{}, nil
	}

	// Step 2: top 3 weakest topics per school.
	schoolIDs := make([]string, len(schools))
	for i, s := range schools {
		schoolIDs[i] = s.SchoolID
	}

	const topicQ = `
		WITH ranked AS (
			SELECT
				gr.school_id,
				gr.topic_id,
				ct.title                       AS topic_title,
				AVG(gr.mastery_score)          AS avg_mastery,
				COUNT(DISTINCT gr.student_id)  AS affected_students,
				ROW_NUMBER() OVER (
					PARTITION BY gr.school_id
					ORDER BY AVG(gr.mastery_score) ASC
				) AS rn
			FROM students.gap_records gr
			JOIN curriculum.topics ct ON ct.id = gr.topic_id
			WHERE gr.school_id = ANY($1::uuid[])
			GROUP BY gr.school_id, gr.topic_id, ct.title
		)
		SELECT school_id, topic_id, topic_title, avg_mastery, affected_students
		FROM ranked
		WHERE rn <= 3
		ORDER BY school_id, avg_mastery ASC`

	trows, err := r.pool.Query(ctx, topicQ, schoolIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("query weak topics: %w", err)
	}
	defer trows.Close()

	topicsBySchool := make(map[string][]WeakTopicRow, len(schools))
	for trows.Next() {
		var sid string
		var t WeakTopicRow
		if err := trows.Scan(&sid, &t.TopicID, &t.TopicTitle, &t.AvgMastery, &t.AffectedStudents); err != nil {
			return nil, nil, fmt.Errorf("scan weak topic: %w", err)
		}
		topicsBySchool[sid] = append(topicsBySchool[sid], t)
	}
	if err := trows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate weak topics: %w", err)
	}

	return schools, topicsBySchool, nil
}
