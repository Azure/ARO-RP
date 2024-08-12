/*
Package exponential provides an exponential backoff mechanism. Most useful when setting a single policy
for all retries within a package or set of packages.

This package comes with a default policy, but can be customized for your own needs.

There is no maximum retries here, as exponential retries should be based on a maximum delay. This is set
via a Context timeout. Note that the Context timeout is some point in the future after which the operation
will not be retried. But setting 30 * seconds does not mean that the Retry() will return after 30 seconds.
It means that after Retry() is called, no attempt will be made after 30 seconds from that point. If the first
call takes 30 seconds and then fails, no retries will happen. If the first call takes 29 seconds and then fails,
the second call may or may not happen depending on policy settings.

And error returned will be the last error returned by the Op.
It will not be a context.Canceled or context.DeadlineExceeded error if the retry timer was cancelled. However it may still yield
a context error if the Op returns a context error. Error.Cancelled() tells you if the retry was cancelled.
Error.IsCancelled() tells you if the last error returned by the Op was a context error.

To understand the consequences of using any specific policy, we provide a tool to generate a time table
for a given policy. This can be used to understand the consequences of a policy.
It is located in the timetable sub-package. Here is sample output giving the progression to the
maximum interval for a policy with the default settings:

	Generating TimeTable for -1 attempts and the following settings:
	{
		"InitialInterval":     1000000000, // 100 * time.Millisecond
		"Multiplier":          2,
		"RandomizationFactor": 0.5,
		"MaxInterval":         60000000000 // 60 * time.Second,
	}

	=============
	= TimeTable =
	=============
	+---------+----------+-------------+-------------+
	| ATTEMPT | INTERVAL | MININTERVAL | MAXINTERVAL |
	+---------+----------+-------------+-------------+
	|       1 |       0s |          0s |          0s |
	|       2 |       1s |       500ms |        1.5s |
	|       3 |       2s |          1s |          3s |
	|       4 |       4s |          2s |          6s |
	|       5 |       8s |          4s |         12s |
	|       6 |      16s |          8s |         24s |
	|       7 |      32s |         16s |         48s |
	|       8 |     1m0s |         30s |       1m30s |
	+---------+----------+-------------+-------------+
	|         |  MINTIME |     MAXTIME |             |
	|         |          |      1M1.5S |      3M4.5S |
	+---------+----------+-------------+-------------+

Attempt is the retry attempt, with 1 being the first (not 0). Interval is the calculated interval
without randomization. MinInterval is the minimum interval after randomization.
MaxInterval is the maximum interval after randomization. MINTIME and MAXTIME are the minimum and maximum
that would be taken to reach that the last attempt listed.

Documentation for the timetable application is in the timetable/ directory.

The following is a list of examples of how to use this package, it is not exhaustive.

Example: With default policy and maximum time of 30 seconds while capturing a return value:

	boff := exponential.New()

	// Captured return data.
	var data Data

	// This sets the time in which to retry to 30 seconds. This is based on the parent context, so a cancel
	// on the parent cancels this context.
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)

	err := boff.Retry(ctx, func(ctx context.Context, r Record) error {
		var err error
		data, err = getData(ctx)
		return err
	})
	cancel() // Always cancel the context when done to avoid lingering goroutines. Avoid defer.

	if err != nil {
		// Handle the error.
		// This will always contain the last error returned by the Op, not a context error unless
		// the last error by the Op was a context error.
		// If the retry was cancelled, you can detect this with errors.Is(err, ErrRetryCancelled).
		// You can determine if this was a permanent error with errors.Is(err, ErrPermanent).
	}

Example: With the default policy, maximum execution time of 30 seconds and each attempt can take
up to 5 seconds:

	boff := exponential.New()

	var data Data

	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)

	err := boff.Retry(ctx, func(ctx context.Context, r Record) error {
		var err error
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		data, err = getData(callCtx)
		callCancel()
		return err
	})
	cancel() // Always cancel the context when done to avoid lingering goroutines.
	...

Example: Retry forever:

	boff := exponential.New()
	var data Data

	err := boff.Retry(ctx, func(ctx context.Context, r Record) error {
		var err error
		data, err = getData(ctx)
		return err
	})
	cancel()
	...

Example: Same as before but with a permanent error that breaks the retries:

	...
	err := exponential.Retry(ctx, func(ctx context.Context, r Record) error {
		var err error
		data, err = getData(ctx)
		if err != nil && err == badError {
			return fmt.Errorrf("%w: %w", err, exponential.ErrPermanent)
		}
		return err
	})
	cancel()
	...

Example: No return data:

	err := exponential.Retry(ctx, func(ctx context.Context, r Record) error {
		return doSomeOperation(ctx)
	})
	cancel()
	...

Example: Create a custom policy:

	policy := exponential.Policy{
		InitialInterval:     1 * time.Second,
		Multiplier:          2,
		RandomizationFactor: 0.2,
		MaxInterval:         30 * time.Second,
	}
	boff := exponential.New(exponential.WithPolicy(policy))
	...

Example: Retry a call that fails, but honor the service's retry timer:

	...
	err := exponential.Retry(ctx, func(ctx context.Context, r Record) error {
		resp, err := client.Call(ctx, req)
		if err != nil {
			// extractRetry is a function that extracts the retry time from the error the server sends.
			// This might also be in the body of an http.Response or in some header. Just think of
			// extractRetryTime as a placeholder for whatever that is.
			t := extractRetryTime(err)
			return ErrRetryAfter{Time: t, Err: err}
		}
		return nil
	})

Example: Test a function without any delay that eventually succeeds

	boff := exponential.New(exponential.WithTesting())

	var data Data

	err := boff.Retry(ctx, func(ctx context.Context, r Record) error {
		data, err := getData(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	cancel()

Example: Test a function that eventually fails with permanent error

	boff := exponential.New(exponential.WithTesting())

	var data Data

	err := boff.Retry(ctx, func(ctx context.Context, r Record) error {
		data, err := getData(ctx)
		if err != nil {
			return fmt.Errorrf("%w: %w", err, exponential.ErrPermanent)
		}
		return nil
	}
	cancel()

Example: Test a function around a gRPC call that fails on certain status codes using an ErrTransformer

	boff := exponential.New(
		WithErrTransformer(grpc.New()), // find this in the helpers sub-package
	)

	ctx, cancel := context.WithTimeout(parentCtx, 1*time.Minute)

	req := &pb.HelloRequest{Name: "John"}
	var resp *pb.HelloReply{}

	err := boff.Retry(ctx, func(ctx context.Context, r Record) error {
		var err error
		resp, err = client.Call(ctx, req)
		return err
	})
	cancel() // Always cancel the context when done to avoid lingering goroutines.
*/
package exponential
