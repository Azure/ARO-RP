package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
)

type listStep struct {
	s []Step
}

func (s listStep) run(ctx context.Context, log *logrus.Entry) error {
	return errors.New("cannot be run directly")
}

func (s listStep) String() string {
	return "[ListAction]"
}

func (s listStep) metricsName() string {
	return "listaction"
}

func ListStep(s []Step) Step {
	return listStep{
		s: s,
	}
}
