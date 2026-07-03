// Package errors defines the application's error type and its mapping to
// HTTP status codes, so handlers never hand-roll status logic.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func New(status int, code, message string) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

func Wrap(err error, status int, code, message string) *AppError {
	return &AppError{Code: code, Message: message, Status: status, Err: err}
}

func NotFound(message string) *AppError   { return New(http.StatusNotFound, "not_found", message) }
func BadRequest(message string) *AppError { return New(http.StatusBadRequest, "bad_request", message) }
func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, "unauthorized", message)
}
func Forbidden(message string) *AppError { return New(http.StatusForbidden, "forbidden", message) }
func Conflict(message string) *AppError  { return New(http.StatusConflict, "conflict", message) }
func NotImplemented(message string) *AppError {
	return New(http.StatusNotImplemented, "not_implemented", message)
}

func Internal(err error) *AppError {
	return Wrap(err, http.StatusInternalServerError, "internal_error", "an internal error occurred")
}

// As unwraps err into an *AppError if possible.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	ok := errors.As(err, &appErr)
	return appErr, ok
}
