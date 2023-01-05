package serviceprincipalchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api/validate/dynamic"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
)

type servicePrincipalChecker interface {
	Check(ctx context.Context, AZEnvironment string) error
}

type credentialsGetter interface {
	Get(ctx context.Context, azEnv *azureclient.AROEnvironment) (token *adal.ServicePrincipalToken, err error)
}

type credGetter struct {
	kubernetescli kubernetes.Interface
	log           *logrus.Entry
}

func (g credGetter) Get(ctx context.Context, azEnv *azureclient.AROEnvironment) (token *adal.ServicePrincipalToken, err error) {
	tokenClient := aad.NewTokenClient()

	azCred, err := clusterauthorizer.AzCredentials(ctx, g.kubernetescli)
	if err != nil {
		return nil, err
	}

	token, err = tokenClient.GetToken(ctx, g.log, string(azCred.ClientID), string(azCred.ClientSecret), string(azCred.TenantID), azEnv.ActiveDirectoryEndpoint, azEnv.ResourceManagerEndpoint)
	return token, err
}

type checker struct {
	log *logrus.Entry

	credentials func(ctx context.Context) (*clusterauthorizer.Credentials, error)
	credGetter  credentialsGetter
	spValidator dynamic.ServicePrincipalValidator
	tokenClient aad.TokenClient
}

func newServicePrincipalChecker(log *logrus.Entry, kubernetescli kubernetes.Interface, spValidator dynamic.ServicePrincipalValidator) *checker {
	return &checker{
		log: log,

		credentials: func(ctx context.Context) (*clusterauthorizer.Credentials, error) {
			return clusterauthorizer.AzCredentials(ctx, kubernetescli)
		},
		spValidator: spValidator,
		tokenClient: aad.NewTokenClient(),
		credGetter:  credGetter{kubernetescli: kubernetescli, log: log},
	}
}

func (r *checker) Check(ctx context.Context, AZEnvironment string) error {
	azEnv, err := azureclient.EnvironmentFromName(AZEnvironment)
	if err != nil {
		return err
	}

	token, err := r.credGetter.Get(ctx, &azEnv)
	if err != nil {
		return err
	}
	return r.spValidator.Validate(token)

	// return spDynamic.ValidateServicePrincipal(ctx, string(azCred.ClientID), string(azCred.ClientSecret), string(azCred.TenantID))
}
