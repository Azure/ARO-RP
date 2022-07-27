package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"encoding/base64"
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

	kc, err := hiveEmbeddedFiles.ReadFile("testdata/kubeconfig")
	if err != nil {
		t.Fatal(err)
	}
	enc := make([]byte, base64.StdEncoding.EncodedLen(len(kc)))
	base64.StdEncoding.Encode(enc, kc)

	kcresp := &[]mgmtcontainerservice.CredentialResult{
		{
			Name:  to.StringPtr("example"),
			Value: to.ByteSlicePtr(enc),
		},
	}

	resp := mgmtcontainerservice.CredentialResults{
		Kubeconfigs: kcresp,
	}

	mcc.EXPECT().ListClusterUserCredentials(gomock.Any(), "rp-eastus", "aro-aks-cluster-001", "").Return(resp, nil)

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
