package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
// AppLens Client created from CosmosDB Client
// (https://github.com/Azure/azure-sdk-for-go/blob/3f7acd20691214ef2cb1f0132f82115f1df01a8c/sdk/data/azcosmos/cosmos_client.go)

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

// AppLens client is used to interact with the Azure AppLens service.
type Client struct {
	endpoint string
	pipeline runtime.Pipeline
}

// Endpoint used to create the client.
func (c *Client) Endpoint() string {
	return c.endpoint
}

// NewClient creates a new instance of AppLens client with Azure AD access token authentication. It uses the default pipeline configuration.
// endpoint - The applens service endpoint to use.
// cred - The credential used to authenticate with the applens service.
// options - Optional AppLens client options.  Pass nil to accept default values.
func NewClient(endpoint, scope string, cred azcore.TokenCredential, o *ClientOptions) (*Client, error) {
	return &Client{endpoint: endpoint, pipeline: newPipeline([]policy.Policy{runtime.NewBearerTokenPolicy(cred, []string{fmt.Sprintf("%s/.default", scope)}, nil)}, o)}, nil
}

func newPipeline(authPolicy []policy.Policy, options *ClientOptions) runtime.Pipeline {
	if options == nil {
		options = NewClientOptions()
	}

	return runtime.NewPipeline("applens", serviceLibVersion,
		runtime.PipelineOptions{
			PerCall:  []policy.Policy{},
			PerRetry: authPolicy,
		},
		&options.ClientOptions)
}

// ListDetectors obtains the list of detectors for a service from AppLens.
// ctx - The context for the request.
// o - Options for Read operation.
func (c *Client) ListDetectors(
	ctx context.Context,
	o *ListDetectorsOptions) (*http.Response, error) {
	if o == nil {
		o = &ListDetectorsOptions{}
	}

	azResponse, err := c.sendPostRequest(
		ctx,
		o,
		nil)
	if err != nil {
		return nil, err
	}

	return azResponse, nil
}

// GetDetector obtains detector information from AppLens.
// ctx - The context for the request.
// o - Options for Read operation.
func (c *Client) GetDetector(
	ctx context.Context,
	o *GetDetectorOptions) (*http.Response, error) {
	if o == nil {
		o = &GetDetectorOptions{}
	}

	azResponse, err := c.sendPostRequest(
		ctx,
		o,
		nil)
	if err != nil {
		return nil, err
	}

	return azResponse, nil
}

func (c *Client) sendPostRequest(
	ctx context.Context,
	requestOptions appLensRequestOptions,
	requestEnricher func(*policy.Request)) (*http.Response, error) {
	req, err := c.createRequest(ctx, http.MethodPost, requestOptions, requestEnricher)
	if err != nil {
		return nil, err
	}

	return c.executeAndEnsureSuccessResponse(req)
}

func (c *Client) createRequest(
	ctx context.Context,
	method string,
	requestOptions appLensRequestOptions,
	requestEnricher func(*policy.Request)) (*policy.Request, error) {
	if requestOptions != nil {
		header := requestOptions.toHeader()
		ctx = policy.WithHTTPHeader(ctx, header)
	}

	req, err := runtime.NewRequest(ctx, method, c.endpoint)
	if err != nil {
		return nil, err
	}

	if requestEnricher != nil {
		requestEnricher(req)
	}

	return req, nil
}

func (c *Client) executeAndEnsureSuccessResponse(request *policy.Request) (*http.Response, error) {
	response, err := c.pipeline.Do(request)
	if err != nil {
		return nil, err
	}

	successResponse := (response.StatusCode >= 200 && response.StatusCode < 300) || response.StatusCode == 304
	if successResponse {
		return response, nil
	}

	return nil, newAppLensError(response)
}
