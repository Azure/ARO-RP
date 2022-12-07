package serviceprincipalchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

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

type checker struct {
	log *logrus.Entry

	credentialsGetter      func(ctx context.Context) (*clusterauthorizer.Credentials, error)
	spValidatorConstructor func(azEnv *azureclient.AROEnvironment) (dynamic.ServicePrincipalValidator, error)
}

func newServicePrincipalChecker(log *logrus.Entry, kubernetescli kubernetes.Interface) servicePrincipalChecker {
	tokenClient := aad.NewTokenClient()

	return &checker{
		log: log,

		credentialsGetter: func(ctx context.Context) (*clusterauthorizer.Credentials, error) {
			return clusterauthorizer.AzCredentials(ctx, kubernetescli)
		},
		spValidatorConstructor: func(azEnv *azureclient.AROEnvironment) (dynamic.ServicePrincipalValidator, error) {
			return dynamic.NewServicePrincipalValidator(log, azEnv, dynamic.AuthorizerClusterServicePrincipal, tokenClient)
		},
	}
}

func (r *checker) Check(ctx context.Context, AZEnvironment string) error {
	azEnv, err := azureclient.EnvironmentFromName(AZEnvironment)
	if err != nil {
		return err
	}

	azCred, err := r.credentialsGetter(ctx)
	if err != nil {
		return err
	}

	spDynamic, err := r.spValidatorConstructor(&azEnv)
	if err != nil {
		return err
	}

	return spDynamic.ValidateServicePrincipal(ctx, string(azCred.ClientID), string(azCred.ClientSecret), string(azCred.TenantID))
}
