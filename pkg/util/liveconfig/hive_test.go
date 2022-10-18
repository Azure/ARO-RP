package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"testing"

	mgmtcontainerservice "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-10-01/containerservice"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	mock_containerservice "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/containerservice"
)

//go:embed testdata
var hiveEmbeddedFiles embed.FS

func TestProdHive(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	mcc := mock_containerservice.NewMockManagedClustersClient(controller)

	managedClustersList := mgmtcontainerservice.ManagedClusterListResult{
		Value: &[]mgmtcontainerservice.ManagedCluster{
			{
				Name:     to.StringPtr("aro-aks-cluster-001"),
				Location: to.StringPtr("eastus"),
				ManagedClusterProperties: &mgmtcontainerservice.ManagedClusterProperties{
					NodeResourceGroup: to.StringPtr("rp-eastus-aks1"),
				},
			},
			{
				Name:     to.StringPtr("aro-aks-cluster-002"),
				Location: to.StringPtr("eastus"),
				ManagedClusterProperties: &mgmtcontainerservice.ManagedClusterProperties{
					NodeResourceGroup: to.StringPtr("rp-eastus-aks2"),
				},
			},
			{
				Name:     to.StringPtr("aro-aks-cluster-003"),
				Location: to.StringPtr("eastus"),
				ManagedClusterProperties: &mgmtcontainerservice.ManagedClusterProperties{
					NodeResourceGroup: to.StringPtr("rp-eastus-aks3"),
				},
			},
			{
				Name:     to.StringPtr("aro-aks-cluster-001"),
				Location: to.StringPtr("westus"),
				ManagedClusterProperties: &mgmtcontainerservice.ManagedClusterProperties{
					NodeResourceGroup: to.StringPtr("rp-eastus-aks1"),
				},
			},
			{
				Name:     to.StringPtr("aro-aks-cluster-001"),
				Location: to.StringPtr("australiaeast"),
				ManagedClusterProperties: &mgmtcontainerservice.ManagedClusterProperties{
					NodeResourceGroup: to.StringPtr("rp-eastus-aks2"),
				},
			},
			{
				Name:     to.StringPtr("aro-aks-cluster-001"),
				Location: to.StringPtr("brazilsouth"),
				ManagedClusterProperties: &mgmtcontainerservice.ManagedClusterProperties{
					NodeResourceGroup: to.StringPtr("rp-eastus-aks1"),
				},
			},
		},
	}

	resultPage := mgmtcontainerservice.NewManagedClusterListResultPage(managedClustersList, func(ctx context.Context, mclr mgmtcontainerservice.ManagedClusterListResult) (mgmtcontainerservice.ManagedClusterListResult, error) {
		return mgmtcontainerservice.ManagedClusterListResult{}, nil
	})
	mcc.EXPECT().List(gomock.Any()).Return(resultPage, nil)

	kc, err := hiveEmbeddedFiles.ReadFile("testdata/kubeconfig")
	if err != nil {
		t.Fatal(err)
	}

	kcresp := &[]mgmtcontainerservice.CredentialResult{
		{
			Name:  to.StringPtr("example"),
			Value: to.ByteSlicePtr(kc),
		},
	}

	resp := mgmtcontainerservice.CredentialResults{
		Kubeconfigs: kcresp,
	}

	mcc.EXPECT().ListClusterAdminCredentials(gomock.Any(), "rp-eastus", "aro-aks-cluster-001", "public").Return(resp, nil)

	lc := NewProd("eastus", mcc)

	restConfig, err := lc.HiveRestConfig(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	// rudimentary loading checks
	if restConfig.Host != "https://api.crc.testing:6443" {
		t.Error(restConfig.String())
	}

	if restConfig.BearerToken != "none" {
		t.Error(restConfig.String())
	}

	// Make a second call, so that it uses the cache
	restConfig2, err := lc.HiveRestConfig(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if restConfig2.Host != "https://api.crc.testing:6443" {
		t.Error(restConfig2.String())
	}
}
