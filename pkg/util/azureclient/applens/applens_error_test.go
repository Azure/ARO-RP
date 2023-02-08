package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

func TestAppLensErrorOnEmptyResponse(t *testing.T) {
	srv, close := NewTLSServer()
	defer close()
	srv.SetResponse(
		WithStatusCode(404))

	req, err := runtime.NewRequest(context.Background(), http.MethodGet, srv.URL())
	if err != nil {
		t.Fatal(err)
	}

	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{Transport: srv})
	resp, _ := pl.Do(req)

	var azErr *azcore.ResponseError
	if err := newAppLensError(resp); !errors.As(err, &azErr) {
		t.Fatalf("unexpected error type %T", err)
	}
	if azErr.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected status code %d", azErr.StatusCode)
	}
	if azErr.ErrorCode != "" {
		t.Errorf("unexpected error code %s", azErr.ErrorCode)
	}
	if azErr.RawResponse == nil {
		t.Error("unexpected nil RawResponse")
	}
}

func TestAppLensErrorOnNonJsonBody(t *testing.T) {
	srv, close := NewTLSServer()
	defer close()
	srv.SetResponse(
		WithBody([]byte("This is not JSON")),
		WithStatusCode(404))

	req, err := runtime.NewRequest(context.Background(), http.MethodGet, srv.URL())
	if err != nil {
		t.Fatal(err)
	}

	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{Transport: srv})
	resp, _ := pl.Do(req)

	var azErr *azcore.ResponseError
	if err := newAppLensError(resp); !errors.As(err, &azErr) {
		t.Fatalf("unexpected error type %T", err)
	}
	if azErr.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected status code %d", azErr.StatusCode)
	}
	if azErr.ErrorCode != "" {
		t.Errorf("unexpected error code %s", azErr.ErrorCode)
	}
	if azErr.RawResponse == nil {
		t.Error("unexpected nil RawResponse")
	}
	if !strings.Contains(azErr.Error(), "This is not JSON") {
		t.Error("missing error message")
	}
}

func TestAppLensErrorOnJsonBody(t *testing.T) {
	someError := &appLensErrorResponse{
		Code: "SomeCode",
	}

	jsonString, err := json.Marshal(someError)
	if err != nil {
		t.Fatal(err)
	}

	srv, close := NewTLSServer()
	defer close()
	srv.SetResponse(
		WithBody(jsonString),
		WithStatusCode(404))

	req, err := runtime.NewRequest(context.Background(), http.MethodGet, srv.URL())
	if err != nil {
		t.Fatal(err)
	}

	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{Transport: srv})
	resp, _ := pl.Do(req)

	var azErr *azcore.ResponseError
	if err := newAppLensError(resp); !errors.As(err, &azErr) {
		t.Fatalf("unexpected error type %T", err)
	}
	if azErr.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected status code %d", azErr.StatusCode)
	}
	if azErr.ErrorCode != someError.Code {
		t.Errorf("unexpected error code %s", azErr.ErrorCode)
	}
	if azErr.RawResponse == nil {
		t.Error("unexpected nil RawResponse")
	}
	if !strings.Contains(azErr.Error(), `"Code": "SomeCode"`) {
		t.Error("missing error JSON")
	}
}
