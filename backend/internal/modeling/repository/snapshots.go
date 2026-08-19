package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/modeling/dto"
)

// ListSkillStateSnapshots returns the historical snapshot record for one
// (student, topic) pair, newest first -- backs the "historical state
// comparison" checklist requirement.
func (r *Repository) ListSkillStateSnapshots(ctx context.Context, studentID, topicID uuid.UUID) ([]dto.SkillStateSnapshot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, mastery_probability, mastery_status, uncertainty, evidence_count,
		       trend, snapshot_reason, source_event_range_start, source_event_range_end, taken_at
		FROM students.skill_state_snapshots
		WHERE student_id = $1 AND topic_id = $2
		ORDER BY taken_at DESC
	`, studentID, topicID)
	if err != nil {
		return nil, fmt.Errorf("list skill state snapshots: %w", err)
	}
	defer rows.Close()

	out := make([]dto.SkillStateSnapshot, 0)
	for rows.Next() {
		var s dto.SkillStateSnapshot
		if err := rows.Scan(
			&s.ID, &s.MasteryProbability, &s.MasteryStatus, &s.Uncertainty, &s.EvidenceCount,
			&s.Trend, &s.SnapshotReason, &s.SourceEventRangeStart, &s.SourceEventRangeEnd, &s.TakenAt,
		); err != nil {
			return nil, fmt.Errorf("scan skill state snapshot: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
