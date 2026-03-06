package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	SharedMSIKeyVaultNameSuffix = "-dev-msi"
)

type dev struct {
	*prod
}

func newDev(ctx context.Context, log *logrus.Entry, component ServiceName) (Interface, error) {
	d := &dev{}

	var err error
	d.prod, err = newProd(ctx, log, component)
	if err != nil {
		return nil, err
	}

	for _, feature := range []Feature{
		FeatureDisableDenyAssignments,
		FeatureDisableSignedCertificates,
		FeatureDisableReadinessDelay,
		FeatureRequireOIDCStorageWebEndpoint,
		FeatureUseMockMsiRp,
	} {
		d.features[feature] = true
	}

	d.clusterGenevaLoggingAccount = version.DevClusterGenevaLoggingAccount
	d.clusterGenevaLoggingConfigVersion = version.DevClusterGenevaLoggingConfigVersion
	d.clusterGenevaLoggingEnvironment = version.DevGenevaLoggingEnvironment
	d.clusterGenevaLoggingNamespace = version.DevClusterGenevaLoggingNamespace

	// ugh: run this again after RP_MODE=development has caused the feature flag
	// to be set.
	d.ARMHelper, err = newARMHelper(ctx, log, d)
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

// OtelAuditQueueSize returns the size of the audit queue for the OTel audit.
// In development environment this size is set to zero as we create noop connection to audit server.
func (d *dev) OtelAuditQueueSize() (int, error) {
	return 0, nil
}

func (d *dev) Listen() (net.Listener, error) {
	if d.Service() == string(SERVICE_MIMO_ACTUATOR) {
		return net.Listen("tcp", ":8445")
	}
	return net.Listen("tcp", ":8443")
}

// TODO: Delete FPAuthorizer once the replace from track1 to track2 is done.
func (d *dev) FPAuthorizer(tenantID string, additionalTenants []string, scopes ...string) (autorest.Authorizer, error) {
	fpTokenCredential, err := d.FPNewClientCertificateCredential(tenantID, additionalTenants)
	if err != nil {
		return nil, err
	}

	return azidext.NewTokenCredentialAdapter(fpTokenCredential, scopes), nil
}

func (d *dev) FPNewClientCertificateCredential(tenantID string, additionalTenants []string) (*azidentity.ClientCertificateCredential, error) {
	fpPrivateKey, fpCertificates := d.fpCertificateRefresher.GetCertificates()

	options := d.Environment().ClientCertificateCredentialOptions(additionalTenants)
	credential, err := azidentity.NewClientCertificateCredential(tenantID, d.fpClientID, fpCertificates, fpPrivateKey, options)
	if err != nil {
		return nil, err
	}

	return credential, nil
}

func (d *dev) MsiRpEndpoint() string {
	return "https://iamaplaceholder.com"
}

func (d *dev) ClusterMsiKeyVaultName() string {
	prefix := os.Getenv("RESOURCEGROUP")
	if len(prefix) > 10 {
		prefix = prefix[:10]
	}
	return prefix + SharedMSIKeyVaultNameSuffix
}
