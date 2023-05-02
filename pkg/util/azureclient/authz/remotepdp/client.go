package remotepdp

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// this asserts that &remotePDPClient{} would always implement RemotePDPClient
var _ RemotePDPClient = &remotePDPClient{}

// RemotePDPClient represents the Microsoft Remote PDP API Spec
type RemotePDPClient interface {
	CheckAccess(context.Context, AuthorizationRequest) (*AuthorizationDecisionResponse, error)
}

// remotePDPClient implements RemotePDPClient
type remotePDPClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

// NewRemotePDPClient returns an implementation of RemotePDPClient
// endpoint - the fqdn of the regional specific endpoint of PDP
// scope - the oauth scope required by the PDP server
// cred - the credential of the client to call the PDP server
func NewRemotePDPClient(endpoint, scope string, cred azcore.TokenCredential) *remotePDPClient {
	authPolicy := runtime.NewBearerTokenPolicy(cred, []string{scope}, nil)

	customRoundTripper := azureclient.NewCustomRoundTripper(http.DefaultTransport)
	clientOptions := &azcore.ClientOptions{
		Transport: &http.Client{
			Transport: customRoundTripper,
		},
	}

	pipeline := runtime.NewPipeline(
		modulename,
		version,
		runtime.PipelineOptions{
			PerCall:  []policy.Policy{},
			PerRetry: []policy.Policy{authPolicy},
		},
		clientOptions,
	)

	return &remotePDPClient{endpoint, pipeline}
}

// CheckAccess sends an Authorization query to the PDP server specified in the client
// ctx - the context to propagate
// authzReq - the actual AuthorizationRequest
func (r *remotePDPClient) CheckAccess(ctx context.Context, authzReq AuthorizationRequest) (*AuthorizationDecisionResponse, error) {
	req, err := runtime.NewRequest(ctx, http.MethodPost, r.endpoint)
	if err != nil {
		return nil, err
	}
	runtime.MarshalAsJSON(req, authzReq)

	res, err := r.pipeline.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, newCheckAccessError(res)
	}

	var accessDecision AuthorizationDecisionResponse
	if err := runtime.UnmarshalAsJSON(res, &accessDecision); err != nil {
		return nil, err
	}

	return &accessDecision, nil
}

// newCheckAccessError returns an error when non HTTP 200 response is returned.
func newCheckAccessError(r *http.Response) error {
	payload, err := runtime.Payload(r)
	if err != nil {
		return err
	}
	var checkAccessError CheckAccessErrorResponse
	err = json.Unmarshal(payload, &checkAccessError)
	if err != nil {
		return err
	}
	return &azcore.ResponseError{
		StatusCode:  r.StatusCode,
		RawResponse: r,
		ErrorCode:   fmt.Sprint(checkAccessError.StatusCode),
	}
}
