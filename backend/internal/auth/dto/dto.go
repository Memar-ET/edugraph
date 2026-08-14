package dto

import "time"

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required"`
	Role     string `json:"role" validate:"required,oneof=student teacher school_admin regional_admin ministry_admin curriculum_officer"`
	Phone    string `json:"phone,omitempty"`
	RegionID string `json:"region_id,omitempty" validate:"omitempty,uuid"`
	SchoolID string `json:"school_id,omitempty" validate:"omitempty,uuid"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
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

// AuthResponse is the service layer's return type -- AccessToken/
// RefreshToken are `json:"-"` deliberately (checklist 11.1): they're
// consumed by the handler to set HttpOnly cookies (see
// middleware.SetAuthCookies) and must never reach the JSON response
// body, or a client-side script could read them there even though it
// can't read the cookie itself. This is a defense-in-depth default, not
// just handler discipline -- even a future bug that passes this struct
// straight to middleware.WriteJSON can't leak the tokens.
type AuthResponse struct {
	AccessToken  string       `json:"-"`
	RefreshToken string       `json:"-"`
	ExpiresIn    int          `json:"expires_in"`
	User         UserResponse `json:"user"`
}
