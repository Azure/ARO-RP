package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/util/feature"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) Install(ctx context.Context) error {
	var (
		installConfig *installconfig.InstallConfig
		image         *releaseimage.Image
		g             graph.Graph
	)

	s := []steps.Step{
		steps.Action(func(ctx context.Context) error {
			var err error
			installConfig, image, err = m.generateInstallConfig(ctx)
			return err
		}),
		steps.Action(func(ctx context.Context) error {
			return m.determineOutboundType(m.oc, m.sub)
		}),
		steps.Action(func(ctx context.Context) error {
			var err error
			// Applies ARO-specific customisations to the InstallConfig
			g, err = m.applyInstallConfigCustomisations(installConfig, image)
			return err
		}),
		steps.Action(func(ctx context.Context) error {
			return m.persistGraph(ctx, g)
		}),
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.deployResourceTemplate)),
		steps.Action(m.initializeKubernetesClients),
		steps.Condition(m.bootstrapConfigMapReady, 30*time.Minute, true),
	}

	_, err := steps.Run(ctx, m.log, 10*time.Second, s, nil)
	return err
}

// initializeKubernetesClients initializes clients using the Installer-generated
// kubeconfig.
func (m *manager) initializeKubernetesClients(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.oc.Properties.StorageSuffix

	// Load the installer's generated kubeconfig from the graph
	pg, err := m.graph.LoadPersisted(ctx, resourceGroup, account)
	if err != nil {
		return err
	}

	var adminInternalClient *kubeconfig.AdminInternalClient
	err = pg.Get(&adminInternalClient)
	if err != nil {
		return err
	}

	// must not proceed if PrivateEndpointIP is not set.  In
	// k8s.io/client-go/transport/cache.go, k8s caches our transport, and it
	// can't tell if data in the restconfig.Dial closure has changed.  We don't
	// want it to cache a transport that can never work.
	if m.oc.Properties.NetworkProfile.APIServerPrivateEndpointIP == "" {
		return errors.New("privateEndpointIP is empty")
	}

	config, err := clientcmd.Load(adminInternalClient.File.Data)
	if err != nil {
		return err
	}

	r, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return err
	}
	r.Dial = restconfig.DialContext(m.env, m.oc)

	m.kubernetescli, err = kubernetes.NewForConfig(r)
	return err
}

func (m *manager) determineOutboundType(oc *api.OpenShiftCluster, sub *api.Subscription) error {
	// Determine if this is a cluster with user defined routing
	oc.Properties.NetworkProfile.OutboundType = api.OutboundTypeLoadbalancer
	if oc.Properties.APIServerProfile.Visibility == api.VisibilityPrivate &&
		oc.Properties.IngressProfiles[0].Visibility == api.VisibilityPrivate &&
		feature.IsRegisteredForFeature(sub.Properties, api.FeatureFlagUserDefinedRouting) {
		oc.Properties.NetworkProfile.OutboundType = api.OutboundTypeUserDefinedRouting
	}
	return nil
}
