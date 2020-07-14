package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
)

// OnlyInProd returns a wrapper Step which will only run the provided `step` if
// `env` is a production environment.
func OnlyInProd(env env.Interface, step Step) onlyInProd {
	return onlyInProd{
		env:  env,
		step: step,
	}
}

type onlyInProd struct {
	step Step
	env  env.Interface
}

func (s onlyInProd) run(ctx context.Context, log *logrus.Entry) error {
	if _, ok := s.env.(env.Dev); !ok {
		return s.step.run(ctx, log)
	} else {
		log.Infof("skipping %s as not in production", s.step)
		return nil
	}
}
func (s onlyInProd) String() string {
	return fmt.Sprintf("[OnlyInProd %s]", s.step)
}
