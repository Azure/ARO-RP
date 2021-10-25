package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

// conditionFunction is a function that takes a context and returns whether the
// condition has been met and an error.
//
// Suitable for polling external sources for readiness.
type conditionFunction func(context.Context) (bool, error)

// Condition returns a Step suitable for checking whether subsequent Steps can
// be executed.
//
// The Condition will execute f repeatedly (every Runner.pollInterval), timing
// out with a failure when more time than the provided timeout has elapsed
// without f returning (true, nil). Errors from `f` are returned directly.
// If fail is set to false - it will not fail after timeout.
func Condition(f conditionFunction, timeout time.Duration, fail bool) *conditionStep {
	return &conditionStep{
		f:       f,
		fail:    fail,
		timeout: timeout,
	}
}

type conditionStep struct {
	f            conditionFunction
	fail         bool
	timeout      time.Duration
	pollInterval time.Duration
}

func (c *conditionStep) setPollInterval(t time.Duration) {
	c.pollInterval = t
}

func (c *conditionStep) setTimeout(t time.Duration) {
	c.timeout = t
}

func (c *conditionStep) run(ctx context.Context, log *logrus.Entry) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Run the condition function immediately, and then every
	// runner.pollInterval, until the condition returns true or timeoutCtx's
	// timeout fires. Errors from `f` are returned directly.
	err := wait.PollImmediateUntil(c.pollInterval, func() (bool, error) {
		// We use the outer context, not the timeout context, as we do not want
		// to time out the condition function itself, only stop retrying once
		// timeoutCtx's timeout has fired.
		return c.f(ctx)
	}, timeoutCtx.Done())

	if err != nil && !c.fail {
		log.Warnf("step %s failed but has configured 'fail=%t'. Continuing. Error: %s", c, c.fail, err.Error())
		return nil
	}
	return err
}

func (c *conditionStep) String() string {
	return fmt.Sprintf("[Condition %s, timeout %s]", friendlyName(c.f), c.timeout)
}
