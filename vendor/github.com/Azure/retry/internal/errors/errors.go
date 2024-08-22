// Package errors contains internal types for the retry set of packages.
package errors

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrRetryCanceled is an error that is returned when a retry is canceled. This is substituted for a context.Canceled
	// or context.DeadlineExceeded error to differentiate between a retry being cancelled and the last error from the Op being
	// context.Canceled or context.DeadlineExceeded.
	ErrRetryCanceled = errors.New("retry canceled")

	// ErrPermanent is an error that is permanent and cannot be retried. This
	// is similar to errors.ErrUnsupported in that it shouldn't be used directly, but instead
	// wrapped in another error. You can determine if you have a permanent error with
	// Is(err, ErrPermanent).
	ErrPermanent = errors.New("permanent error")
)

// ErrRetryAfter can be used to wrap an error to indicate that the error can be retried after a certain time.
// This is useful when a remote service returns a retry interval in the response and you want to carry the
// signal to your retry logic. This error should not be returned to the caller of Retry().
// DO NOT use this as &ErrRetryAfter{}, simply ErrRetryAfter{} or it won't work.'
type ErrRetryAfter struct {
	// Time after which the call can be retried.
	Time time.Time
	// The error that can be retried.
	Err error
}

// Error implements error.Error().
func (e ErrRetryAfter) Error() string {
	return e.Err.Error() + fmt.Sprintf(", can be retried after %v", e.Time.UTC())
}

// Unwrap unwraps the error.
func (e ErrRetryAfter) Unwrap() error {
	return e.Err
}

// Is is a wrapper for errors.Is.
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// As is a wrapper for errors.As.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}
