package nsgflowlogs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	aropreviewv1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/preview.aro.openshift.io/v1alpha1"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
)

func NewFeature(flowLogsClient network.FlowLogsClient) *nsgFlowLogsFeature {
	return &nsgFlowLogsFeature{
		flowLogsClient: flowLogsClient,
	}
}

type nsgFlowLogsFeature struct {
	flowLogsClient network.FlowLogsClient
}

func (n *nsgFlowLogsFeature) Name() string {
	return "nsgFlowLogsFeature"
}

func (n *nsgFlowLogsFeature) Reconcile(ctx context.Context, instance *aropreviewv1alpha1.PreviewFeature) error {
	if instance.Spec.NSGFlowLogs == nil {
		return nil
	}

	if !instance.Spec.NSGFlowLogs.Enabled {
		return n.Disable(instance)
	}

	return n.Enable(instance)
}

func (n *nsgFlowLogsFeature) Enable(instance *aropreviewv1alpha1.PreviewFeature) error {
	// TODO: Implement
	return nil
}

func (n *nsgFlowLogsFeature) Disable(instance *aropreviewv1alpha1.PreviewFeature) error {
	// TODO: Implement
	return nil
}
