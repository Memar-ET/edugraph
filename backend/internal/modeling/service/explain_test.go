package service

import (
	"testing"

	"github.com/edugraph-ai/edugraph/internal/modeling/dto"
	"github.com/edugraph-ai/edugraph/internal/modeling/repository"
)

func floatPtr(f float64) *float64 { return &f }

func TestConfidenceLabel(t *testing.T) {
	cases := []struct {
		name  string
		state *repository.SkillState
		want  string
	}{
		{"no evidence", &repository.SkillState{EvidenceCount: 0}, "unknown"},
		{"sparse evidence", &repository.SkillState{EvidenceCount: 1}, "low"},
		{"two observations still low", &repository.SkillState{EvidenceCount: 2}, "low"},
		{"moderate evidence", &repository.SkillState{EvidenceCount: 3}, "moderate"},
		{"seven observations still moderate", &repository.SkillState{EvidenceCount: 7}, "moderate"},
		{"well evidenced", &repository.SkillState{EvidenceCount: 8}, "high"},
		{"lots of evidence", &repository.SkillState{EvidenceCount: 100}, "high"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := confidenceLabel(c.state); got != c.want {
				t.Errorf("confidenceLabel(%+v) = %q, want %q", c.state, got, c.want)
			}
		})
	}
}

func TestRecommend_NoEstimateYet(t *testing.T) {
	state := &repository.SkillState{MasteryProbability: nil}
	rec, reason := recommend(state, nil)
	if rec != "Run a short diagnostic check" {
		t.Errorf("recommendation = %q, want a diagnostic-check suggestion for a cold-start topic", rec)
	}
	if reason == "" {
		t.Error("reason must not be empty")
	}
}

func TestRecommend_AboveThreshold_NoActionNeeded(t *testing.T) {
	state := &repository.SkillState{MasteryProbability: floatPtr(0.75), EvidenceCount: 5}
	rec, _ := recommend(state, nil)
	if rec != "No action needed right now" {
		t.Errorf("recommendation = %q, want 'no action needed' for mastery above threshold", rec)
	}
}

func TestRecommend_WeakWithWeakPrerequisite_RecommendsThePrerequisite(t *testing.T) {
	// Mirrors the spec's Fractions worked example (section 9.1): a weak
	// topic whose own direct prerequisite is ALSO weak should point at
	// the prerequisite, not at practicing the topic itself directly.
	state := &repository.SkillState{MasteryProbability: floatPtr(0.3), EvidenceCount: 5}
	prereqs := []dto.PrerequisiteMasterySummary{
		{TopicID: "fractions", Title: "Fractions", EdgeType: "requires", MasteryProbability: floatPtr(0.2)},
		{TopicID: "notation", Title: "Notation", EdgeType: "related_to", MasteryProbability: floatPtr(0.1)}, // soft edge -- must be ignored
	}
	rec, reason := recommend(state, prereqs)
	if rec != `Review the prerequisite "Fractions" first` {
		t.Errorf("recommendation = %q, want it to point at the weak hard-dependency prerequisite", rec)
	}
	if reason == "" {
		t.Error("reason must not be empty")
	}
}

func TestRecommend_WeakWithStrongPrerequisite_AndFewObservations_SuggestsDiagnostic(t *testing.T) {
	state := &repository.SkillState{MasteryProbability: floatPtr(0.3), EvidenceCount: 1}
	prereqs := []dto.PrerequisiteMasterySummary{
		{TopicID: "multiplication", Title: "Multiplication", EdgeType: "requires", MasteryProbability: floatPtr(0.9)},
	}
	rec, _ := recommend(state, prereqs)
	if rec != "Run a short diagnostic check" {
		t.Errorf("recommendation = %q, want a diagnostic suggestion when evidence is too sparse to trust", rec)
	}
}

func TestRecommend_WeakWithStrongPrerequisite_AndEnoughEvidence_SuggestsPractice(t *testing.T) {
	state := &repository.SkillState{MasteryProbability: floatPtr(0.3), EvidenceCount: 5}
	prereqs := []dto.PrerequisiteMasterySummary{
		{TopicID: "multiplication", Title: "Multiplication", EdgeType: "requires", MasteryProbability: floatPtr(0.9)},
	}
	rec, _ := recommend(state, prereqs)
	if rec != "Assign targeted practice on this topic" {
		t.Errorf("recommendation = %q, want targeted practice when the topic itself is the gap", rec)
	}
}

func TestRecommend_IgnoresSoftEdgeTypesWhenPickingWeakestPrerequisite(t *testing.T) {
	state := &repository.SkillState{MasteryProbability: floatPtr(0.3), EvidenceCount: 5}
	prereqs := []dto.PrerequisiteMasterySummary{
		// Only a similar_to edge is weak -- must NOT be treated as a
		// blocking prerequisite (soft associations aren't dependencies).
		{TopicID: "analogous", Title: "Analogous Topic", EdgeType: "similar_to", MasteryProbability: floatPtr(0.1)},
	}
	rec, _ := recommend(state, prereqs)
	if rec != "Assign targeted practice on this topic" {
		t.Errorf("recommendation = %q, a weak similar_to edge must not be treated as a blocking prerequisite", rec)
	}
}
