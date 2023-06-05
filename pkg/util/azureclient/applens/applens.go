package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// AppLensClient is a minimal interface for azure AppLensClient
type AppLensClient interface {
	GetDetector(ctx context.Context, o *GetDetectorOptions) (*ResponseMessageEnvelope, error)
	ListDetectors(ctx context.Context, o *ListDetectorsOptions) (*ResponseMessageCollectionEnvelope, error)
}

type appLensClient struct {
	*Client
}

var _ AppLensClient = &appLensClient{}

// NewAppLensClient returns a new AppLensClient
func NewAppLensClient(env *azureclient.AROEnvironment, cred azcore.TokenCredential) (AppLensClient, error) {
	client, err := NewClient(env.AppLensEndpoint, env.PkiIssuerUrlTemplate, env.PkiCaName, env.AppLensScope, cred, nil)

	if err != nil {
		return nil, err
	}

	return &appLensClient{
		Client: client,
	}, nil
}
