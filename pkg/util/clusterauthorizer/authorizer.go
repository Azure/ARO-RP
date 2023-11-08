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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
	credential, err := azidentity.NewDefaultAzureCredential(environment.DefaultAzureCredentialOptions())

	if err != nil {
		return nil, err
	}

	return credential, nil
}

// AzCredentials gets Cluster Service Principal credentials from the Kubernetes Secret. It returns nil if the Secret is not found.
func AzCredentials(ctx context.Context, kubernetescli kubernetes.Interface) (Credentials, error) {
	clusterSPSecret, err := kubernetescli.CoreV1().Secrets(AzureCredentialSecretNameSpace).Get(ctx, AzureCredentialSecretName, metav1.GetOptions{})

	if kerrors.IsNotFound(err) {
		return Credentials{}, nil
	} else if err != nil {
		return Credentials{}, err
	}

	for _, key := range []string{"azure_client_id", "azure_client_secret", "azure_tenant_id"} {
		if _, ok := clusterSPSecret.Data[key]; !ok {
			return Credentials{}, fmt.Errorf("%s does not exist in the secret", key)
		}
	}

	return Credentials{
		ClientID:     clusterSPSecret.Data["azure_client_id"],
		ClientSecret: clusterSPSecret.Data["azure_client_secret"],
		TenantID:     clusterSPSecret.Data["azure_tenant_id"],
	}, nil
}
