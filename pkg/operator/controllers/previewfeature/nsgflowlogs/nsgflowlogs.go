package nsgflowlogs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"

	aropreviewv1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/preview.aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func NewFeature(flowLogsClient network.FlowLogsClient, kubeSubnets subnet.KubeManager, subnets subnet.Manager, location string) *nsgFlowLogsFeature {
	return &nsgFlowLogsFeature{
		kubeSubnets:    kubeSubnets,
		flowLogsClient: flowLogsClient,
		subnets:        subnets,
		location:       location,
	}
}

type nsgFlowLogsFeature struct {
	kubeSubnets    subnet.KubeManager
	subnets        subnet.Manager
	flowLogsClient network.FlowLogsClient
	location       string
}

func (n *nsgFlowLogsFeature) Name() string {
	return "nsgFlowLogsFeature"
}

func (n *nsgFlowLogsFeature) Reconcile(ctx context.Context, instance *aropreviewv1alpha1.PreviewFeature) error {
	if instance.Spec.NSGFlowLogs == nil {
		return nil
	}

	if !instance.Spec.NSGFlowLogs.Enabled {
		return n.Disable(ctx, instance)
	}

	return n.Enable(ctx, instance)
}

func (n *nsgFlowLogsFeature) Enable(ctx context.Context, instance *aropreviewv1alpha1.PreviewFeature) error {
	nsgs, err := n.getNSGs(ctx)
	if err != nil {
		return err
	}

	for nsgID := range nsgs {
		networkWatcherResource, err := azure.ParseResourceID(instance.Spec.NSGFlowLogs.NetworkWatcherID)
		if err != nil {
			return err
		}

		flowLog := n.newFlowLog(instance, nsgID)

		res, err := azure.ParseResourceID(nsgID)
		if err != nil {
			return err
		}
		err = n.flowLogsClient.CreateOrUpdateAndWait(ctx, networkWatcherResource.ResourceGroup, networkWatcherResource.ResourceName, res.ResourceName, *flowLog)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *nsgFlowLogsFeature) newFlowLog(instance *aropreviewv1alpha1.PreviewFeature, nsgID string) *mgmtnetwork.FlowLog {
	// build a request as described here https://docs.microsoft.com/en-us/azure/network-watcher/network-watcher-nsg-flow-logging-rest#enable-network-security-group-flow-logs
	return &mgmtnetwork.FlowLog{
		Location: to.StringPtr(n.location),
		FlowLogPropertiesFormat: &mgmtnetwork.FlowLogPropertiesFormat{
			TargetResourceID: to.StringPtr(nsgID),
			Enabled:          to.BoolPtr(true),
			Format: &mgmtnetwork.FlowLogFormatParameters{
				Type:    mgmtnetwork.JSON,
				Version: to.Int32Ptr(int32(instance.Spec.NSGFlowLogs.Version)),
			},
			RetentionPolicy: &mgmtnetwork.RetentionPolicyParameters{
				Days: to.Int32Ptr(instance.Spec.NSGFlowLogs.RetentionDays),
			},
			StorageID: to.StringPtr(instance.Spec.NSGFlowLogs.StorageAccountResourceID),
			FlowAnalyticsConfiguration: &mgmtnetwork.TrafficAnalyticsProperties{
				NetworkWatcherFlowAnalyticsConfiguration: &mgmtnetwork.TrafficAnalyticsConfigurationProperties{
					WorkspaceID:              to.StringPtr(instance.Spec.NSGFlowLogs.TrafficAnalyticsLogAnalyticsWorkspaceID),
					TrafficAnalyticsInterval: to.Int32Ptr(int32(instance.Spec.NSGFlowLogs.TrafficAnalyticsInterval.Truncate(time.Minute).Minutes())),
				},
			},
		},
	}
}

func (n *nsgFlowLogsFeature) Disable(ctx context.Context, instance *aropreviewv1alpha1.PreviewFeature) error {
	networkWatcherResource, err := azure.ParseResourceID(instance.Spec.NSGFlowLogs.NetworkWatcherID)
	if err != nil {
		return err
	}

	nsgs, err := n.getNSGs(ctx)
	if err != nil {
		return err
	}

	for nsgID := range nsgs {
		res, err := azure.ParseResourceID(nsgID)
		if err != nil {
			return err
		}

		err = n.flowLogsClient.DeleteAndWait(ctx, networkWatcherResource.ResourceGroup, networkWatcherResource.ResourceName, res.ResourceName)
		if err != nil {
			return err
		}
	}

	return nil
}

// getNSGs collects all unique NSG IDs. By default, master and worker subnets will use the same NSG.
func (n *nsgFlowLogsFeature) getNSGs(ctx context.Context) (map[string]struct{}, error) {
	subnets, err := n.kubeSubnets.List(ctx)
	if err != nil {
		return nil, err
	}

	nsgs := map[string]struct{}{}
	for _, kubeSubnet := range subnets {
		net, err := n.subnets.Get(ctx, kubeSubnet.ResourceID)
		if err != nil {
			return nil, err
		}
		nsgs[*net.NetworkSecurityGroup.ID] = struct{}{}
	}
	return nsgs, nil
}
