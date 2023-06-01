package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type dev struct {
	*prod
}

func newDev(ctx context.Context, log *logrus.Entry) (Interface, error) {
	d := &dev{}

	var err error
	d.prod, err = newProd(ctx, log)
	if err != nil {
		return nil, err
	}

	for _, feature := range []Feature{
		FeatureDisableDenyAssignments,
		FeatureDisableSignedCertificates,
		FeatureRequireD2sV3Workers,
		FeatureDisableReadinessDelay,
	} {
		d.features[feature] = true
	}

	d.prod.clusterGenevaLoggingAccount = version.DevClusterGenevaLoggingAccount
	d.prod.clusterGenevaLoggingConfigVersion = version.DevClusterGenevaLoggingConfigVersion
	d.prod.clusterGenevaLoggingEnvironment = version.DevGenevaLoggingEnvironment
	d.prod.clusterGenevaLoggingNamespace = version.DevClusterGenevaLoggingNamespace

	// ugh: run this again after RP_MODE=development has caused the feature flag
	// to be set.
	d.prod.ARMHelper, err = newARMHelper(ctx, log, d)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *dev) InitializeAuthorizers() error {
	d.armClientAuthorizer = clientauthorizer.NewAll()
	d.adminClientAuthorizer = clientauthorizer.NewAll()
	return nil
}

func (d *dev) AROOperatorImage() string {
	override := os.Getenv("ARO_IMAGE")
	if override != "" {
		return override
	}

	return fmt.Sprintf("%s/aro:%s", d.ACRDomain(), version.GitCommit)
}

func (d *dev) Listen() (net.Listener, error) {
	// in dev mode there is no authentication, so for safety we only listen on
	// localhost
	return net.Listen("tcp", "localhost:8443")
}

func (d *dev) FPAuthorizer(tenantID string, scopes ...string) (autorest.Authorizer, error) {
	tokenCredential, err := d.FPNewClientCertificateCredential(tenantID)
	if err != nil {
		return nil, err
	}

	return azidext.NewTokenCredentialAdapter(tokenCredential, scopes), nil
}

func (d *dev) FPNewClientCertificateCredential(tenantID string) (*azidentity.ClientCertificateCredential, error) {
	fpPrivateKey, fpCertificates := d.fpCertificateRefresher.GetCertificates()

	options := d.Environment().ClientCertificateCredentialOptions()
	credential, err := azidentity.NewClientCertificateCredential(tenantID, d.fpClientID, fpCertificates, fpPrivateKey, options)
	if err != nil {
		return nil, err
	}

	return credential, nil
}
