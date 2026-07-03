// Package validator wraps go-playground/validator to turn struct tag
// validation failures into a single readable error for handlers to return.
package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// Struct validates s against its `validate:"..."` tags and returns nil, or
// a single error joining every failing field.
func Struct(s any) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var fieldErrs validator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		return err
	}

	msgs := make([]string, 0, len(fieldErrs))
	for _, fe := range fieldErrs {
		msgs = append(msgs, fmt.Sprintf("%s failed validation (%s)", fe.Field(), fe.Tag()))
	}
	return errors.New(strings.Join(msgs, "; "))
}
