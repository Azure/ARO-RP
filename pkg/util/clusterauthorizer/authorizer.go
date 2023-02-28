package clusterauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/form3tech-oss/jwt-go"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclaim"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
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
	tokenClient      aad.TokenClient
}

// NewAzRefreshableAuthorizer returns a new refreshable authorizer
// using Cluster Service Principal.
func NewAzRefreshableAuthorizer(log *logrus.Entry, azEnv *azureclient.AROEnvironment, client client.Client, tokenClient aad.TokenClient) (*azRefreshableAuthorizer, error) {
	if log == nil {
		return nil, fmt.Errorf("log entry cannot be nil")
	}
	if azEnv == nil {
		return nil, fmt.Errorf("azureEnvironment cannot be nil")
	}
	return &azRefreshableAuthorizer{
		log:              log,
		azureEnvironment: azEnv,
		client:           client,
		tokenClient:      tokenClient,
	}, nil
}

func (a *azRefreshableAuthorizer) NewRefreshableAuthorizerToken(ctx context.Context) (refreshable.Authorizer, error) {
	// Grab azure-credentials from secret
	credentials, err := AzCredentials(ctx, a.client)
	if err != nil {
		return nil, err
	}
	// create service principal token from azure-credentials
	token, err := a.tokenClient.GetToken(ctx,
		a.log,
		string(credentials.ClientID),
		string(credentials.ClientSecret),
		string(credentials.TenantID),
		a.azureEnvironment.ActiveDirectoryEndpoint,
		a.azureEnvironment.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	p := &jwt.Parser{}
	c := &azureclaim.AzureClaim{}
	_, _, err = p.ParseUnverified(token.OAuthToken(), c)
	if err != nil {
		return nil, err
	}

	return refreshable.NewAuthorizer(token), nil
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
