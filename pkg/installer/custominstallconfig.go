package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/asset/templates/content/bootkube"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/bootstraplogging"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
)

// applyInstallConfigCustomisations modifies the InstallConfig and creates
// parent assets, then regenerates the InstallConfig for use for Ignition
// generation, etc.
func (m *manager) applyInstallConfigCustomisations(ctx context.Context, installConfig *installconfig.InstallConfig, image *releaseimage.Image) (graph.Graph, error) {
	clusterID := &installconfig.ClusterID{
		UUID:    m.doc.ID,
		InfraID: m.doc.OpenShiftCluster.Properties.InfraID,
	}

	bootstrapLoggingConfig, err := bootstraplogging.GetConfig(m.env, m.doc)
	if err != nil {
		return nil, err
	}

	httpSecret := make([]byte, 64)
	_, err = rand.Read(httpSecret)
	if err != nil {
		return nil, err
	}

	imageRegistryConfig := &bootkube.AROImageRegistryConfig{
		AccountName:   m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName,
		ContainerName: "image-registry",
		HTTPSecret:    hex.EncodeToString(httpSecret),
	}

	dnsConfig := &bootkube.ARODNSConfig{
		APIIntIP:  m.doc.OpenShiftCluster.Properties.APIServerProfile.IntIP,
		IngressIP: m.doc.OpenShiftCluster.Properties.IngressProfiles[0].IP,
	}

	if m.doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateEndpointIP != "" {
		dnsConfig.GatewayPrivateEndpointIP = m.doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateEndpointIP
		dnsConfig.GatewayDomains = append(m.env.GatewayDomains(), m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName+".blob."+m.env.Environment().StorageEndpointSuffix)
	}

	g := graph.Graph{}
	g.Set(installConfig, image, clusterID, bootstrapLoggingConfig, dnsConfig, imageRegistryConfig)

	m.log.Print("resolving graph")
	for _, a := range targets.Cluster {
		err = g.Resolve(a)
		if err != nil {
			return nil, err
		}
	}

	// Handle MTU3900 feature flag
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.MTUSize == api.MTU3900 {
		m.log.Printf("applying feature flag %s", api.FeatureFlagMTU3900)
		if err = m.overrideEthernetMTU(g); err != nil {
			return nil, err
		}
	}

	return g, nil
}
