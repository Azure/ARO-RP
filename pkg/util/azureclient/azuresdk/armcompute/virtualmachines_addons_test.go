package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	fakeazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func newTestVMClient(t *testing.T, transport transportFunc) VirtualMachinesClient {
	t.Helper()
	opts := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: transport,
			Retry:     policy.RetryOptions{MaxRetries: -1},
		},
	}
	c, err := NewVirtualMachinesClient("sub-id", &fakeazcore.TokenCredential{}, opts)
	if err != nil {
		t.Fatalf("NewVirtualMachinesClient: %v", err)
	}
	return c
}

// Test_virtualMachinesClient_GetWithInstanceView_setsExpandParam verifies that
// GetWithInstanceView sets the $expand=instanceView query parameter, which is
// what actually causes the service to populate InstanceView on the response.
func Test_virtualMachinesClient_GetWithInstanceView_setsExpandParam(t *testing.T) {
	var gotExpand string
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		gotExpand = req.URL.Query().Get("$expand")
		return httpResponse(req, http.StatusOK, `{"name":"vm1","location":"eastus"}`), nil
	})

	c := newTestVMClient(t, transport)
	vm, err := c.GetWithInstanceView(context.Background(), "rg", "vm1")
	if err != nil {
		t.Fatalf("GetWithInstanceView() unexpected error: %v", err)
	}
	if gotExpand != "instanceView" {
		t.Errorf("expected $expand=instanceView, got %q", gotExpand)
	}
	if vm.Name == nil || *vm.Name != "vm1" {
		t.Errorf("expected VM name %q, got %v", "vm1", vm.Name)
	}
}

// Test_virtualMachinesClient_GetDefault_noExpandParam verifies that GetDefault
// does NOT request the InstanceView expansion, distinguishing it from
// GetWithInstanceView.
func Test_virtualMachinesClient_GetDefault_noExpandParam(t *testing.T) {
	var gotExpand string
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		gotExpand = req.URL.Query().Get("$expand")
		return httpResponse(req, http.StatusOK, `{"name":"vm1","location":"eastus"}`), nil
	})

	c := newTestVMClient(t, transport)
	vm, err := c.GetDefault(context.Background(), "rg", "vm1")
	if err != nil {
		t.Fatalf("GetDefault() unexpected error: %v", err)
	}
	if gotExpand != "" {
		t.Errorf("expected no $expand param, got %q", gotExpand)
	}
	if vm.Name == nil || *vm.Name != "vm1" {
		t.Errorf("expected VM name %q, got %v", "vm1", vm.Name)
	}
}

// Test_virtualMachinesClient_CreateOrUpdateAndWait_asyncOperation covers the
// representative ARM async-LRO path for CreateOrUpdateAndWait: a 2xx response
// carrying an Azure-AsyncOperation header, a poll of that URL until
// status=Succeeded, and — because this is a PUT — a final GET on the resource
// URL (azcore's async poller issues that final GET for PATCH/PUT regardless of
// FinalStateVia; see internal/pollers/async.Poller.Result()).
func Test_virtualMachinesClient_CreateOrUpdateAndWait_asyncOperation(t *testing.T) {
	const asyncURL = "https://management.azure.com/subscriptions/sub-id/providers/Microsoft.Compute/locations/eastus/operations/op1?api-version=2024-07-01"

	var putCount, pollCount, finalGetCount int
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut:
			putCount++
			resp := httpResponse(req, http.StatusOK, `{"name":"vm1","properties":{"provisioningState":"Updating"}}`)
			resp.Header.Set("Azure-AsyncOperation", asyncURL)
			return resp, nil
		case req.Method == http.MethodGet && req.URL.String() == asyncURL:
			pollCount++
			return httpResponse(req, http.StatusOK, `{"status":"Succeeded"}`), nil
		case req.Method == http.MethodGet:
			finalGetCount++
			return httpResponse(req, http.StatusOK, `{"name":"vm1","properties":{"provisioningState":"Succeeded"}}`), nil
		default:
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	c := newTestVMClient(t, transport)
	err := c.CreateOrUpdateAndWait(context.Background(), "rg", "vm1", armcompute.VirtualMachine{
		Location: pointerutils.ToPtr("eastus"),
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateAndWait() unexpected error: %v", err)
	}
	if putCount != 1 {
		t.Errorf("expected 1 PUT, got %d", putCount)
	}
	if pollCount != 1 {
		t.Errorf("expected 1 poll of the Azure-AsyncOperation URL, got %d", pollCount)
	}
	if finalGetCount != 1 {
		t.Errorf("expected 1 final GET on the resource URL, got %d", finalGetCount)
	}
}
