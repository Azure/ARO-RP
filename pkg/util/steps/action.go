package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// actionFunction is a function that takes a context and returns an error.
//
// Suitable for performing tasks.
type actionFunction func(context.Context) error

// Action returns a Step which will execute the action function `f`. Errors from
// `f` are returned directly.
func Action(f actionFunction) actionStep {
	return actionStep{f}
}

type actionStep struct {
	f actionFunction
}

func (s actionStep) run(ctx context.Context, log *logrus.Entry) error {
	return s.f(ctx)
}
func (s actionStep) String() string {
	return fmt.Sprintf("[Action %s]", friendlyName(s.f))
}

// conditionalActionFunction is a function that takes a context and returns boolean.
//
// Suitable for condition based actions to prevent unnecessary execution
type conditionalActionFunction func(context.Context) bool

// ConditionalAction returns a wrapper Step which will execute
// `step` if the conditional function returns true
func ConditionalAction(f conditionalActionFunction, step Step) conditionalActionStep {
	return conditionalActionStep{
		step: step,
		f:    f,
	}
}

type conditionalActionStep struct {
	step Step
	f    conditionalActionFunction
}

func (s conditionalActionStep) run(ctx context.Context, log *logrus.Entry) error {
	if s.f(ctx) {
		return s.step.run(ctx, log)
	}
	return nil
}
func (s conditionalActionStep) String() string {
	return fmt.Sprintf("[ConditionalActionStep %s]", s.step)
}
