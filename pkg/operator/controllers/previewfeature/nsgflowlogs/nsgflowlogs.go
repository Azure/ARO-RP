package nsgflowlogs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/go-autorest/autorest/azure"

	aropreviewv1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/preview.aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func NewFeature(flowLogsClient armnetwork.FlowLogsClientInterface, kubeSubnets subnet.KubeManager, subnets armnetwork.SubnetsClient, location string) *nsgFlowLogsFeature {
	return &nsgFlowLogsFeature{
		kubeSubnets:    kubeSubnets,
		flowLogsClient: flowLogsClient,
		subnets:        subnets,
		location:       location,
	}
}

type nsgFlowLogsFeature struct {
	kubeSubnets    subnet.KubeManager
	subnets        armnetwork.SubnetsClient
	flowLogsClient armnetwork.FlowLogsClientInterface
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
		err = n.flowLogsClient.CreateOrUpdateAndWait(ctx, networkWatcherResource.ResourceGroup, networkWatcherResource.ResourceName, res.ResourceName, *flowLog, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *nsgFlowLogsFeature) newFlowLog(instance *aropreviewv1alpha1.PreviewFeature, nsgID string) *sdknetwork.FlowLog {
	// build a request as described here https://docs.microsoft.com/en-us/azure/network-watcher/network-watcher-nsg-flow-logging-rest#enable-network-security-group-flow-logs
	return &sdknetwork.FlowLog{
		Location: &n.location,
		Properties: &sdknetwork.FlowLogPropertiesFormat{
			TargetResourceID: &nsgID,
			Enabled:          to.Ptr(true),
			Format: &sdknetwork.FlowLogFormatParameters{
				Type:    to.Ptr(sdknetwork.FlowLogFormatTypeJSON),
				Version: to.Ptr(int32(instance.Spec.NSGFlowLogs.Version)),
			},
			RetentionPolicy: &sdknetwork.RetentionPolicyParameters{
				Days: &instance.Spec.NSGFlowLogs.RetentionDays,
			},
			StorageID: &instance.Spec.NSGFlowLogs.StorageAccountResourceID,
			FlowAnalyticsConfiguration: &sdknetwork.TrafficAnalyticsProperties{
				NetworkWatcherFlowAnalyticsConfiguration: &sdknetwork.TrafficAnalyticsConfigurationProperties{
					WorkspaceID:              &instance.Spec.NSGFlowLogs.TrafficAnalyticsLogAnalyticsWorkspaceID,
					TrafficAnalyticsInterval: to.Ptr(int32(instance.Spec.NSGFlowLogs.TrafficAnalyticsInterval.Truncate(time.Minute).Minutes())),
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

		err = n.flowLogsClient.DeleteAndWait(ctx, networkWatcherResource.ResourceGroup, networkWatcherResource.ResourceName, res.ResourceName, nil)
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
		r, err := arm.ParseResourceID(kubeSubnet.ResourceID)
		if err != nil {
			return nil, err
		}
		net, err := n.subnets.Get(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, nil)
		if err != nil {
			return nil, err
		}
		nsgs[*net.Properties.NetworkSecurityGroup.ID] = struct{}{}
	}
	return nsgs, nil
}
