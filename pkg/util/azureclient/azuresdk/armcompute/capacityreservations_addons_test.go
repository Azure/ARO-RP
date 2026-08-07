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

// Test_capacityReservationsClient_CreateOrUpdateAndWait_asyncOperation models the
// response sequence observed against real Compute ARM (API version 2025-04-01,
// validated in a disposable RG): the initial PUT returns 201 Created with an
// Azure-AsyncOperation header and properties.provisioningState=Creating, the
// operation URL is polled until it reports status=Succeeded, and — because this
// is a PUT — the SDK's async poller issues a final GET on the original resource
// URL to fetch the result (see azcore internal/pollers/async: Result() does a
// final GET on OrigURL for PATCH/PUT regardless of FinalStateVia).
func Test_capacityReservationsClient_CreateOrUpdateAndWait_asyncOperation(t *testing.T) {
	const asyncURL = "https://management.azure.com/subscriptions/sub-id/providers/Microsoft.Compute/locations/eastus/capacityReservationOperations/op1?api-version=2025-04-01"

	var putCount, pollCount, finalGetCount int
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut:
			putCount++
			resp := httpResponse(req, http.StatusCreated, `{"name":"cr1","properties":{"provisioningState":"Creating"}}`)
			resp.Header.Set("Azure-AsyncOperation", asyncURL)
			return resp, nil
		// Exact match is intentional, not fragile: PollHelper requests p.AsyncURL
		// (the literal Azure-AsyncOperation header value) unmodified — see azcore
		// internal/pollers/async.Poll(). If a future SDK bump changes that, this
		// test should fail rather than silently match on a loosened comparison.
		case req.Method == http.MethodGet && req.URL.String() == asyncURL:
			pollCount++
			return httpResponse(req, http.StatusOK, `{"status":"Succeeded"}`), nil
		case req.Method == http.MethodGet:
			finalGetCount++
			return httpResponse(req, http.StatusOK, `{"name":"cr1","properties":{"provisioningState":"Succeeded"}}`), nil
		default:
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	c := newTestCRClient(t, transport)
	err := c.CreateOrUpdateAndWait(context.Background(), "rg", "crg", "cr1", defaultCapacityReservation())
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

// Test_capacityReservationsClient_DeleteAndWait_asyncOperation models the delete
// response sequence observed against real Compute ARM (validated in a disposable
// RG): DELETE returns 202 Accepted with both Location and Azure-AsyncOperation
// headers. Azure-AsyncOperation takes priority for poller selection, and — unlike
// CreateOrUpdate's PUT — the async poller's Result() has no final-GET branch for
// DELETE (see azcore internal/pollers/async.Poller.Result(): only PATCH/PUT/POST
// trigger a final GET). Any GET beyond the async-operation poll would indicate a
// regression in that assumption, so the fake transport hard-fails on one.
func Test_capacityReservationsClient_DeleteAndWait_asyncOperation(t *testing.T) {
	const asyncURL = "https://management.azure.com/subscriptions/sub-id/providers/Microsoft.Compute/locations/eastus/capacityReservationOperations/op2?api-version=2025-04-01"
	const locURL = "https://management.azure.com/subscriptions/sub-id/providers/Microsoft.Compute/locations/eastus/capacityReservationOperationsStatus/op2?api-version=2025-04-01"

	var deleteCount, pollCount int
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodDelete:
			deleteCount++
			resp := httpResponse(req, http.StatusAccepted, "")
			resp.Header.Set("Azure-AsyncOperation", asyncURL)
			resp.Header.Set("Location", locURL)
			return resp, nil
		case req.Method == http.MethodGet && req.URL.String() == asyncURL:
			pollCount++
			return httpResponse(req, http.StatusOK, `{"status":"Succeeded"}`), nil
		default:
			t.Fatalf("unexpected request %s %s — DeleteAndWait's async poller must not issue a final GET for DELETE", req.Method, req.URL.String())
			return nil, nil
		}
	})

	c := newTestCRClient(t, transport)
	err := c.DeleteAndWait(context.Background(), "rg", "crg", "cr1")
	if err != nil {
		t.Fatalf("DeleteAndWait() unexpected error: %v", err)
	}
	if deleteCount != 1 {
		t.Errorf("expected 1 DELETE, got %d", deleteCount)
	}
	if pollCount != 1 {
		t.Errorf("expected 1 poll of the Azure-AsyncOperation URL, got %d", pollCount)
	}
}
