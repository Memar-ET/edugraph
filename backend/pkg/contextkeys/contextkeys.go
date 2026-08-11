// Package contextkeys defines typed keys for values stored on request
// contexts, so packages never collide on plain string keys.
package contextkeys

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	UserRoleKey  contextKey = "user_role"
	RequestIDKey contextKey = "request_id"
)
