package bootstraplogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/openshift/installer/pkg/asset/bootstraplogging"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// GetConfig prepares a bootstraplogging.Config object based on
// the environment
func GetConfig(env env.Interface, oc *api.OpenShiftCluster) (*bootstraplogging.Config, error) {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return nil, err
	}

	key, cert := env.ClusterGenevaLoggingSecret()

	gcsKeyBytes, err := utilpem.Encode(key)
	if err != nil {
		return nil, err
	}

	gcsCertBytes, err := utilpem.Encode(cert)
	if err != nil {
		return nil, err
	}

	return &bootstraplogging.Config{
		Certificate:       string(gcsCertBytes),
		Key:               string(gcsKeyBytes),
		Namespace:         env.ClusterGenevaLoggingNamespace(),
		Account:           env.ClusterGenevaLoggingAccount(),
		Environment:       env.ClusterGenevaLoggingEnvironment(),
		ConfigVersion:     env.ClusterGenevaLoggingConfigVersion(),
		Region:            env.Location(),
		ResourceID:        oc.ID,
		SubscriptionID:    r.SubscriptionID,
		ResourceName:      r.ResourceName,
		ResourceGroupName: r.ResourceGroup,
		MdsdImage:         version.MdsdImage(env.ACRDomain()),
		FluentbitImage:    version.FluentbitImage(env.ACRDomain()),
	}, nil
}
