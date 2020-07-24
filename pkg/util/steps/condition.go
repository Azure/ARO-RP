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
func Condition(f conditionFunction, timeout time.Duration) conditionStep {
	return conditionStep{
		f:       f,
		timeout: timeout,
	}
}

type conditionStep struct {
	f            conditionFunction
	timeout      time.Duration
	pollInterval time.Duration
}

func (c conditionStep) run(ctx context.Context, log *logrus.Entry) error {
	var pollInterval time.Duration
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// If no pollInterval has been set, use a default
	if c.pollInterval == time.Duration(0) {
		pollInterval = 10 * time.Second
	} else {
		pollInterval = c.pollInterval
	}

	// Run the condition function immediately, and then every
	// runner.pollInterval, until the condition returns true or timeoutCtx's
	// timeout fires. Errors from `f` are returned directly.
	return wait.PollImmediateUntil(pollInterval, func() (bool, error) {
		// We use the outer context, not the timeout context, as we do not want
		// to time out the condition function itself, only stop retrying once
		// timeoutCtx's timeout has fired.
		return c.f(ctx)
	}, timeoutCtx.Done())
}

func (c conditionStep) String() string {
	return fmt.Sprintf("[Condition %s, timeout %s]", friendlyName(c.f), c.timeout)
}
