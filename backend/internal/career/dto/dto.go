package dto

import "time"

type CreateCareerPathRequest struct {
	Title            string   `json:"title" validate:"required"`
	Description      string   `json:"description,omitempty"`
	RequiredSubjects []string `json:"required_subjects,omitempty"`
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
