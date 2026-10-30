package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

const (
	resizeControlPlanePerNodeTimeout = 45 * time.Minute

	etcdHealthPollTimeout  = 10 * time.Minute
	etcdHealthPollInterval = 10 * time.Second
)

type resizeControlPlaneOperation struct {
	log                      *logrus.Entry
	k                        adminactions.KubeActions
	a                        adminactions.AzureActions
	desiredVMSize            string
	deallocateVM             bool
	clusterResourceGroupName string
}

// newResizeControlPlaneExecutionContext stays local because only the admin
// handler knows when resize work must outlive the incoming HTTP request.
// The resize operation (~45 min for 3 nodes) must complete even if the HTTP
// connection drops — the Geneva Actions client sets Timeout =
// InfiniteTimeSpan, but intermediary load balancers between the ACIS host and
// the RP VMSS can still close long-lived connections. ARM is not in this path;
// Geneva Actions calls the admin API directly via client certificate.
//
// Parallel invocation is unlikely — this is restricted to on-call SREs via
// Geneva Actions (ClientPlatformServiceOperator claim) and requires manual
// parameter entry. The real risk is not parallel calls but RP pod restart
// (OOM, node eviction, rolling update) or network partition during the
// operation, which would cancel the HTTP context and leave nodes mid-resize.
// Context detachment ensures the resize runs to completion before returning.
func newResizeControlPlaneExecutionContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.WithoutCancel(parent),
		time.Duration(api.ControlPlaneNodeCount)*resizeControlPlanePerNodeTimeout,
	)
}

func newResizeControlPlaneOperation(
	log *logrus.Entry,
	k adminactions.KubeActions,
	a adminactions.AzureActions,
	desiredVMSize string,
	deallocateVM bool,
	clusterResourceGroupName string,
) *resizeControlPlaneOperation {
	return &resizeControlPlaneOperation{
		log:                      log,
		k:                        k,
		a:                        a,
		desiredVMSize:            desiredVMSize,
		deallocateVM:             deallocateVM,
		clusterResourceGroupName: clusterResourceGroupName,
	}
}

func (o *resizeControlPlaneOperation) runStep(ctx context.Context, stepName string, step steps.Step) error {
	_, err := steps.Run(ctx, o.log, 0, []steps.Step{step}, nil, "")
	err = unwrapSyntheticStepRunnerError(err)
	if err != nil {
		return fmt.Errorf("%s: %w", stepName, err)
	}
	return nil
}

func unwrapSyntheticStepRunnerError(err error) error {
	var cloudErr *api.CloudError
	if errors.As(err, &cloudErr) &&
		cloudErr.Target == "encountered error" &&
		(cloudErr.Code == api.CloudErrorCodeInvalidServicePrincipalCredentials ||
			cloudErr.Code == api.CloudErrorCodeInternalServerError) {
		return errors.New(cloudErr.Message)
	}

	return err
}

func (o *resizeControlPlaneOperation) resizeNode(ctx context.Context, nodeName string) error {
	type resizeNodeStep struct {
		name string
		step steps.Step
	}

	stepEntries := []resizeNodeStep{
		{
			name: "cordon",
			step: steps.Action(func(ctx context.Context) error {
				return cordonNode(ctx, o.k, nodeName)
			}),
		},
		{
			name: "drain",
			step: steps.Action(func(ctx context.Context) error {
				return o.k.DrainNodeWithRetries(ctx, nodeName)
			}),
		},
		{
			name: "stop",
			step: steps.Action(func(ctx context.Context) error {
				return o.a.VMStopAndWait(ctx, nodeName, o.deallocateVM)
			}),
		},
		{
			name: "resize",
			step: steps.Action(func(ctx context.Context) error {
				return o.a.VMResize(ctx, nodeName, o.desiredVMSize)
			}),
		},
		{
			name: "start",
			step: steps.Action(func(ctx context.Context) error {
				return o.a.VMStartAndWait(ctx, nodeName)
			}),
		},
		{
			name: "waitReady",
			step: steps.Action(func(ctx context.Context) error {
				return waitForNodeReady(ctx, o.log, o.k, nodeName)
			}),
		},
		{
			name: "waitEtcd",
			step: steps.Action(func(ctx context.Context) error {
				return waitForEtcdHealthy(ctx, o.log, o.k)
			}),
		},
		{
			name: "uncordon",
			step: steps.Action(func(ctx context.Context) error {
				return uncordonNode(ctx, o.k, nodeName)
			}),
		},
		{
			name: "updateMachine",
			step: steps.Action(func(ctx context.Context) error {
				return updateMachineVMSize(ctx, o.k, nodeName, o.desiredVMSize)
			}),
		},
		{
			name: "updateNodeLabels",
			step: steps.Action(func(ctx context.Context) error {
				return setNodeInstanceTypeLabels(ctx, o.k, nodeName, o.desiredVMSize)
			}),
		},
	}

	for _, step := range stepEntries {
		if err := o.runStep(ctx, step.name, step.step); err != nil {
			return err
		}
	}

	return nil
}

func waitForEtcdHealthy(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions) error {
	var lastErr error
	waitErr := wait.PollUntilContextTimeout(ctx, etcdHealthPollInterval, etcdHealthPollTimeout, true, func(innerCtx context.Context) (bool, error) {
		lastErr = validateEtcdHealth(innerCtx, k)
		if lastErr != nil {
			log.Infof("Waiting for etcd to become healthy: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if waitErr != nil {
		return errors.Join(waitErr, lastErr)
	}
	return nil
}

func ensureControlPlaneAndEtcdHealthy(ctx context.Context, k adminactions.KubeActions, nodeNames []string) error {
	if err := ensureControlPlaneNodesReadyAndSchedulable(ctx, k, nodeNames); err != nil {
		return err
	}
	return validateEtcdHealth(ctx, k)
}
