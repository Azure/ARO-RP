package bootstraplogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/openshift/installer/pkg/asset/bootstraplogging"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// GetConfig prepares a bootstraplogging.Config object based on
// the environment
func GetConfig(env env.Interface, gl env.ClustersGenevaLoggingInterface, doc *api.OpenShiftClusterDocument) (*bootstraplogging.Config, error) {
	versions, err := version.New(env)
	if err != nil {
		return nil, err
	}

	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	gcsKeyBytes, gcsCertBytes := gl.ClustersGenevaLoggingSecret()

	environment, configVersion := genevalogging.EnvironmentAndVersion(env)

	return &bootstraplogging.Config{
		Certificate:       string(gcsCertBytes),
		Key:               string(gcsKeyBytes),
		Namespace:         genevalogging.ClusterLogsNamespace,
		Environment:       environment,
		ConfigVersion:     configVersion,
		Region:            env.Location(),
		ResourceID:        doc.OpenShiftCluster.ID,
		SubscriptionID:    r.SubscriptionID,
		ResourceName:      r.ResourceName,
		ResourceGroupName: r.ResourceGroup,
		MdsdImage:         versions.GetVersion(version.MDSD),
		FluentbitImage:    versions.GetVersion(version.Fluentbit),
	}, nil
}
