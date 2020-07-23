package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
)

// OnlyInEnv returns a wrapper Step which will only run the provided `step` if
// `env`'s type matches `environmentType`
func OnlyInEnv(env env.Interface, environmentType env.EnvironmentType, step Step) onlyInEnv {
	return onlyInEnv{
		env:             env,
		environmentType: environmentType,
		step:            step,
	}
}

type onlyInEnv struct {
	step            Step
	env             env.Interface
	environmentType env.EnvironmentType
}

func (s onlyInEnv) run(ctx context.Context, log *logrus.Entry) error {
	if s.env.Type()&s.environmentType != 0 {
		return s.step.run(ctx, log)
	} else {
		log.Infof("skipping %s as %s != %s", s.step, envTypeToString(s.env.Type()), envTypeToString(s.environmentType))
		return nil
	}
}

func (s onlyInEnv) String() string {
	return fmt.Sprintf("[OnlyInEnv %s %s]", envTypeToString(s.environmentType), s.step)
}

func envTypeToString(envType env.EnvironmentType) string {
	res := make([]string, 0, 3)

	vals := map[env.EnvironmentType]string{
		env.EnvironmentTypeDevelopment: "Development",
		env.EnvironmentTypeIntegration: "Integration",
		env.EnvironmentTypeProduction:  "Production",
	}

	for k, v := range vals {
		if k&envType != 0 {
			res = append(res, v)
		}
	}

	return strings.Join(res, " | ")
}
