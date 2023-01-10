package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

func TestEnsureErrorIsGeneratedOnResponse(t *testing.T) {
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

	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}
	_, err = client.sendPostRequest(context.Background(), &GetDetectorOptions{}, nil)
	if err == nil {
		t.Fatal("Expected error")
	}

	asError := err.(*azcore.ResponseError)
	if asError.ErrorCode != someError.Code {
		t.Errorf("Expected %v, but got %v", someError.Code, asError.ErrorCode)
	}

	if err.Error() != asError.Error() {
		t.Errorf("Expected %v, but got %v", err.Error(), asError.Error())
	}
}

func TestEnsureErrorIsNotGeneratedOnResponse(t *testing.T) {
	srv, close := NewTLSServer()
	defer close()
	srv.SetResponse(
		WithStatusCode(200))

	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}
	_, err := client.sendPostRequest(context.Background(), &GetDetectorOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRequestEnricherIsCalled(t *testing.T) {
	srv, close := NewTLSServer()
	defer close()
	srv.SetResponse(
		WithStatusCode(200))

	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}

	addHeader := func(r *policy.Request) {
		r.Raw().Header.Add("my-header", "12345")
	}

	req, err := client.createRequest(context.Background(), http.MethodGet, &GetDetectorOptions{}, addHeader)
	if err != nil {
		t.Fatal(err)
	}

	if req.Raw().Header.Get("my-header") != "12345" {
		t.Errorf("Expected %v, but got %v", "12345", req.Raw().Header.Get("my-header"))
	}
}

func TestNoOptionsIsCalled(t *testing.T) {
	srv, close := NewTLSServer()
	defer close()
	srv.SetResponse(
		WithStatusCode(200))

	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}

	_, err := client.createRequest(context.Background(), http.MethodGet, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateRequest(t *testing.T) {
	srv, close := NewTLSServer()
	defer close()
	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}

	req, err := client.createRequest(context.Background(), http.MethodGet, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if req.Raw().URL.String() != srv.URL() {
		t.Errorf("Expected %v, but got %v", srv.URL(), req.Raw().URL.String())
	}

	if req.Raw().Method != http.MethodGet {
		t.Errorf("Expected %v, but got %v", http.MethodGet, req.Raw().Method)
	}
}

func TestSendPost(t *testing.T) {
	srv, close := NewTLSServer()
	defer close()
	srv.SetResponse(
		WithStatusCode(200))
	verifier := pipelineVerifier{}
	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{PerCall: []policy.Policy{&verifier}}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}

	_, err := client.sendPostRequest(context.Background(), &GetDetectorOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if verifier.requests[0].method != http.MethodPost {
		t.Errorf("Expected %v, but got %v", http.MethodPost, verifier.requests[0].method)
	}
}

func TestGetDetector(t *testing.T) {
	srv, close := NewTLSServer()
	defer close()
	srv.SetResponse(
		WithStatusCode(200))
	verifier := pipelineVerifier{}
	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{PerCall: []policy.Policy{&verifier}}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}

	testResourceId := "testResourceId"
	testDetectorID := "testDetector"
	_, err := client.sendPostRequest(context.Background(),
		&GetDetectorOptions{
			ResourceID: testResourceId,
			DetectorID: testDetectorID,
		}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if verifier.requests[0].method != http.MethodPost {
		t.Errorf("Expected %v, but got %v", http.MethodPost, verifier.requests[0].method)
	}

	if verifier.requests[0].headers.Get(headerXmsDate) == "" {
		t.Errorf("Expected %v, but got %v", "", verifier.requests[0].headers.Get(headerXmsDate))
	}

	if verifier.requests[0].headers.Get(headerXmsClientRequestId) == "" {
		t.Errorf("Expected uuid in %v header field, but got empty string", headerXmsClientRequestId)
	}

	if verifier.requests[0].headers.Get(headerXmsRequestId) == "" {
		t.Errorf("Expected uuid in %v header field, but got empty string", headerXmsRequestId)
	}

	if verifier.requests[0].headers.Get(headerXmsPathQuery) != fmt.Sprintf("%s/detectors/%s", testResourceId, testDetectorID) {
		t.Errorf("Expected %v in %v header field, but got %v", fmt.Sprintf("%s/detectors/%s", testResourceId, testDetectorID), headerXmsPathQuery, verifier.requests[0].headers.Get(headerXmsPathQuery))
	}
}

func TestListDetectors(t *testing.T) {
	srv, close := NewTLSServer()
	defer close()
	srv.SetResponse(
		WithStatusCode(200))
	verifier := pipelineVerifier{}
	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{PerCall: []policy.Policy{&verifier}}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}

	testResourceId := "testResourceId"
	_, err := client.sendPostRequest(context.Background(),
		&ListDetectorsOptions{
			ResourceID: testResourceId,
		}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if verifier.requests[0].method != http.MethodPost {
		t.Errorf("Expected %v, but got %v", http.MethodPost, verifier.requests[0].method)
	}

	if verifier.requests[0].headers.Get(headerXmsDate) == "" {
		t.Errorf("Expected %v, but got %v", "", verifier.requests[0].headers.Get(headerXmsDate))
	}

	if verifier.requests[0].headers.Get(headerXmsClientRequestId) == "" {
		t.Errorf("Expected uuid in %v header field, but got empty string", headerXmsClientRequestId)
	}

	if verifier.requests[0].headers.Get(headerXmsRequestId) == "" {
		t.Errorf("Expected uuid in %v header field, but got empty string", headerXmsRequestId)
	}

	if verifier.requests[0].headers.Get(headerXmsPathQuery) != fmt.Sprintf("%s/detectors", testResourceId) {
		t.Errorf("Expected %v in %v header field, but got %v", fmt.Sprintf("%s/detectors", testResourceId), headerXmsPathQuery, verifier.requests[0].headers.Get(headerXmsPathQuery))
	}
}

type pipelineVerifier struct {
	requests []pipelineVerifierRequest
}

type pipelineVerifierRequest struct {
	method  string
	body    string
	url     *url.URL
	headers http.Header
}

func (p *pipelineVerifier) Do(req *policy.Request) (*http.Response, error) {
	pr := pipelineVerifierRequest{}
	pr.method = req.Raw().Method
	pr.url = req.Raw().URL
	if req.Body() != nil {
		readBody, _ := ioutil.ReadAll(req.Body())
		pr.body = string(readBody)
	}
	pr.headers = req.Raw().Header
	p.requests = append(p.requests, pr)
	return req.Next()
}
