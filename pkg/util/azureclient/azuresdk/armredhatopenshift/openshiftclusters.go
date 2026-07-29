package armredhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	armredhatopenshift "github.com/Azure/ARO-RP/pkg/client/sdk/resourcemanager/redhatopenshift/armredhatopenshift"
	"github.com/Azure/ARO-RP/pkg/env"
)

type OpenShiftClustersClient interface {
	ListAdminCredentials(ctx context.Context, resourceGroupName string, resourceName string, options *armredhatopenshift.OpenShiftClustersClientListAdminCredentialsOptions) (armredhatopenshift.OpenShiftClustersClientListAdminCredentialsResponse, error)
	ListCredentials(ctx context.Context, resourceGroupName string, resourceName string, options *armredhatopenshift.OpenShiftClustersClientListCredentialsOptions) (armredhatopenshift.OpenShiftClustersClientListCredentialsResponse, error)
	Get(ctx context.Context, resourceGroupName string, resourceName string, options *armredhatopenshift.OpenShiftClustersClientGetOptions) (armredhatopenshift.OpenShiftClustersClientGetResponse, error)
	OpenShiftClustersAddons
}

type openShiftClustersClient struct {
	*armredhatopenshift.OpenShiftClustersClient
}

var _ OpenShiftClustersClient = &openShiftClustersClient{}

func NewOpenShiftClustersClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (OpenShiftClustersClient, error) {
	options = configureDevMode(options)

	client, err := armredhatopenshift.NewOpenShiftClustersClient(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}

	return &openShiftClustersClient{
		OpenShiftClustersClient: client,
	}, nil
}

func configureDevMode(options *arm.ClientOptions) *arm.ClientOptions {
	if !env.IsLocalDevelopmentMode() {
		return options
	}

	if options == nil {
		options = &arm.ClientOptions{}
	} else {
		opts := *options
		options = &opts
	}

	endpointOverridePolicy := policyFunc(func(req *policy.Request) (*http.Response, error) {
		httpReq := req.Raw()
		httpReq.URL.Scheme = "https"
		httpReq.URL.Host = "localhost:8443"
		return req.Next()
	})

	options.ClientOptions.Transport = &insecureTransport{}
	options.ClientOptions.PerCallPolicies = append([]policy.Policy{endpointOverridePolicy}, options.ClientOptions.PerCallPolicies...)

	return options
}

type policyFunc func(req *policy.Request) (*http.Response, error)

func (p policyFunc) Do(req *policy.Request) (*http.Response, error) {
	return p(req)
}

var _ policy.Policy = policyFunc(nil)

type insecureTransport struct{}

func (t *insecureTransport) Do(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // #nosec G402 // CodeQL [SM03511] only used in local development
			},
		},
	}
	return client.Do(req)
}
