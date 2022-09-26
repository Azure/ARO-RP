package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"os"
	"reflect"
	"testing"

	mgmtcontainerservice "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-10-01/containerservice"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	mock_containerservice "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/containerservice"
)

//go:embed testdata
var hiveEmbeddedFiles embed.FS

func TestProdHiveAdmin(t *testing.T) {
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
		},
	}

	resultPage := mgmtcontainerservice.NewManagedClusterListResultPage(managedClustersList, func(ctx context.Context, mclr mgmtcontainerservice.ManagedClusterListResult) (mgmtcontainerservice.ManagedClusterListResult, error) {
		return mgmtcontainerservice.ManagedClusterListResult{}, nil
	})
	// Note that ".AnyTimes()" is not added to the 'List' function below to ensure it can only
	// run once, which ensures that the caching for the credentials is taking place successfully
	mcc.EXPECT().List(gomock.Any()).Return(resultPage, nil)

	kc, err := hiveEmbeddedFiles.ReadFile("testdata/kubeconfigAdmin")
	if err != nil {
		t.Fatal(err)
	}

	kcresp := &[]mgmtcontainerservice.CredentialResult{
		{
			Name:  to.StringPtr("admin config"),
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

func TestOCMValidClientIDs(t *testing.T) {
	for _, tt := range []struct {
		name          string
		setupEnv      func(string)
		envIds        string
		wantClientIds []string
		wantErr       string
	}{
		{
			name: "env not set, empty array returned",
			setupEnv: func(clientIds string) {
				// no op
			},
			envIds:        "",
			wantClientIds: []string{},
			wantErr:       "",
		},
		{
			name: "env set, clientIds returned",
			setupEnv: func(clientIds string) {
				os.Setenv(ocmValidClientIDs, clientIds)
			},
			envIds:        `["abc"]`,
			wantClientIds: []string{"abc"},
			wantErr:       "",
		},
		{
			name: "invalid value for clientIds, error returned",
			setupEnv: func(clientIds string) {
				os.Setenv(ocmValidClientIDs, clientIds)
			},
			envIds:  `what is going on`,
			wantErr: "invalid character 'w' looking for beginning of value",
		},
	} {
		tt.setupEnv(tt.envIds)
		prod := &prod{}
		clientIds, err := prod.OCMValidClientIDs()

		if tt.wantErr != "" && tt.wantErr != err.Error() || tt.wantErr == "" && err != nil {
			t.Fatalf("Failed: expected error: %q but got: %q", tt.wantErr, err)
		}

		if !reflect.DeepEqual(clientIds, tt.wantClientIds) {
			t.Fatalf("clientIds do not match. Wanted: %q, got: %q", tt.wantClientIds, clientIds)
		}
	}
}
