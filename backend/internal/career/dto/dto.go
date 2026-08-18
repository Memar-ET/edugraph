package dto

import "time"

type CreateCareerPathRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description,omitempty"`
	// Sector/MinEduLevel map to careers.careers' NOT NULL sector/
	// min_edu_level columns -- required_subjects has no home there
	// (requirements are topic-level via careers.career_topic_requirements,
	// which has no curation UI yet), so it was dropped from this request
	// rather than silently accepted and discarded.
	Sector      string `json:"sector" validate:"required"`
	MinEduLevel string `json:"minEduLevel" validate:"required"`
}

type CareerPathResponse struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Description      string    `json:"description,omitempty"`
	RequiredSubjects []string  `json:"required_subjects,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type MatchResponse struct {
	CareerPathID string  `json:"career_path_id"`
	Title        string  `json:"title"`
	Score        float64 `json:"score"`
}
