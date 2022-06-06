package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/steps"
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
			var err error
			// Applies ARO-specific customisations to the InstallConfig
			g, err = m.applyInstallConfigCustomisations(ctx, installConfig, image)
			return err
		}),
		steps.Action(func(ctx context.Context) error {
			return m.persistGraph(ctx, g)
		}),
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.deployResourceTemplate)),
		steps.Action(m.initializeKubernetesClients),
		steps.Condition(m.bootstrapConfigMapReady, 30*time.Minute, true),
	}

	err := steps.Run(ctx, m.log, 10*time.Second, s)
	return err
}

// initializeKubernetesClients initializes clients which are used
// once the cluster is up later on in the install process.
func (m *manager) initializeKubernetesClients(ctx context.Context) error {
	restConfig, err := restconfig.RestConfig(m.env, m.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	m.kubernetescli, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	return err
}
