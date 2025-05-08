package serviceprincipalchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/sirupsen/logrus"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
	"github.com/Azure/ARO-RP/pkg/validate/dynamic"
)

type servicePrincipalChecker interface {
	Check(ctx context.Context, AZEnvironment string) error
}

type checker struct {
	log *logrus.Entry

	getTokenCredential func(azEnv *azureclient.AROEnvironment) (azcore.TokenCredential, error)
	newSPValidator     func(azEnv *azureclient.AROEnvironment) dynamic.ServicePrincipalValidator
}

func newServicePrincipalChecker(log *logrus.Entry, client client.Client) *checker {
	return &checker{
		log: log,

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

	spDynamic := r.newSPValidator(&azEnv)

	spTokenCredential, err := r.getTokenCredential(&azEnv)
	if err != nil {
		return err
	}

	return spDynamic.ValidateServicePrincipal(ctx, spTokenCredential)
}
