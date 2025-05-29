package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"testing"

	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	armcontainerservice "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
	fake "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6/fake"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	utilcontainerservice "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerservice"
)

//go:embed testdata
var hiveEmbeddedFiles embed.FS

func SetupTestHarness() {

}

func TestProdHiveAdmin(t *testing.T) {
	ctx := context.Background()

	dummySubscription := "/fake/resource/id"

	managedClustersList := []*armcontainerservice.ManagedCluster{
		{
			Name:     to.Ptr("aro-aks-cluster-001"),
			Location: to.Ptr("eastus"),
			Properties: &armcontainerservice.ManagedClusterProperties{
				NodeResourceGroup: to.Ptr("rp-eastus-aks1"),
			},
		},
		{
			Name:     to.Ptr("aro-aks-cluster-002"),
			Location: to.Ptr("eastus"),
			Properties: &armcontainerservice.ManagedClusterProperties{
				NodeResourceGroup: to.Ptr("rp-eastus-aks2"),
			},
		},
	}

	kc, err := hiveEmbeddedFiles.ReadFile("testdata/kubeconfigAdmin")
	if err != nil {
		t.Fatal(err)
	}

	kcresp := []*armcontainerservice.CredentialResult{
		{
			Name:  to.Ptr("admin config"),
			Value: kc,
		},
	}

	resp := armcontainerservice.CredentialResults{
		Kubeconfigs: kcresp,
	}

	r := armcontainerservice.ManagedClustersClientListClusterAdminCredentialsResponse{
		CredentialResults: resp,
	}

	transporter := fake.ManagedClustersServer{
		NewListPager: func(options *armcontainerservice.ManagedClustersClientListOptions) (resp azfake.PagerResponder[armcontainerservice.ManagedClustersClientListResponse]) {
			pagerResp := azfake.PagerResponder[armcontainerservice.ManagedClustersClientListResponse]{}
			pagerResp.AddPage(http.StatusOK, armcontainerservice.ManagedClustersClientListResponse{
				ManagedClusterListResult: armcontainerservice.ManagedClusterListResult{
					Value: managedClustersList,
				},
			}, nil)

			return pagerResp
		},
		ListClusterAdminCredentials: func(ctx context.Context, resourceGroupName, resourceName string, options *armcontainerservice.ManagedClustersClientListClusterAdminCredentialsOptions) (resp azfake.Responder[armcontainerservice.ManagedClustersClientListClusterAdminCredentialsResponse], errResp azfake.ErrorResponder) {
			nilErr := azfake.ErrorResponder{}
			rResp := azfake.Responder[armcontainerservice.ManagedClustersClientListClusterAdminCredentialsResponse]{}
			rResp.SetResponse(http.StatusOK, r, nil)
			return rResp, nilErr
		},
	}

	mcc, err := utilcontainerservice.NewManagedClustersClientWithTransport(&azureclient.PublicCloud, dummySubscription, &azfake.TokenCredential{}, fake.NewManagedClustersServerTransport(&transporter))
	if err != nil {
		t.Fatal(err)
	}

	resultPage := mcc.List(ctx)

	if resultPage == nil {
		t.Fatal(errors.New("result page is nil"))
	}

	hasMore := resultPage.More()
	if !hasMore {
		t.Fatal(errors.New("result page has no results"))
	}

	results, err := resultPage.NextPage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Value) != len(managedClustersList) {
		t.Fatal(errors.New("results page has an invalid number of results"))
	}

	if *results.Value[0].Name != *managedClustersList[0].Name {
		t.Logf("expected %s to match %s", *results.Value[0].Name, *managedClustersList[0].Name)
		t.Fatal(errors.New("expected name of first managed clusters list to match first page results"))
	}

	adminCredsResult, err := mcc.ListClusterAdminCredentials(ctx, "rp-eastus", "aro-ak-cluster-001", "public")
	if err != nil {
		t.Fatal(err)
	}

	if len(adminCredsResult.Kubeconfigs) != 1 {
		t.Fatal(errors.New("invalid number of credentials returned"))
	}

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
