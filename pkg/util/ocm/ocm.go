package ocm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/ocm/api"
)

func GetClusterInfoWithUpgradePolices(ctx context.Context, apiInstance api.API) (*api.ClusterInfo, error) {
	clusterList, err := apiInstance.GetClusterList(ctx, map[string]string{})
	if err != nil {
		return nil, err
	}

	if len(clusterList.Items) != 1 {
		return nil, fmt.Errorf("expected 1 cluster, got %d", len(clusterList.Items))
	}

	clusterInfo := &clusterList.Items[0]
	ocmClusterID := clusterInfo.Id
	upgradePolicyList, err := apiInstance.GetClusterUpgradePolicies(ctx, ocmClusterID)
	if err != nil {
		return nil, err
	}

	for i, policy := range upgradePolicyList.Items {
		upgradePolicyState, err := apiInstance.GetClusterUpgradePolicyState(ctx, ocmClusterID, policy.Id)
		if err != nil {
			return nil, err
		}
		upgradePolicyList.Items[i].State = upgradePolicyState.State
		upgradePolicyList.Items[i].Description = upgradePolicyState.Description
	}
	clusterInfo.UpgradePolicies = upgradePolicyList.Items

	return clusterInfo, nil
}

func CancelClusterUpgradePolicy(ctx context.Context, apiInstance api.API, policyID string) (*api.CancelUpgradeResponse, error) {
	clusterInfo, err := GetClusterInfoWithUpgradePolices(ctx, apiInstance)
	if err != nil {
		return nil, err
	}

	ocmClusterID := clusterInfo.Id
	cancelUpgradeResponse, err := apiInstance.CancelClusterUpgradePolicy(ctx, ocmClusterID, policyID)
	if err != nil {
		return nil, err
	}

	return cancelUpgradeResponse, nil
}
