package clusterauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	getTokenCredential func(*azureclient.AROEnvironment, *Credentials) (azcore.TokenCredential, error)
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
	// Grab azure-credentials from secret
	credentials, err := AzCredentials(ctx, a.client)
	if err != nil {
		return nil, err
	}

	// Create service principal token from azure-credentials
	tokenCredential, err := a.getTokenCredential(a.azureEnvironment, credentials)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "the provided service principal is invalid")
	}

	scopes := []string{a.azureEnvironment.ResourceManagerScope}

	return azidext.NewTokenCredentialAdapter(tokenCredential, scopes), nil
}

func GetTokenCredential(environment *azureclient.AROEnvironment, credentials *Credentials) (azcore.TokenCredential, error) {
	return azidentity.NewClientSecretCredential(
		string(credentials.TenantID),
		string(credentials.ClientID),
		string(credentials.ClientSecret),
		environment.ClientSecretCredentialOptions())
}

// AzCredentials gets Cluster Service Principal credentials from the Kubernetes secrets
func AzCredentials(ctx context.Context, client client.Client) (*Credentials, error) {
	clusterSPSecret := &corev1.Secret{}
	err := client.Get(ctx, types.NamespacedName{Namespace: AzureCredentialSecretNameSpace, Name: AzureCredentialSecretName}, clusterSPSecret)
	if err != nil {
		return nil, err
	}

	for _, key := range []string{"azure_client_id", "azure_client_secret", "azure_tenant_id"} {
		if _, ok := clusterSPSecret.Data[key]; !ok {
			return nil, fmt.Errorf("%s does not exist in the secret", key)
		}
	}

	return &Credentials{
		ClientID:     clusterSPSecret.Data["azure_client_id"],
		ClientSecret: clusterSPSecret.Data["azure_client_secret"],
		TenantID:     clusterSPSecret.Data["azure_tenant_id"],
	}, nil
}
