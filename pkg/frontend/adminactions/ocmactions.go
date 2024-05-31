package adminactions

import (
	"context"
	"github.com/Azure/ARO-RP/pkg/util/ocm"
	"github.com/Azure/ARO-RP/pkg/util/ocm/api"
)

type OCMActions interface {
	GetClusterInfoWithUpgradePolicies(ctx context.Context) (*api.ClusterInfo, error)
	CancelClusterUpgradePolicy(ctx context.Context, policyID string) (*api.CancelUpgradeResponse, error)
}

type ocmActions struct {
	clusterID   string
	apiInstance api.API
}

func NewOCMActions(clusterID, baseURL, token string) OCMActions {
	apiInstance := api.NewClient(clusterID, baseURL, token)
	return &ocmActions{
		clusterID:   clusterID,
		apiInstance: apiInstance,
	}
}

func (o *ocmActions) GetClusterInfoWithUpgradePolicies(ctx context.Context) (*api.ClusterInfo, error) {
	return ocm.GetClusterInfoWithUpgradePolices(ctx, o.apiInstance)
}

func (o *ocmActions) CancelClusterUpgradePolicy(ctx context.Context, policyID string) (*api.CancelUpgradeResponse, error) {
	return ocm.CancelClusterUpgradePolicy(ctx, o.apiInstance, policyID)
}
