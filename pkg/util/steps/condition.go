package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
)

// Functions that run as condition-steps return Error
// instead of InternalServerError
// Efforts are being made  to not have generic Hive errors but specific, actionable failure cases.
// Instead of providing Hive-specific error messages to customers, the below will send a timeout error message.
// The below functions are run during Install, Update, AdminUpdate.
var timeoutConditionErrors = map[string]string{
	"attachNSGs":                             "Failed to attach the ARO NSG to the cluster subnets.",
	"apiServersReady":                        "Kube API has not initialised successfully and is unavailable.",
	"minimumWorkerNodesReady":                "Minimum number of worker nodes have not been successfully created.",
	"operatorConsoleExists":                  "Console Cluster Operator has failed to initialize successfully.",
	"operatorConsoleReady":                   "Console Cluster Operator has not started successfully.",
	"clusterVersionReady":                    "Cluster Verion is not reporting status as ready.",
	"ingressControllerReady":                 "Ingress Cluster Operator has not started successfully.",
	"aroDeploymentReady":                     "ARO Cluster Operator has failed to initialize successfully.",
	"ensureAROOperatorRunningDesiredVersion": "ARO Cluster Operator is not running desired version.",
	"hiveClusterDeploymentReady":             "Timed out waiting for the condition to be ready.",
	"hiveClusterInstallationComplete":        "Timed out waiting for the condition to complete.",
}

// conditionFunction is a function that takes a context and returns whether the
// condition has been met and an error.
//
// Suitable for polling external sources for readiness.
type conditionFunction func(context.Context) (bool, bool, error)

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
	// timeout fires. Errors from `f` are returned directly unless the error
	// is ErrWaitTimeout. Internal ErrWaitTimeout errors are wrapped to avoid
	// confusion with wait.PollImmediateUntil's own behavior of returning
	// ErrWaitTimeout when the condition is not met.
	err := wait.PollImmediateUntil(pollInterval, func() (bool, error) {
		// We use the outer context, not the timeout context, as we do not want
		// to time out the condition function itself, only stop retrying once
		// timeoutCtx's timeout has fired.
		cnd, retry, cndErr := c.f(ctx)
		if errors.Is(cndErr, wait.ErrWaitTimeout) && !retry {
			return cnd, fmt.Errorf("condition encountered internal timeout: %w", cndErr)
		}

		return cnd, cndErr
	}, timeoutCtx.Done())

	if err != nil && !c.fail {
		log.Warnf("step %s failed but has configured 'fail=%t'. Continuing. Error: %s", c, c.fail, err.Error())
		return nil
	}
	if errors.Is(err, wait.ErrWaitTimeout) {
		return enrichConditionTimeoutError(c.f, err)
	}
	return err
}

// Instead of giving Generic, timed out waiting for a condition, error
// returns enriched error messages mentioned in timeoutConditionErrors
func enrichConditionTimeoutError(f conditionFunction, originalErr error) error {
	funcNameParts := strings.Split(FriendlyName(f), ".")
	funcName := strings.TrimSuffix(funcNameParts[len(funcNameParts)-1], "-fm")

	message, exists := timeoutConditionErrors[funcName]
	if !exists {
		return originalErr
	}
	return api.NewCloudError(
		http.StatusInternalServerError,
		api.CloudErrorCodeDeploymentFailed,
		"", message+" Please retry, and if the issue persists, raise an Azure support ticket",
	)
}

func (c conditionStep) String() string {
	return fmt.Sprintf("[Condition %s, timeout %s]", FriendlyName(c.f), c.timeout)
}

func (c conditionStep) metricsName() string {
	return fmt.Sprintf("condition.%s", shortName(FriendlyName(c.f)))
}
