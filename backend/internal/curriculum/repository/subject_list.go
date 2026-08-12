package repository

import (
	"context"
	"fmt"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
)

// ListSubjects returns every promoted subject system-wide, newest first --
// backs the Ministry curriculum browser (unlike ListJobsByUser, this is
// not scoped to a single uploader). Counts are computed per subject so the
// list view can show scale (units/topics/CLOs) without a second round
// trip per row.
func (r *Repository) ListSubjects(ctx context.Context) ([]dto.SubjectListItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			s.code, s.name_en, s.name_am, s.grade_level, s.academic_year, s.moe_code,
			s.is_mandatory, s.version, s.is_current, s.previous_version_code, s.created_at,
			j.file_name, j.approved_at, u.full_name,
			(SELECT count(*) FROM curriculum.units WHERE subject_code = s.code) AS unit_count,
			(SELECT count(*) FROM curriculum.topics WHERE subject_code = s.code AND parent_topic_id IS NULL) AS topic_count,
			(SELECT count(*) FROM curriculum.topics WHERE subject_code = s.code AND parent_topic_id IS NOT NULL) AS subtopic_count,
			(SELECT count(DISTINCT m.clo_code) FROM curriculum.topic_clo_mappings m
				JOIN curriculum.topics t ON t.id = m.topic_id WHERE t.subject_code = s.code) AS clo_count
		FROM curriculum.subjects s
		LEFT JOIN curriculum.upload_jobs j ON j.id = s.upload_job_id
		LEFT JOIN users u ON u.id = j.uploaded_by
		ORDER BY s.code
	`)
	if err != nil {
		return nil, fmt.Errorf("list subjects: %w", err)
	}
	defer rows.Close()

	items := make([]dto.SubjectListItem, 0)
	for rows.Next() {
		var item dto.SubjectListItem
		if err := rows.Scan(
			&item.Code, &item.NameEn, &item.NameAm, &item.GradeLevel, &item.AcademicYear, &item.MoeCode,
			&item.IsMandatory, &item.Version, &item.IsCurrent, &item.PreviousVersionCode, &item.CreatedAt,
			&item.FileName, &item.ApprovedAt, &item.UploadedByName,
			&item.UnitCount, &item.TopicCount, &item.SubtopicCount, &item.CloCount,
		); err != nil {
			return nil, fmt.Errorf("scan subject row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subject rows: %w", err)
	}

	return items, nil
}
