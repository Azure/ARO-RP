package clusterauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/form3tech-oss/jwt-go"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

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

// NewAzRefreshableAuthorizer returns a new refreshable authorizer
// using Cluster Service Principal.
func NewAzRefreshableAuthorizer(ctx context.Context, log *logrus.Entry, azEnv *azureclient.AROEnvironment, kubernetescli kubernetes.Interface) (refreshable.Authorizer, error) {
	// Grab azure-credentials from secret
	credentials, err := AzCredentials(ctx, kubernetescli)
	if err != nil {
		return nil, err
	}
	// create service principal token from azure-credentials
	token, err := aad.GetToken(ctx, log, string(credentials.ClientID), string(credentials.ClientSecret), string(credentials.TenantID), azEnv.ActiveDirectoryEndpoint, azEnv.ResourceManagerEndpoint)
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
