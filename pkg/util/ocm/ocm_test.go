package ocm_test

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	mock_api "github.com/Azure/ARO-RP/pkg/util/mocks/ocm/api"
	"github.com/Azure/ARO-RP/pkg/util/ocm"
	"github.com/Azure/ARO-RP/pkg/util/ocm/api"
)

func TestGetClusterInfoWithUpgradePolicies(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mock_api.NewMockAPI(ctrl)
	ctx := context.TODO()

	expectedClusterList := &api.ClusterList{
		Items: []api.ClusterInfo{
			{Id: "cluster-id"},
		},
	}

	expectedUpgradePolicies := &api.UpgradePolicyList{
		Items: []api.UpgradePolicy{
			{Id: "policy-id-1"},
			{Id: "policy-id-2"},
		},
	}

	expectedUpgradePolicyState := &api.UpgradePolicyState{
		Kind: "testKind",
		UpgradePolicyStatus: api.UpgradePolicyStatus{
			State:       "completed",
			Description: "Upgrade completed successfully",
		},
	}

	mockAPI.EXPECT().GetClusterList(ctx, map[string]string{}).Return(expectedClusterList, nil)
	mockAPI.EXPECT().GetClusterUpgradePolicies(ctx, expectedClusterList.Items[0].Id).Return(expectedUpgradePolicies, nil)
	mockAPI.EXPECT().GetClusterUpgradePolicyState(ctx, expectedClusterList.Items[0].Id, "policy-id-1").Return(expectedUpgradePolicyState, nil)
	mockAPI.EXPECT().GetClusterUpgradePolicyState(ctx, expectedClusterList.Items[0].Id, "policy-id-2").Return(expectedUpgradePolicyState, nil)

	clusterInfo, err := ocm.GetClusterInfoWithUpgradePolices(ctx, mockAPI)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if clusterInfo.Id != expectedClusterList.Items[0].Id {
		t.Errorf("expected cluster ID %s, got %s", expectedClusterList.Items[0].Id, clusterInfo.Id)
	}

	for _, policy := range clusterInfo.UpgradePolicies {
		if policy.State != "completed" {
			t.Errorf("expected state 'completed', got %s", policy.State)
		}
		if policy.Description != "Upgrade completed successfully" {
			t.Errorf("expected description 'Upgrade completed successfully', got %s", policy.Description)
		}
	}
}

func TestCancelClusterUpgradePolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mock_api.NewMockAPI(ctrl)
	ctx := context.TODO()

	expectedClusterInfo := &api.ClusterInfo{Id: "cluster-id"}
	clusterList := &api.ClusterList{Items: []api.ClusterInfo{*expectedClusterInfo}}

	cancelUpgradeResponse := &api.CancelUpgradeResponse{
		Kind:        "testKind",
		Value:       "cancelled",
		Description: "Manually cancelled by SRE",
	}

	expectedClusterList := &api.ClusterList{
		Items: []api.ClusterInfo{
			{Id: "cluster-id"},
		},
	}

	expectedUpgradePolicies := &api.UpgradePolicyList{
		Items: []api.UpgradePolicy{
			{Id: "policy-id-1"},
			{Id: "policy-id-2"},
		},
	}

	expectedUpgradePolicyState := &api.UpgradePolicyState{
		Kind: "testKind",
		UpgradePolicyStatus: api.UpgradePolicyStatus{
			State:       "completed",
			Description: "Upgrade completed successfully",
		},
	}

	mockAPI.EXPECT().GetClusterList(ctx, map[string]string{}).Return(clusterList, nil)
	mockAPI.EXPECT().GetClusterUpgradePolicies(ctx, expectedClusterList.Items[0].Id).Return(expectedUpgradePolicies, nil)
	mockAPI.EXPECT().GetClusterUpgradePolicyState(ctx, expectedClusterList.Items[0].Id, "policy-id-1").Return(expectedUpgradePolicyState, nil)
	mockAPI.EXPECT().GetClusterUpgradePolicyState(ctx, expectedClusterList.Items[0].Id, "policy-id-2").Return(expectedUpgradePolicyState, nil)
	mockAPI.EXPECT().CancelClusterUpgradePolicy(ctx, expectedClusterInfo.Id, "policy-id").Return(cancelUpgradeResponse, nil)

	cancelResponse, err := ocm.CancelClusterUpgradePolicy(ctx, mockAPI, "policy-id")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cancelResponse.Value != "cancelled" {
		t.Errorf("expected value 'cancelled', got %s", cancelResponse.Value)
	}
	if cancelResponse.Description != "Manually cancelled by SRE" {
		t.Errorf("expected description 'Manually cancelled by SRE', got %s", cancelResponse.Description)
	}
}
