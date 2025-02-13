package exponential

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// timer is a type that wraps a channel that will receive a time.Time when the timer is done.
// This is used to allow internal testing of the package.
type timer struct {
	// C is the channel that will receive a time.Time when the timer is done.
	// When faking, "c" contains the channel without the <- so that we can feed it.
	C <-chan time.Time
	// c only exists if this is faking and is the same channel as C except without the <-.
	c chan time.Time
	// when is the time the timer is set to go off.
	when time.Time
	// timer is used when not faking and is the real time.Timer.
	timer *time.Timer
	// mu protects everything below.
	mu sync.Mutex
	// stopped is true if Stop() has been called. Only valid if faking.
	stopped bool
}

// Stop implements time.Timer.Stop().
func (t *timer) Stop() bool {
	if t.timer == nil {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.stopped = true
		return true
	}

	return t.timer.Stop()
}

// clock provides access to the various time functions we need.
// This allows internal testing of the package.
type clock interface {
	Now() time.Time
	NewTimer(d time.Duration) *timer
	Until(t time.Time) time.Duration
}

// Backoff provides a mechanism for retrying operations with exponential backoff. This can be used in
// tests without a fake/mock interface to simulate retries either by using the WithTesting()
// option or by setting a Policy that works with your test. This keeps code leaner, avoids
// dynamic dispatch, unneeded allocations and is easier to test.
type Backoff struct {
	// policy is the backoff policy to use.
	policy Policy
	// useTest is true if we are using the test options. Set with WithTesting().
	useTest bool
	// transformers is a list of error transformers to apply to the error before determining
	// if we should retry.
	transformers []ErrTransformer

	// clock is used to allow internal testing of the package.
	// If not set, uses the time package.
	clock clock
}

// Options are used to configure the backoff policy.
type Option func(*Backoff) error

// WithPolicy sets the backoff policy to use. If not specified, then DefaultPolicy is used.
func WithPolicy(policy Policy) Option {
	return func(b *Backoff) error {
		b.policy = policy
		return nil
	}
}

// testOptions is a placeholder for future test options.
type testOptions struct{}

// TestOption is an option for WithTesting(). Functions that implement TestOption
// provide options for tests. This is a placeholder for future test options
// and is not used at this time.
type TestOption func(t *testOptions) error

// WithTesting invokes the backoff policy with no actual delay.
// Cannot be used outside of a test or this will panic.
func WithTesting(options ...TestOption) Option {
	if !testing.Testing() {
		panic("called WithTesting outside of a test")
	}

	return func(b *Backoff) error {
		b.useTest = true
		return nil
	}
}

// ErrTransformer is a function that can be used to transform an error before it is returned.
// The typical case is to make an error a permanent error based on some criteria in order to
// stop retries. The other use is to use errors.ErrRetryAfter as a wrapper to specify the minimum
// time the retry must wait based on a response from a service. This type allows packaging of custom
// retry logic in one place for reuse instead of in the Op. As ErrTransformrers are applied in order,
// the last one to change an error will be the error returned.
type ErrTransformer func(err error) error

// WithErrTransformer sets the error transformers to use. If not specified, then no transformers are used.
// Passing multiple transformers will apply them in order. If WithErrTransformer is passed multiple times,
// only the final transformers are used (aka don't do that).
func WithErrTransformer(transformers ...ErrTransformer) Option {
	return func(b *Backoff) error {
		b.transformers = transformers
		return nil
	}
}

// New creates a new Backoff instance with the given options.
func New(options ...Option) (*Backoff, error) {
	b := &Backoff{
		policy: defaults(),
	}

	for _, o := range options {
		if err := o(b); err != nil {
			return nil, err
		}
	}
	if err := b.policy.validate(); err != nil {
		return nil, err
	}

	return b, nil
}

// Record is the record of a Retry attempt.
type Record struct {
	// Attempt is the number of attempts (initial + retries). A zero value of Record has Attempt == 0.
	Attempt int
	// LastInterval is the last interval used.
	LastInterval time.Duration
	// TotalInterval is the total amount of time spent in intervals between attempts.
	TotalInterval time.Duration
	// Err is the last error returned by an operation. It is important to remember that this is
	// the last error returned by the prior invocation of the Op and should only be used for logging
	// purposes.
	Err error
}

// now returns the current time. This is used to allow internal testing of the package.
// We do this instead of using clock directly to avoid dynamic dispatch.
func (b *Backoff) now() time.Time {
	if b.clock == nil {
		return time.Now()
	}
	return b.clock.Now()
}

// until returns the time until the given time. This is used to allow internal testing of the package.
// We do this instead of using clock directly to avoid dynamic dispatch.
func (b *Backoff) until(t time.Time) time.Duration {
	if b.clock == nil {
		return time.Until(t)
	}
	return b.clock.Until(t)

}

// newTimer creates a new timer. This is used to allow internal testing of the package.
// We do this instead of using clock directly to avoid dynamic dispatch.
func (b *Backoff) newTimer(d time.Duration) *timer {
	if b.clock == nil {
		t := time.NewTimer(d)
		return &timer{C: t.C, timer: t}
	}
	return b.clock.NewTimer(d)
}

// Op is a function that can be retried.
type Op func(context.Context, Record) error

// RetryOption is an option for the Retry method. Functions that implement RetryOption
// provide an override on a single call.
type RetryOption func(o *retryOptions) error

// retryOptions provides override options on a single Retry() call. Currently empty, but provided
// for future extensibility without breaking the API.
type retryOptions struct{}

// Retry will retry the given operation until it succeeds, the context is cancelled or an error
// is returned with PermanentErr(). This is safe to call concurrently.
func (b *Backoff) Retry(ctx context.Context, op Op, options ...RetryOption) error {
	r := Record{Attempt: 1}

	// Make our first attempt.
	err := op(ctx, r)
	if err == nil {
		return nil
	}

	// Well, that didn't work, so let's start our retry work.
	r.Err = err
	baseInterval := b.policy.InitialInterval
	realInterval := b.randomize(baseInterval)

	for {
		err = b.applyTransformers(err)

		if errors.Is(err, ErrPermanent) {
			return err
		}

		// Check to see if the error contained an interval that is longer
		// than the exponential retry timer. If it is, we will use the error
		// retry timer.
		realInterval = b.intervalSpecified(err, realInterval)

		// If our context is done or our interval goes over the context deadline,
		// then we are done.
		if !b.ctxOK(ctx, realInterval) {
			return fmt.Errorf("r.Err: %w", ErrRetryCanceled)
		}

		// Do this if they did not pass the WithTesting() option.
		if !b.useTest {
			timer := b.newTimer(realInterval)
			select {
			case <-ctx.Done():
				timer.Stop() // Prevent goroutine leak
				return fmt.Errorf("%w: %w ", r.Err, ErrRetryCanceled)
			case <-timer.C:
			}
		}

		// Record attempt last attempt number, our last interval and total interval.
		r.LastInterval = realInterval
		r.TotalInterval += realInterval
		r.Attempt++

		// NO WHAMMIES, NO WHAMMIES, STOP!
		// https://www.youtube.com/watch?v=1mGrM72Z4-Y
		err = op(ctx, r)
		if err == nil {
			return nil
		}

		// Captures our last error in the record.
		r.Err = err

		// Create our new base interval for the next attempt.
		baseInterval = time.Duration(float64(baseInterval) * b.policy.Multiplier)
		// Our base interval cannot exceed the maximum interval.
		if baseInterval > b.policy.MaxInterval {
			baseInterval = b.policy.MaxInterval
		}
		// Randomize the interval based on our randomization factor.
		realInterval = b.randomize(baseInterval)
	}
}

// applyTransformers applies the error transformers to the error. If there are no transformers, the error
// is returned as is.
func (b *Backoff) applyTransformers(err error) error {
	if len(b.transformers) == 0 {
		return err
	}
	for _, t := range b.transformers {
		err = t(err)
	}
	return err
}

// randomize randomizes the interval based on the policy randomization factor. This can be be in the negative
// or positive direction.
func (b *Backoff) randomize(interval time.Duration) time.Duration {
	if b.policy.RandomizationFactor == 0 {
		return interval
	}

	// Calculate the random range.
	delta := b.policy.RandomizationFactor * float64(interval)
	min := interval - time.Duration(delta)
	max := interval + time.Duration(delta)

	// Get a random number in the range. So if RandomizationFactor is 0.5, and interval is 1s,
	// then we will get a random number between 0.5s and 1.5s.
	return time.Duration(rand.Int63n(int64(max-min))) + min // #nosec
}

// internalSpecified is used to check if the error message contains retry hints. If it does
// and it is more than the exponential retry timer, we will use the retry timer from the server.
// If it is less than the exponential retry timer, we will use the exponential retry timer.
// If the WithTextMatching() option is not used, we will always use the exponential retry timer.
func (b *Backoff) intervalSpecified(err error, expInterval time.Duration) time.Duration {
	// We always honor a retry internal specified in the error if it is greater than the exponential retry timer.
	serverInterval := b.errHasRetryInterval(err)
	if serverInterval > 0 {
		if serverInterval > expInterval {
			return serverInterval
		}
		return expInterval
	}
	return expInterval
}

// errHasRetryInterval looks to see if the error contains errors.ErrRetryAfter. If so, the one with
// the longest time is returned as a duration from now. If there are no errors.ErrRetryAfter, then
// 0 is returned.
func (b *Backoff) errHasRetryInterval(err error) time.Duration {
	var d time.Duration

	for {
		e := ErrRetryAfter{}
		if errors.As(err, &e) {
			newDur := b.until(e.Time)
			if newDur > d {
				d = newDur
			}
			err = errors.Unwrap(err)
			continue
		}
		break
	}
	return d
}

// ctxOK takes in a Context and interval and returns if we should continue execution.
// This returns false if a Context deadline is shorter than our interval or the Context
// has been cancelled or timed out.
func (b *Backoff) ctxOK(ctx context.Context, interval time.Duration) bool {
	if ctx.Err() != nil {
		return false
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return true
	}

	// We have a deadline, so let's see if we have time for another attempt.
	remaining := b.until(deadline)
	if remaining <= 0 {
		return false
	}

	// We have time for another attempt, but we need to see if we have time for the interval.
	if remaining < interval {
		return false
	}

	// We have time for the interval.
	return true
}
