package bootstraplogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/openshift/installer/pkg/asset/bootstraplogging"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	"github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// GetConfig prepares a bootstraplogging.Config object based on
// the environment
func GetConfig(env env.Interface, doc *api.OpenShiftClusterDocument) (*bootstraplogging.Config, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	key, cert := env.ClustersGenevaLoggingSecret()

	gcsKeyBytes, err := tls.PrivateKeyAsBytes(key)
	if err != nil {
		return nil, err
	}

	gcsCertBytes, err := tls.CertAsBytes(cert)
	if err != nil {
		return nil, err
	}

	return &bootstraplogging.Config{
		Certificate:       string(gcsCertBytes),
		Key:               string(gcsKeyBytes),
		Namespace:         genevalogging.ClusterLogsNamespace,
		Environment:       env.ClustersGenevaLoggingEnvironment(),
		ConfigVersion:     env.ClustersGenevaLoggingConfigVersion(),
		Region:            env.Location(),
		ResourceID:        doc.OpenShiftCluster.ID,
		SubscriptionID:    r.SubscriptionID,
		ResourceName:      r.ResourceName,
		ResourceGroupName: r.ResourceGroup,
		MdsdImage:         version.MdsdImage(env.ACRName()),
		FluentbitImage:    version.FluentbitImage(env.ACRName()),
	}, nil
}
