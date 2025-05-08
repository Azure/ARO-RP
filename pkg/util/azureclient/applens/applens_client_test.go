package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	testhttp "github.com/Azure/ARO-RP/test/util/http/server"
)

func TestEnsureErrorIsGeneratedOnResponse(t *testing.T) {
	someError := &appLensErrorResponse{
		Code: "SomeCode",
	}

	jsonString, err := json.Marshal(someError)
	if err != nil {
		t.Fatal(err)
	}

	srv, close := testhttp.NewTLSServer()
	defer close()
	srv.SetResponse(
		testhttp.WithBody(jsonString),
		testhttp.WithStatusCode(404))

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
	srv, close := testhttp.NewTLSServer()
	defer close()
	srv.SetResponse(
		testhttp.WithStatusCode(200))

	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}
	_, err := client.sendPostRequest(context.Background(), &GetDetectorOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRequestEnricherIsCalled(t *testing.T) {
	srv, close := testhttp.NewTLSServer()
	defer close()
	srv.SetResponse(
		testhttp.WithStatusCode(200))

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
	srv, close := testhttp.NewTLSServer()
	defer close()
	srv.SetResponse(
		testhttp.WithStatusCode(200))

	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}

	_, err := client.createRequest(context.Background(), http.MethodGet, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateRequest(t *testing.T) {
	srv, close := testhttp.NewTLSServer()
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
	srv, close := testhttp.NewTLSServer()
	defer close()
	srv.SetResponse(
		testhttp.WithStatusCode(200))
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
	srv, close := testhttp.NewTLSServer()
	defer close()
	srv.SetResponse(
		testhttp.WithStatusCode(200))
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
}

func TestListDetectorsDirect(t *testing.T) {
	srv, close := testhttp.NewTLSServer()
	defer close()
	srv.SetResponse(testhttp.WithStatusCode(200))
	verifier := pipelineVerifier{}
	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{PerCall: []policy.Policy{&verifier}}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}

	testResourceId := "testResourceId"
	testLocation := "eastus"
	_, err := client.sendPostRequest(context.Background(),
		&ListDetectorsOptions{
			ResourceID: testResourceId,
			Location:   testLocation,
		}, nil)

	if err != nil {
		t.Fatal(err)
	}

	if verifier.requests[0].method != http.MethodPost {
		t.Errorf("Expected %v, but got %v", http.MethodPost, verifier.requests[0].method)
	}
}

func TestListDetectors(t *testing.T) {
	srv, close := testhttp.NewTLSServer()
	defer close()
	testResourceId := "testResourceId"
	testLocation := "eastus"
	testDetectorName := "aroauthhealth"
	testBody := `[{
		"appFilter": null,
		"dataProvidersMetadata": null,
		"dataset": [],
		"metadata": {
			"analysisType": "arooperatorinsights,aroclusterinsights",
			"analysisTypes": [
				"arooperatorinsights",
				"aroclusterinsights"
			],
			"author": "",
			"category": "Operator Health",
			"description": "Reports if the ARO Auth Operator becomes unhealthy.",
			"id": "aroauthhealth",
			"name": "Authentication Operator",
			"score": 0,
			"supportTopicList": [],
			"type": "Detector",
			"typeId": "6820fea2-a74f-4059-b7ba-688cc943d2d8"
		},
		"status": {
			"message": null,
			"statusId": 4
		},
		"suggestedUtterances": null
	}]`

	srv.SetResponse(testhttp.WithBody([]byte(testBody)))
	verifier := pipelineVerifier{}
	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{PerCall: []policy.Policy{&verifier}}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}

	detectors, err := client.ListDetectors(context.Background(),
		&ListDetectorsOptions{
			ResourceID: testResourceId,
			Location:   testLocation,
		})

	if err != nil {
		t.Fatal(err)
	}

	if verifier.requests[0].method != http.MethodPost {
		t.Errorf("Expected %v, but got %v", http.MethodPost, verifier.requests[0].method)
	}

	if len(detectors.Value) != 1 {
		t.Error("Expected count of detectors equal 1")
	}

	if detectors.Value[0].Id != fmt.Sprintf("%s/detectors/aroauthhealth", testResourceId) {
		t.Error("Expected detector Id does not match")
	}

	if detectors.Value[0].Name != testDetectorName {
		t.Error("Expected detector Name does not match")
	}

	if detectors.Value[0].Location != testLocation {
		t.Error("Expected detector Name does not match")
	}

	if detectors.Value[0].Type != "Microsoft.RedHatOpenShift/openShiftClusters/detectors" {
		t.Error("Expected detector Name does not match")
	}

	if detectors.Value[0].Properties.(map[string]interface{})["metadata"].(map[string]interface{})["id"].(string) != testDetectorName {
		t.Error("Expected detector properties does not match")
	}
}

func TestListDetectorAroAuthHealth(t *testing.T) {
	srv, close := testhttp.NewTLSServer()
	defer close()
	testResourceId := "testResourceId"
	testLocation := "eastus"
	testDetectorName := "aroclusterinsights"
	testBody := `{
		"appFilter": null,
		"dataProvidersMetadata": null,
		"dataset": [],
		"metadata": {
		  "analysisType": "",
		  "analysisTypes": null,
		  "author": "",
		  "category": "Insights",
		  "description": "Identifies scenarios that may cause a cluster to no longer be manageable.",
		  "id": "aroclusterinsights",
		  "name": "Cluster Insights",
		  "score": 0,
		  "supportTopicList": [],
		  "type": "Analysis",
		  "typeId": "a881d7f8-6385-4f33-9f43-063744b61452"
		},
		"status": {
		  "message": null,
		  "statusId": 4
		},
		"suggestedUtterances": null
	  }`

	srv.SetResponse(testhttp.WithBody([]byte(testBody)))
	verifier := pipelineVerifier{}
	pl := runtime.NewPipeline("applenstest", "v1.0.0", runtime.PipelineOptions{PerCall: []policy.Policy{&verifier}}, &policy.ClientOptions{Transport: srv})
	client := &Client{endpoint: srv.URL(), pipeline: pl}

	detector, err := client.GetDetector(context.Background(),
		&GetDetectorOptions{
			ResourceID: testResourceId,
			Location:   testLocation,
			DetectorID: testDetectorName,
		})

	if err != nil {
		t.Fatal(err)
	}

	if detector.Id != fmt.Sprintf("%s/detectors/%s", testResourceId, testDetectorName) {
		t.Error("Expected detector Id does not match")
	}

	if detector.Name != testDetectorName {
		t.Error("Expected name of detector does not match")
	}

	if detector.Type != "Microsoft.RedHatOpenShift/openShiftClusters/detectors" {
		t.Error("Expected type of detector does not match")
	}

	if detector.Location != testLocation {
		t.Error("Expected type of detector does not match")
	}

	if detector.Properties.(map[string]interface{})["metadata"].(map[string]interface{})["id"].(string) != testDetectorName {
		t.Error("Expected detector properties does not match")
	}
}

type pipelineVerifier struct {
	requests []pipelineVerifierRequest
}

type pipelineVerifierRequest struct {
	method string
	body   string
	url    *url.URL
}

func (p *pipelineVerifier) Do(req *policy.Request) (*http.Response, error) {
	pr := pipelineVerifierRequest{}
	pr.method = req.Raw().Method
	pr.url = req.Raw().URL
	if req.Body() != nil {
		readBody, _ := io.ReadAll(req.Body())
		pr.body = string(readBody)
	}
	p.requests = append(p.requests, pr)
	return req.Next()
}
