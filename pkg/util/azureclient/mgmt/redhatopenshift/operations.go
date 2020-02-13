package redhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/client/services/preview/redhatopenshift/mgmt/2019-12-31-preview/redhatopenshift"
)

// OperationsClient is a minimal interface for azure OperationsClient
type OperationsClient interface {
	List(ctx context.Context) (result redhatopenshift.OperationList, err error)
}

type operationsClient struct {
	redhatopenshift.OperationsClient
}

var _ OperationsClient = &operationsClient{}

// NewOperationsClient creates a new OperationsClient
func NewOperationsClient(subscriptionID string, authorizer autorest.Authorizer) OperationsClient {
	var client redhatopenshift.OperationsClient
	if os.Getenv("RP_MODE") == "development" {
		client = redhatopenshift.NewOperationsClientWithBaseURI("https://localhost:8443", subscriptionID)
		client.Sender = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	} else {
		client = redhatopenshift.NewOperationsClient(subscriptionID)
		client.Authorizer = authorizer
	}

	return &operationsClient{
		OperationsClient: client,
	}
}
