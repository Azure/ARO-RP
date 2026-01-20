package cluster

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func MigrateInternalLoadBalancerZonesStep(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	tenantID := th.GetTenantID()

	_env := th.Environment()

	fpCredClusterTenant, err := _env.FPNewClientCertificateCredential(tenantID, nil)
	if err != nil {
		return nil, err
	}

	armLoadBalancersClient, err := armnetwork.NewLoadBalancersClient(r.SubscriptionID, fpCredClusterTenant, clientOptions)
	if err != nil {
		return nil, err
	}

	_, err := cluster.MigrateInternalLoadBalancerZones(ctx, th.Environment(), th.Log(), th.PatchOpenShiftClusterDocument, lbc, pls, resSkus, th.GetOpenshiftClusterDocument())

	return err

}
