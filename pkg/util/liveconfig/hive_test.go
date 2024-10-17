package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"testing"

	armcontainerservice "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"

	"github.com/Azure/go-autorest/autorest/to"
	"go.uber.org/mock/gomock"

	mock_containerservice "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/containerservice"
)

//go:embed testdata
var hiveEmbeddedFiles embed.FS

func TestProdHiveAdmin(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	mcc := mock_containerservice.NewMockManagedClustersClient(controller)

	managedClustersList := armcontainerservice.ManagedClusterListResult{
		Value: &[]armcontainerservice.ManagedCluster{
			{
				Name:     to.StringPtr("aro-aks-cluster-001"),
				Location: to.StringPtr("eastus"),
				ManagedClusterProperties: &armcontainerservice.ManagedClusterProperties{
					NodeResourceGroup: to.StringPtr("rp-eastus-aks1"),
				},
			},
			{
				Name:     to.StringPtr("aro-aks-cluster-002"),
				Location: to.StringPtr("eastus"),
				ManagedClusterProperties: &armcontainerservice.ManagedClusterProperties{
					NodeResourceGroup: to.StringPtr("rp-eastus-aks2"),
				},
			},
		},
	}

	resultPage := armcontainerservice.NewManagedClusterListResultPage(managedClustersList, func(ctx context.Context, mclr armcontainerservice.ManagedClusterListResult) (armcontainerservice.ManagedClusterListResult, error) {
		return armcontainerservice.ManagedClusterListResult{}, nil
	})
	// Note that ".AnyTimes()" is not added to the 'List' function below to ensure it can only
	// run once, which ensures that the caching for the credentials is taking place successfully
	mcc.EXPECT().List(gomock.Any()).Return(resultPage, nil)

	kc, err := hiveEmbeddedFiles.ReadFile("testdata/kubeconfigAdmin")
	if err != nil {
		t.Fatal(err)
	}

	kcresp := &[]armcontainerservice.CredentialResult{
		{
			Name:  to.StringPtr("admin config"),
			Value: to.ByteSlicePtr(kc),
		},
	}

	resp := armcontainerservice.CredentialResults{
		Kubeconfigs: kcresp,
	}

	mcc.EXPECT().ListClusterAdminCredentials(gomock.Any(), "rp-eastus", "aro-aks-cluster-001", "public").Return(resp, nil)

	lc := NewProd("eastus", mcc)

	restConfig, err := lc.HiveRestConfig(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	// rudimentary loading checks
	if restConfig.Host != "https://api.admin.testing:6443" {
		t.Error("Invalid credentials returned for test 1")
	}

	if restConfig.BearerToken != "admin" {
		t.Error("Invalid admin BearerToken returned for test 1")
	}

	// Make a second call, so that it uses the cache
	restConfig2, err := lc.HiveRestConfig(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if restConfig2.Host != "https://api.admin.testing:6443" {
		t.Error("Invalid credentials returned for test 2")
	}

	if restConfig2.BearerToken != "admin" {
		t.Error("Invalid admin BearerToken returned for test 2")
	}
}
