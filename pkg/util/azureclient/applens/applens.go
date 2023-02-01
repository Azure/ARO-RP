package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// AppLensClient is a minimal interface for azure AppLensClient
type AppLensClient interface {
	GetDetector(ctx context.Context, o *GetDetectorOptions) (*http.Response, error)
	ListDetectors(ctx context.Context, o *ListDetectorsOptions) (*http.Response, error)
}

type appLensClient struct {
	*Client
}

var _ AppLensClient = &appLensClient{}

// NewAppLensClient returns a new AppLensClient
func NewAppLensClient(env *azureclient.AROEnvironment, cred azcore.TokenCredential) AppLensClient {
	client, _ := NewClient(env.AppLensEndpoint, env.AppLensScope, cred, nil)
	return &appLensClient{
		Client: client,
	}
}
