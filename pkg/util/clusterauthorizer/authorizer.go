package clusterauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type Credentials struct {
	ClientID     []byte
	ClientSecret []byte
	TenantID     []byte
}

type azRefreshableAuthorizer struct {
	log              *logrus.Entry
	azureEnvironment *azureclient.AROEnvironment
	client           client.Client

	getTokenCredential func(*azureclient.AROEnvironment) (azcore.TokenCredential, error)
}

// NewAzRefreshableAuthorizer returns a new refreshable authorizer
// using Cluster Service Principal.
func NewAzRefreshableAuthorizer(log *logrus.Entry, azEnv *azureclient.AROEnvironment, client client.Client) (*azRefreshableAuthorizer, error) {
	if log == nil {
		return nil, fmt.Errorf("log entry cannot be nil")
	}
	if azEnv == nil {
		return nil, fmt.Errorf("azureEnvironment cannot be nil")
	}
	return &azRefreshableAuthorizer{
		log:                log,
		azureEnvironment:   azEnv,
		client:             client,
		getTokenCredential: GetTokenCredential,
	}, nil
}

func (a *azRefreshableAuthorizer) NewRefreshableAuthorizerToken(ctx context.Context) (autorest.Authorizer, error) {
	tokenCredential, err := a.getTokenCredential(a.azureEnvironment)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "the provided service principal is invalid")
	}

	scopes := []string{a.azureEnvironment.ResourceManagerScope}

	return azidext.NewTokenCredentialAdapter(tokenCredential, scopes), nil
}

func GetTokenCredential(environment *azureclient.AROEnvironment) (azcore.TokenCredential, error) {
	return azidentity.NewDefaultAzureCredential(environment.DefaultAzureCredentialOptions())
}
