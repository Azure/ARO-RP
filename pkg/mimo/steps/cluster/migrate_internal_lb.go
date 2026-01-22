package cluster

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func MigrateInternalLoadBalancerZonesStep(ctx context.Context) error {
	th, err := mimo.GetTaskContextWithAzureClients(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	lbc, err := th.LoadBalancersClient()
	if err != nil {
		return mimo.TerminalError(err)
	}

	pls, err := th.PrivateLinkServicesClient()
	if err != nil {
		return mimo.TerminalError(err)
	}

	resSkus, err := th.ResourceSKUsClient()
	if err != nil {
		return mimo.TerminalError(err)
	}

	_, err = cluster.MigrateInternalLoadBalancerZones(ctx, th.Environment(), th.Log(), th.PatchOpenShiftClusterDocument, lbc, pls, resSkus, th.GetOpenshiftClusterDocument())
	if err != nil {
		return mimo.TerminalError(err)
	}

	return nil
}
