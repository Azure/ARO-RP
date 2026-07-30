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
)

func newTestCRClient(t *testing.T, transport transportFunc) CapacityReservationsClient {
	t.Helper()
	opts := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: transport,
			Retry:     policy.RetryOptions{MaxRetries: -1},
		},
	}
	c, err := NewCapacityReservationsClient("sub-id", &fakeazcore.TokenCredential{}, opts)
	if err != nil {
		t.Fatalf("NewCapacityReservationsClient: %v", err)
	}
	return c
}

func Test_capacityReservationsClient_CreateOrUpdateAndWait_success(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusOK, `{"name":"cr1","location":"eastus"}`), nil
	})

	c := newTestCRClient(t, transport)
	err := c.CreateOrUpdateAndWait(context.Background(), "rg", "crg", "cr1", defaultCapacityReservation())
	if err != nil {
		t.Errorf("CreateOrUpdateAndWait() unexpected error: %v", err)
	}
}

func Test_capacityReservationsClient_CreateOrUpdateAndWait_error(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusConflict, `{"error":{"code":"Conflict"}}`), nil
	})

	c := newTestCRClient(t, transport)
	err := c.CreateOrUpdateAndWait(context.Background(), "rg", "crg", "cr1", defaultCapacityReservation())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_capacityReservationsClient_DeleteAndWait_success(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusOK, ""), nil
	})

	c := newTestCRClient(t, transport)
	err := c.DeleteAndWait(context.Background(), "rg", "crg", "cr1")
	if err != nil {
		t.Errorf("DeleteAndWait() unexpected error: %v", err)
	}
}

func Test_capacityReservationsClient_DeleteAndWait_error(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusNotFound, `{"error":{"code":"ResourceNotFound"}}`), nil
	})

	c := newTestCRClient(t, transport)
	err := c.DeleteAndWait(context.Background(), "rg", "crg", "cr1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
