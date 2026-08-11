package dto

import "time"

type CreateNotificationRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	Title  string `json:"title" validate:"required"`
	Body   string `json:"body" validate:"required"`
}

type NotificationResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}
