package dto

import "time"

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required"`
	Role     string `json:"role" validate:"required,oneof=student teacher school_admin regional_admin ministry_admin"`
	Phone    string `json:"phone,omitempty"`
	RegionID string `json:"region_id,omitempty" validate:"omitempty,uuid"`
	SchoolID string `json:"school_id,omitempty" validate:"omitempty,uuid"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	Phone     string    `json:"phone,omitempty"`
	RegionID  string    `json:"region_id,omitempty"`
	SchoolID  string    `json:"school_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
	User         UserResponse `json:"user"`
}
