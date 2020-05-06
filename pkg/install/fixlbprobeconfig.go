package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

type probe struct {
	lbName    string
	probeName string
	httpsPath string
}

func (i *Installer) fixLBProbeConfig(ctx context.Context, resourceGroup, lbName, probeName, path string) error {
	lb, err := i.loadbalancers.Get(ctx, resourceGroup, lbName, "")
	if err != nil {
		return err
	}

	for pix, probe := range *lb.LoadBalancerPropertiesFormat.Probes {
		if *probe.Name == probeName && probe.ProbePropertiesFormat.Protocol == mgmtnetwork.ProbeProtocolTCP {
			(*lb.LoadBalancerPropertiesFormat.Probes)[pix].ProbePropertiesFormat.Protocol = mgmtnetwork.ProbeProtocolHTTPS
			(*lb.LoadBalancerPropertiesFormat.Probes)[pix].RequestPath = to.StringPtr(path)
		}
	}

	return i.loadbalancers.CreateOrUpdateAndWait(ctx, resourceGroup, lbName, lb)
}

func (i *Installer) fixLBProbes(ctx context.Context) error {
	infraID := i.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro" // TODO: remove after deploy
	}
	resourceGroup := stringutils.LastTokenByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	fixes := []probe{
		{
			lbName:    infraID + "-internal-lb",
			probeName: "api-internal-probe",
			httpsPath: "/readyz",
		},
		{
			lbName:    infraID + "-internal-lb",
			probeName: "sint-probe",
			httpsPath: "/healthz",
		},
	}

	// the public LB with visibility != api.VisibilityPublic has no probes
	if i.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		fixes = append(fixes, probe{lbName: infraID + "-public-lb", probeName: "api-internal-probe", httpsPath: "/readyz"})
	}

	for _, f := range fixes {
		err := i.fixLBProbeConfig(ctx, resourceGroup, f.lbName, f.probeName, f.httpsPath)
		if err != nil {
			return err
		}
	}

	return nil
}
