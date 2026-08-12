package repository

import (
	"context"
	"fmt"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
)

// ListTopicsBySubject returns every topic (and subtopic) under a subject,
// ordered by unit then sequence -- backs the prerequisites UI's topic
// picker (see dto.TopicListItem).
func (r *Repository) ListTopicsBySubject(ctx context.Context, subjectCode string) ([]dto.TopicListItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.title_en, u.number, t.sequence_order, t.parent_topic_id
		FROM curriculum.topics t
		JOIN curriculum.units u ON u.id = t.unit_id
		WHERE t.subject_code = $1
		ORDER BY u.number, t.sequence_order
	`, subjectCode)
	if err != nil {
		return nil, fmt.Errorf("list topics for subject %q: %w", subjectCode, err)
	}
	defer rows.Close()

	out := make([]dto.TopicListItem, 0)
	for rows.Next() {
		var item dto.TopicListItem
		if err := rows.Scan(&item.ID, &item.TitleEn, &item.UnitNumber, &item.SequenceOrder, &item.ParentTopicID); err != nil {
			return nil, fmt.Errorf("scan topic row: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
