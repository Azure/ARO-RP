package redhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	gofrsuuid "github.com/gofrs/uuid"

	mgmtredhatopenshift20240812preview "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2024-08-12-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// OperationsClient is a minimal interface for azure OperationsClient
type OperationsClient interface {
	OperationsClientAddons
}

type operationsClient struct {
	mgmtredhatopenshift20240812preview.OperationsClient
}

var _ OperationsClient = &operationsClient{}

// NewOperationsClient creates a new OperationsClient
func NewOperationsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) OperationsClient {
	var client mgmtredhatopenshift20240812preview.OperationsClient
	if env.IsLocalDevelopmentMode() {
		client = mgmtredhatopenshift20240812preview.NewOperationsClientWithBaseURI("https://localhost:8443", gofrsuuid.FromStringOrNil(subscriptionID))
		client.Sender = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // #nosec G402
				},
			},
		}
	} else {
		client = mgmtredhatopenshift20240812preview.NewOperationsClientWithBaseURI(environment.ResourceManagerEndpoint, gofrsuuid.FromStringOrNil(subscriptionID))
		client.Authorizer = authorizer
	}

	return &operationsClient{
		OperationsClient: client,
	}
}
