package containerservice

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"testing"

	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	armcontainerservice "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6/fake"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func TestListClusterUserCredentialsEndpointSelection(t *testing.T) {
	for _, serverFqdn := range []string{"", "public", "private"} {
		t.Run("server-fqdn="+serverFqdn, func(t *testing.T) {
			called := false
			server := fake.ManagedClustersServer{
				ListClusterUserCredentials: func(_ context.Context, _, _ string, options *armcontainerservice.ManagedClustersClientListClusterUserCredentialsOptions) (azfake.Responder[armcontainerservice.ManagedClustersClientListClusterUserCredentialsResponse], azfake.ErrorResponder) {
					called = true
					if serverFqdn == "" {
						if options != nil && options.ServerFqdn != nil {
							t.Fatal("default endpoint selection must omit server-fqdn")
						}
					} else if options == nil || options.ServerFqdn == nil || *options.ServerFqdn != serverFqdn {
						t.Fatalf("explicit endpoint selection was not preserved: %#v", options)
					}
					var response azfake.Responder[armcontainerservice.ManagedClustersClientListClusterUserCredentialsResponse]
					response.SetResponse(http.StatusOK, armcontainerservice.ManagedClustersClientListClusterUserCredentialsResponse{}, nil)
					return response, azfake.ErrorResponder{}
				},
			}
			client, err := NewManagedClustersClientWithTransport(
				&azureclient.PublicCloud, "subscription", &azfake.TokenCredential{},
				fake.NewManagedClustersServerTransport(&server))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.ListClusterUserCredentials(context.Background(), "resource-group", "cluster", serverFqdn); err != nil {
				t.Fatal(err)
			}
			if !called {
				t.Fatal("expected the AKS user-credentials API to be called")
			}
		})
	}
}
