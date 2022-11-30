package clusterauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/form3tech-oss/jwt-go"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
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
	kubernetescli    kubernetes.Interface
	tokenClient      aad.TokenClient
}

// NewAzRefreshableAuthorizer returns a new refreshable authorizer
// using Cluster Service Principal.
func NewAzRefreshableAuthorizer(log *logrus.Entry, azEnv *azureclient.AROEnvironment, kubernetescli kubernetes.Interface, tokenClient aad.TokenClient) (*azRefreshableAuthorizer, error) {
	if log == nil {
		return nil, fmt.Errorf("log entry cannot be nil")
	}
	if azEnv == nil {
		return nil, fmt.Errorf("azureEnvironment cannot be nil")
	}
	return &azRefreshableAuthorizer{
		log:              log,
		azureEnvironment: azEnv,
		kubernetescli:    kubernetescli,
		tokenClient:      tokenClient,
	}, nil
}

func (a *azRefreshableAuthorizer) NewRefreshableAuthorizerToken(ctx context.Context) (refreshable.Authorizer, error) {
	// Grab azure-credentials from secret
	credentials, err := AzCredentials(ctx, a.kubernetescli)
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
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalToken, "properties.servicePrincipalProfile", "The provided service principal generated an invalid token.")
	}

	return refreshable.NewAuthorizer(token), nil
}

// AzCredentials gets Cluster Service Principal credentials from the Kubernetes secrets
func AzCredentials(ctx context.Context, kubernetescli kubernetes.Interface) (*Credentials, error) {
	mysec, err := kubernetescli.CoreV1().Secrets(azureCredentialSecretNameSpace).Get(ctx, azureCredentialSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	for _, key := range []string{"azure_client_id", "azure_client_secret", "azure_tenant_id"} {
		if _, ok := mysec.Data[key]; !ok {
			return nil, fmt.Errorf("%s does not exist in the secret", key)
		}
	}

	return &Credentials{
		ClientID:     mysec.Data["azure_client_id"],
		ClientSecret: mysec.Data["azure_client_secret"],
		TenantID:     mysec.Data["azure_tenant_id"],
	}, nil
}
