package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// actionFunction is a function that takes a context and returns an error.
//
// Suitable for performing tasks.
type actionFunction func(context.Context) error

// Action returns a Step which will execute the action function `f`. Errors from
// `f` are returned directly.
func Action(f actionFunction) Step {
	return actionStep{
		f: f,
	}
}

type actionStep struct {
	f actionFunction
}

func (s actionStep) run(ctx context.Context, log *logrus.Entry) error {
	return s.f(ctx)
}

func (s actionStep) String() string {
	return fmt.Sprintf("[Action %s]", FriendlyName(s.f))
}

func (s actionStep) metricsName() string {
	return fmt.Sprintf("action.%s", shortName(FriendlyName(s.f)))
}

func Concurrent(s []Step) Step {
	return concurrentStep{
		s: s,
	}
}

type concurrentStep struct {
	s []Step
}

func (s concurrentStep) run(ctx context.Context, log *logrus.Entry) error {
	eg := &sync.WaitGroup{}
	errlock := &sync.Mutex{}
	errs := []error{}
	for _, curStep := range s.s {
		eg.Go(func() {
			err := curStep.run(ctx, log)
			errlock.Lock()
			defer errlock.Unlock()
			errs = append(errs, err)
		})
	}
	eg.Wait()
	return errors.Join(errs...)
}

func (s concurrentStep) String() string {
	return fmt.Sprintf("[Actions %v]", s.s)
}

func (s concurrentStep) metricsName() string {
	return fmt.Sprintf("action.concurrent")
}
