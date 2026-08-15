package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/modeling/dto"
	"github.com/edugraph-ai/edugraph/internal/modeling/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// Weak/strong thresholds mirror root_cause.WEAK_THRESHOLD and
// fusion.STATUS_THRESHOLDS on the ai-service side -- kept as documented,
// independently-set constants here (Go and Python don't share a config
// file) rather than a cross-language import, since these are used purely
// for HUMAN-FACING PHRASING (confidence label, recommendation text) here,
// never for a scoring computation that must numerically match the Python
// engines exactly.
const weakThreshold = 0.5

// Explain synthesizes the EG-GCKT spec section 18 five-part explanation
// for one (student, topic) pair, reading directly from skill_states,
// evidence_log, and the prerequisite graph -- no live call into
// ai-service; everything needed already lives in Postgres. callerID
// authorizes the request server-side against an arbitrary studentID path
// param (see authorizeExplain) -- this endpoint must never be reachable
// as an IDOR the way career-matches was before checklist 11.3's fix.
func (s *Service) Explain(ctx context.Context, callerID, studentID, topicID uuid.UUID) (*dto.Explanation, error) {
	if err := s.authorizeExplain(ctx, callerID, studentID); err != nil {
		return nil, err
	}

	state, topicTitle, err := s.repo.FetchSkillState(ctx, studentID, topicID)
	if errors.Is(err, repository.ErrSkillStateNotFound) {
		return nil, apperrors.NotFound("no EG-GCKT state for this student/topic yet -- not enough evidence has been collected")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	evidence, err := s.repo.FetchRecentEvidence(ctx, studentID, topicID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	prereqs, err := s.repo.FetchPrerequisiteMastery(ctx, studentID, topicID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	confidence := confidenceLabel(state)
	recommendation, reason := recommend(state, prereqs)

	return &dto.Explanation{
		StudentID:  studentID.String(),
		TopicID:    topicID.String(),
		TopicTitle: topicTitle,
		CurrentState: dto.ExplanationCurrentState{
			MasteryProbability: state.MasteryProbability,
			MasteryStatus:      state.MasteryStatus,
			Trend:              state.Trend,
			EvidenceCount:      state.EvidenceCount,
			ForgettingRisk:     state.ForgettingRisk,
			LastSeen:           state.LastSeen,
		},
		Evidence:          evidence,
		StructuralContext: dto.ExplanationStructuralContext{Prerequisites: prereqs},
		Confidence:        confidence,
		Recommendation:    recommendation,
		Reason:            reason,
	}, nil
}

// authorizeExplain: a student may only view their own explanation; a
// teacher/school_admin only a student at their own school; ministry_
// admin/regional_admin/curriculum_officer have broader oversight and are
// allowed regardless of school (matching e.g. GetRegionStats/ministry
// overview's scope elsewhere in this codebase).
func (s *Service) authorizeExplain(ctx context.Context, callerID, studentID uuid.UUID) error {
	auth, err := s.repo.FetchAuthContext(ctx, callerID)
	if err != nil {
		return apperrors.Internal(err)
	}

	switch auth.Role {
	case "student":
		if auth.OwnStudentID == nil || *auth.OwnStudentID != studentID {
			return apperrors.Forbidden("cannot view another student's explanation")
		}
		return nil
	case "teacher", "school_admin":
		targetSchool, err := s.repo.FetchStudentSchool(ctx, studentID)
		if errors.Is(err, repository.ErrStudentNotFound) {
			return apperrors.NotFound("student not found")
		}
		if err != nil {
			return apperrors.Internal(err)
		}
		if auth.CallerSchool == nil || *auth.CallerSchool != targetSchool {
			return apperrors.Forbidden("cannot view a student outside your school")
		}
		return nil
	case "ministry_admin", "regional_admin", "curriculum_officer":
		return nil
	default:
		return apperrors.Forbidden("not authorized to view this explanation")
	}
}

func confidenceLabel(state *repository.SkillState) string {
	if state.EvidenceCount == 0 {
		return "unknown"
	}
	if state.EvidenceCount < 3 {
		return "low"
	}
	if state.EvidenceCount < 8 {
		return "moderate"
	}
	return "high"
}

// recommend synthesizes a next-action suggestion using the same shape of
// reasoning as ai-service's root_cause.py/action_ranking.py (weakness,
// prerequisite readiness), in human-readable terms -- a lightweight,
// independent restatement for this read endpoint, not a call into the
// actual ranking algorithm (which runs as part of full study-plan
// generation, not per-topic on demand).
func recommend(state *repository.SkillState, prereqs []dto.PrerequisiteMasterySummary) (string, string) {
	if state.MasteryProbability == nil {
		return "Run a short diagnostic check",
			"No mastery estimate exists yet for this topic -- a small diagnostic assessment would establish a starting point."
	}
	mastery := *state.MasteryProbability

	if mastery >= weakThreshold {
		return "No action needed right now",
			fmt.Sprintf("Estimated mastery is %.0f%%, at or above the proficiency threshold.", mastery*100)
	}

	var weakestPrereq *dto.PrerequisiteMasterySummary
	for i := range prereqs {
		p := &prereqs[i]
		if p.EdgeType != "requires" && p.EdgeType != "strongly_requires" {
			continue
		}
		if p.MasteryProbability != nil && *p.MasteryProbability < weakThreshold {
			if weakestPrereq == nil || *p.MasteryProbability < *weakestPrereq.MasteryProbability {
				weakestPrereq = p
			}
		}
	}

	if weakestPrereq != nil {
		return fmt.Sprintf("Review the prerequisite \"%s\" first", weakestPrereq.Title),
			fmt.Sprintf(
				"Mastery here is %.0f%%, but the prerequisite \"%s\" is estimated at %.0f%% -- fixing the "+
					"upstream gap is likely to unlock progress here more effectively than practicing this topic directly.",
				mastery*100, weakestPrereq.Title, *weakestPrereq.MasteryProbability*100,
			)
	}

	if state.EvidenceCount < 3 {
		return "Run a short diagnostic check",
			fmt.Sprintf("Mastery is estimated at %.0f%%, but only %d observation(s) support that estimate -- more evidence would sharpen it.", mastery*100, state.EvidenceCount)
	}

	return "Assign targeted practice on this topic",
		fmt.Sprintf("Mastery is estimated at %.0f%% with no unresolved prerequisite gap found -- this topic itself is the gap.", mastery*100)
}
