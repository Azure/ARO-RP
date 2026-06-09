package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	fakeazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// transportFunc adapts a function to policy.Transporter so tests can return canned
// HTTP responses deterministically. The Azure SDK's generated fake server races on the
// 202 async-delete path ("send on closed channel"), so we drive the pipeline directly
// instead.
type transportFunc func(req *http.Request) (*http.Response, error)

func (f transportFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func httpResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func newTestCRGClient(t *testing.T, transport transportFunc) CapacityReservationGroupsClient {
	t.Helper()
	opts := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: transport,
			// Disable the pipeline's automatic retries so each test controls the exact
			// number of HTTP calls (e.g. the 500 Get below must not be silently retried).
			Retry: policy.RetryOptions{MaxRetries: -1},
		},
	}
	c, err := NewCapacityReservationGroupsClient("sub-id", &fakeazcore.TokenCredential{}, opts)
	if err != nil {
		t.Fatalf("NewCapacityReservationGroupsClient: %v", err)
	}
	return c
}

// Test_capacityReservationGroupsClient_Delete_202_then_404 verifies that when
// Delete returns 202 Accepted, pollCRGDeleted is invoked and returns nil once
// a subsequent Get returns 404.
func Test_capacityReservationGroupsClient_Delete_202_then_404(t *testing.T) {
	// Reduce poll interval for fast test execution
	orig := crgDeletePollInterval
	crgDeletePollInterval = time.Millisecond
	defer func() { crgDeletePollInterval = orig }()

	getCallCount := 0
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		switch req.Method {
		case http.MethodDelete:
			return httpResponse(req, http.StatusAccepted, ""), nil
		case http.MethodGet:
			getCallCount++
			if getCallCount < 3 {
				// Return 200 for the first two polls (still deleting)
				return httpResponse(req, http.StatusOK, "{}"), nil
			}
			// Third poll: 404 — deleted
			return httpResponse(req, http.StatusNotFound, `{"error":{"code":"ResourceNotFound"}}`), nil
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil, nil
		}
	})

	c := newTestCRGClient(t, transport)
	err := c.Delete(context.Background(), "rg", "crg")
	if err != nil {
		t.Errorf("Delete() unexpected error: %v", err)
	}
	if getCallCount < 3 {
		t.Errorf("expected at least 3 Get calls, got %d", getCallCount)
	}
}

// Test_capacityReservationGroupsClient_Delete_200_immediate verifies that a
// synchronous 200 OK from Delete returns immediately without polling.
func Test_capacityReservationGroupsClient_Delete_200_immediate(t *testing.T) {
	getCallCount := 0
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		switch req.Method {
		case http.MethodDelete:
			return httpResponse(req, http.StatusOK, ""), nil
		case http.MethodGet:
			getCallCount++
			return httpResponse(req, http.StatusOK, "{}"), nil
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil, nil
		}
	})

	c := newTestCRGClient(t, transport)
	err := c.Delete(context.Background(), "rg", "crg")
	if err != nil {
		t.Errorf("Delete() unexpected error: %v", err)
	}
	if getCallCount != 0 {
		t.Errorf("expected no Get calls for sync 200, got %d", getCallCount)
	}
}

// Test_capacityReservationGroupsClient_Delete_unexpectedError verifies that a
// non-202 error from Delete is propagated directly without triggering polling.
func Test_capacityReservationGroupsClient_Delete_unexpectedError(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			t.Fatal("unexpected Get call: a non-202 Delete error must not trigger polling")
		}
		return httpResponse(req, http.StatusConflict, `{"error":{"code":"CannotDeleteResource"}}`), nil
	})

	c := newTestCRGClient(t, transport)
	err := c.Delete(context.Background(), "rg", "crg")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var respErr *azcore.ResponseError
	if !errors.As(err, &respErr) || respErr.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 ResponseError, got %v", err)
	}
}

// Test_pollCRGDeleted_ctxCancelled verifies that pollCRGDeleted returns
// a context error when the context is cancelled before a 404 is observed.
func Test_pollCRGDeleted_ctxCancelled(t *testing.T) {
	orig := crgDeletePollInterval
	crgDeletePollInterval = time.Millisecond
	defer func() { crgDeletePollInterval = orig }()

	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodDelete {
			return httpResponse(req, http.StatusAccepted, ""), nil
		}
		// Always return 200 — resource never disappears
		return httpResponse(req, http.StatusOK, "{}"), nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	c := newTestCRGClient(t, transport)
	err := c.Delete(ctx, "rg", "crg")
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got %v", err)
	}
}

// Test_pollCRGDeleted_getError verifies that a non-404 error from Get during
// polling is propagated immediately.
func Test_pollCRGDeleted_getError(t *testing.T) {
	orig := crgDeletePollInterval
	crgDeletePollInterval = time.Millisecond
	defer func() { crgDeletePollInterval = orig }()

	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodDelete {
			return httpResponse(req, http.StatusAccepted, ""), nil
		}
		return httpResponse(req, http.StatusInternalServerError, `{"error":{"code":"InternalServerError"}}`), nil
	})

	c := newTestCRGClient(t, transport)
	err := c.Delete(context.Background(), "rg", "crg")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var respErr *azcore.ResponseError
	if !errors.As(err, &respErr) || respErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 ResponseError, got %v", err)
	}
}
