package serviceprincipalchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/api/validate/dynamic"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
)

type servicePrincipalChecker interface {
	Check(ctx context.Context, AZEnvironment string) error
}

type checker struct {
	log *logrus.Entry

	credentials        func(ctx context.Context) (*clusterauthorizer.Credentials, error)
	getTokenCredential func(azEnv *azureclient.AROEnvironment, credentials *clusterauthorizer.Credentials) (azcore.TokenCredential, error)
	newSPValidator     func(azEnv *azureclient.AROEnvironment) dynamic.ServicePrincipalValidator
}

func newServicePrincipalChecker(log *logrus.Entry, client client.Client) *checker {
	return &checker{
		log: log,

		credentials: func(ctx context.Context) (*clusterauthorizer.Credentials, error) {
			return clusterauthorizer.AzCredentials(ctx, client)
		},
		getTokenCredential: clusterauthorizer.GetTokenCredential,
		newSPValidator: func(azEnv *azureclient.AROEnvironment) dynamic.ServicePrincipalValidator {
			return dynamic.NewServicePrincipalValidator(log, azEnv, dynamic.AuthorizerClusterServicePrincipal)
		},
	}
}

func (r *checker) Check(ctx context.Context, AZEnvironment string) error {
	azEnv, err := azureclient.EnvironmentFromName(AZEnvironment)
	if err != nil {
		return err
	}

	azCred, err := r.credentials(ctx)
	if err != nil {
		return err
	}

	spDynamic := r.newSPValidator(&azEnv)

	tokenCredential, err := r.getTokenCredential(&azEnv, azCred)
	if err != nil {
		return err
	}

	return spDynamic.ValidateServicePrincipal(ctx, tokenCredential)
}
