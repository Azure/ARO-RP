package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
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
func Condition(f conditionFunction, timeout time.Duration, fail bool) Step {
	return conditionStep{
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
	err := wait.PollImmediateUntil(pollInterval, func() (bool, error) {
		// We use the outer context, not the timeout context, as we do not want
		// to time out the condition function itself, only stop retrying once
		// timeoutCtx's timeout has fired.
		return c.f(ctx)
	}, timeoutCtx.Done())

	if err != nil && !c.fail {
		log.Warnf("step %s failed but has configured 'fail=%t'. Continuing. Error: %s", c, c.fail, err.Error())
		return nil
	}
	if err == wait.ErrWaitTimeout {
		return c.enrichConditionTimeoutError()
	}
	return err
}

func (c conditionStep) String() string {
	return fmt.Sprintf("[Condition %s, timeout %s]", friendlyName(c.f), c.timeout)
}

func (c conditionStep) enrichConditionTimeoutError() error {
	funcNameParts := strings.Split(friendlyName(c.f), ".")
	funcName := strings.Replace(funcNameParts[len(funcNameParts)-1], "-fm", "", 1)

	message := "timed out polling for a prerequisite cluster condition to resolve"

	// All functions passed as a `conditionFunction`: https://github.com/Azure/ARO-RP/blob/master/pkg/cluster/condition.go
	// should have a case block below with a generic error message for the condition when it doesn't resolve
	switch funcName {
	case "apiServersReady":
		message = "Kube API has not initialised successfully and is unavailable"
	case "aroDeploymentReady":
		message = "ARO Cluster Operator has failed to initialise successfully"
	case "bootstrapConfigMapReady":
		message = "bootstrap configuration required to create new cluster nodes is unavailable"
	case "clusterVersionReady":
		message = "Cluster Version Operator has not started successfully"
	case "ingressControllerReady":
		message = "Ingress Controller has not initialised successfully"
	case "minimumWorkerNodesReady":
		message = "minimum number of worker nodes have not been successfully created"
	case "operatorConsoleExists":
		message = "Console Cluster Operator has failed to initialise successfully"
	case "operatorConsoleReady":
		message = "Console Cluster Operator has not started successfully"
	}

	return api.NewCloudError(
		http.StatusFailedDependency,
		api.CloudErrorCodeDeploymentFailed,
		"", message,
	)
}
