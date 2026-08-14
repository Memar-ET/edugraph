package handler

import (
	"context"
	"net/http"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

type deviceContextKey string

const deviceSchoolIDKey deviceContextKey = "sync_device_school_id"

// DeviceVerifier checks a School Box's device_id + secret (see
// V030__sync_device_credentials.sql) and returns the school_id that
// credential is bound to. Implemented by internal/sync/service.
type DeviceVerifier interface {
	VerifyDevice(ctx context.Context, deviceID, secret string) (schoolID string, err error)
}

// DeviceAuth replaces what used to be no authentication at all on
// /api/v1/sync/push and /sync/pull (checklist 10.1 finding -- see this
// migration's own comment for the full story). School Box devices are
// headless and don't have a human to log in, so this is a separate,
// simpler scheme from middleware.Authenticate's JWT: two static headers
// checked against a bcrypt-hashed secret, not a token that expires and
// rotates. The resolved school_id is stashed on the context; Push/Pull
// still need to check it against the request's own school_id (a device
// only proves "I am this device_id", not "and every school_id I claim in
// this request is mine") -- see service.Push/Pull.
func DeviceAuth(verifier DeviceVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			deviceID := r.Header.Get("X-Device-Id")
			secret := r.Header.Get("X-Device-Secret")
			if deviceID == "" || secret == "" {
				middleware.WriteError(w, apperrors.Unauthorized("missing device credentials"))
				return
			}

			schoolID, err := verifier.VerifyDevice(r.Context(), deviceID, secret)
			if err != nil {
				middleware.WriteError(w, err)
				return
			}

			ctx := context.WithValue(r.Context(), deviceSchoolIDKey, schoolID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func deviceSchoolID(ctx context.Context) string {
	v, _ := ctx.Value(deviceSchoolIDKey).(string)
	return v
}
