package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (i *Installer) fixLBProbeConfig(ctx context.Context, resourceGroup, lbName string) error {
	lb, err := i.loadbalancers.Get(ctx, resourceGroup, lbName, "")
	if err != nil {
		return err
	}

	if lb.LoadBalancerPropertiesFormat == nil || lb.LoadBalancerPropertiesFormat.Probes == nil {
		return nil
	}

	var changed bool

loop:
	for pix, probe := range *lb.LoadBalancerPropertiesFormat.Probes {
		var path string

		switch *probe.Name {
		case "api-internal-probe":
			path = "/readyz"
		case "sint-probe":
			path = "/healthz"
		default:
			continue loop
		}

		if probe.ProbePropertiesFormat.Protocol != mgmtnetwork.ProbeProtocolHTTPS {
			(*lb.LoadBalancerPropertiesFormat.Probes)[pix].ProbePropertiesFormat.Protocol = mgmtnetwork.ProbeProtocolHTTPS
			changed = true
		}

		if probe.RequestPath == nil || *probe.RequestPath != path {
			(*lb.LoadBalancerPropertiesFormat.Probes)[pix].RequestPath = &path
			changed = true
		}
	}

	if !changed {
		return nil
	}

	return i.loadbalancers.CreateOrUpdateAndWait(ctx, resourceGroup, lbName, lb)
}

func (i *Installer) fixLBProbes(ctx context.Context) error {
	infraID := i.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	resourceGroup := stringutils.LastTokenByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	for _, lbName := range []string{
		infraID + "-public-lb",
		infraID + "-internal-lb",
	} {
		err := i.fixLBProbeConfig(ctx, resourceGroup, lbName)
		if err != nil {
			return err
		}
	}

	return nil
}
