package remotepdp

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	testhttp "github.com/Azure/ARO-RP/test/util/http"
)

func TestClientCreate(t *testing.T) {
	endpoint := "https://westus.authorization.azure.net/providers/Microsoft.Authorization/checkAccess?api-version=2021-06-01-preview"
	scope := "https://authorization.azure.net/.default"
	cred, err := azidentity.NewClientSecretCredential("888988bf-86f1-31ea-91cd-2d7cd011db48", "clientID", "clientSecret", nil)
	if err != nil {
		t.Error("Unable to create a new PDP client")
	}
	NewRemotePDPClient(endpoint, scope, cred)
}

func TestSuccessfulCallReturnsADecision(t *testing.T) {
	srv, close := testhttp.NewTLSServer()
	defer close()
	srv.SetResponse(
		testhttp.WithStatusCode(http.StatusOK),
	)

	client := createClientWithServer(srv)

	decision, err := client.CheckAccess(context.Background(), AuthorizationRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if decision == nil {
		t.Error("Successful calls should return an access decision")
	}
}

func TestFailedCallReturns(t *testing.T) {
	srv, close := testhttp.NewTLSServer()
	defer close()
	srv.SetResponse(
		testhttp.WithStatusCode(http.StatusUnauthorized),
	)

	client := createClientWithServer(srv)

	_, err := client.CheckAccess(context.Background(), AuthorizationRequest{})
	if err == nil {
		t.Error("Call resulting in a failure should return an error")
	}
}

func createClientWithServer(s *testhttp.Server) RemotePDPClient {
	return &remotePDPClient{
		endpoint: s.URL(),
		pipeline: runtime.NewPipeline(
			"remotepdpclient_test",
			"v1.0.0",
			runtime.PipelineOptions{},
			&policy.ClientOptions{Transport: s},
		),
	}
}
