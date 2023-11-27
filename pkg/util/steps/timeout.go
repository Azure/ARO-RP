package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

type CriticalRunner func(func() error) error
type actionWithCriticalSegmentFunction func(context.Context, CriticalRunner) error

// NetworkErrorRetryingAction will retry a critical section if there are network errors.
func NetworkErrorRetryingAction(action actionWithCriticalSegmentFunction) Step {
	return &networkErrorRetryingAction{
		f: action,
	}
}

type networkErrorRetryingAction struct {
	f            actionWithCriticalSegmentFunction
	retryTimeout time.Duration
}

func (s *networkErrorRetryingAction) run(ctx context.Context, log *logrus.Entry) error {
	var retryTimeout time.Duration

	if s.retryTimeout == time.Duration(0) {
		retryTimeout = 15 * time.Minute
	} else {
		retryTimeout = s.retryTimeout
	}

	retry := func(f func() error) error {
		return retry.OnError(
			wait.Backoff{
				Steps:    10,
				Duration: 2 * time.Second,
				Factor:   1.5,
				Cap:      retryTimeout,
			},
			func(err error) bool {
				// Consider interruptions (timeouts/EOFs/resets) and refusals as
				// retryable errors.
				if net.IsTimeout(err) ||
					net.IsProbableEOF(err) ||
					net.IsConnectionReset(err) ||
					net.IsConnectionRefused(err) {
					log.Printf("network error, retrying: %v", err)
					return true
				}
				return false
			}, func() error {
				// We don't pass in a new a timeout context as we do not want to
				// time out the action function itself, only stop retrying once
				// we have reached retryTimeout.
				return f()
			},
		)
	}

	return s.f(ctx, retry)
}

func (s *networkErrorRetryingAction) String() string {
	return fmt.Sprintf("[NetworkErrorRetryingAction %s]", FriendlyName(s.f))
}

func (s *networkErrorRetryingAction) metricsName() string {
	return fmt.Sprintf("networkerrorretryingaction.%s", shortName(FriendlyName(s.f)))
}
