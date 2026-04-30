package exponential

import (
	"context"
	"errors"

	errspkg "github.com/Azure/retry/internal/errors"
)

var (
	// ErrRetryCanceled is an error that is returned when a retry is canceled. This is substituted for a context.Canceled
	// or context.DeadlineExceeded error to differentiate between a retry being cancelled and the last error from the Op being
	// context.Canceled or context.DeadlineExceeded.
	ErrRetryCanceled = errspkg.ErrRetryCanceled // This is a type alias.

	// ErrPermanent is an error that is permanent and cannot be retried. This
	// is similar to errors.ErrUnsupported in that it shouldn't be used directly, but instead
	// wrapped in another error. You can determine if you have a permanent error with
	// Is(err, ErrPermanent).
	ErrPermanent = errspkg.ErrPermanent // This is a type alias.
)

// ErrRetryAfter can be used to wrap an error to indicate that the error can be retried after a certain time.
// This is useful when a remote service returns a retry interval in the response and you want to carry the
// signal to your retry logic. This error should not be returned to the caller of Retry().
// DO NOT use this as &ErrRetryAfter{}, simply ErrRetryAfter{} or it won't work.
type ErrRetryAfter = errspkg.ErrRetryAfter // This is a type alias.

func isContextCanceled(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
