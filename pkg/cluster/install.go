package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	samplesclient "github.com/openshift/client-go/samples/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/openshift/installer/pkg/asset/bootstraplogging"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// AdminUpgrade performs an admin upgrade of an ARO cluster
func (m *manager) AdminUpgrade(ctx context.Context) error {
	steps := []steps.Step{
		steps.Action(m.initializeKubernetesClients), // must be first
		steps.Action(m.deploySnapshotUpgradeTemplate),
		steps.Action(m.startVMs),
		steps.Condition(m.apiServersReady, 30*time.Minute),
		steps.Action(m.ensureBillingRecord), // belt and braces
		steps.Action(m.fixLBProbes),
		steps.Action(m.fixNSG),
		steps.Action(m.fixPullSecret), // TODO(mj): Remove when operator deployed
		steps.Action(m.ensureRouteFix),
		steps.Action(m.ensureAROOperator),
		steps.Condition(m.aroDeploymentReady, 10*time.Minute),
		steps.Action(m.upgradeCertificates),
		steps.Action(m.configureAPIServerCertificate),
		steps.Action(m.configureIngressCertificate),
		steps.Action(m.addResourceProviderVersion), // Run this last so we capture the resource provider only once the upgrade has been fully performed
	}

	return m.runSteps(ctx, steps)
}

// Install installs an ARO cluster
func (m *manager) Install(ctx context.Context, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds, image *releaseimage.Image, bootstrapLoggingConfig *bootstraplogging.Config) error {
	steps := map[api.InstallPhase][]steps.Step{
		api.InstallPhaseBootstrap: {
			steps.Action(m.createDNS),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(func(ctx context.Context) error {
				return m.deployStorageTemplate(ctx, installConfig, platformCreds, image, bootstrapLoggingConfig)
			})),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.attachNSGsAndPatch)),
			steps.Action(m.ensureBillingRecord),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.deployResourceTemplate)),
			steps.Action(m.createPrivateEndpoint),
			steps.Action(m.updateAPIIP),
			steps.Action(m.createCertificates),
			steps.Action(m.initializeKubernetesClients),
			steps.Condition(m.bootstrapConfigMapReady, 60*time.Minute),
			steps.Action(m.ensureRouteFix),
			steps.Action(m.ensureAROOperator),
			steps.Action(m.incrInstallPhase),
		},
		api.InstallPhaseRemoveBootstrap: {
			steps.Action(m.initializeKubernetesClients),
			steps.Action(m.removeBootstrap),
			steps.Action(m.removeBootstrapIgnition),
			steps.Action(m.configureAPIServerCertificate),
			steps.Condition(m.apiServersReady, 30*time.Minute),
			steps.Condition(m.operatorConsoleExists, 30*time.Minute),
			steps.Action(m.updateConsoleBranding),
			steps.Condition(m.operatorConsoleReady, 30*time.Minute),
			steps.Condition(m.clusterVersionReady, 30*time.Minute),
			steps.Condition(m.aroDeploymentReady, 10*time.Minute),
			steps.Action(m.disableUpdates),
			steps.Action(m.disableSamples),
			steps.Action(m.disableOperatorHubSources),
			steps.Action(m.updateRouterIP),
			steps.Action(m.configureIngressCertificate),
			steps.Condition(m.ingressControllerReady, 30*time.Minute),
			steps.Action(m.finishInstallation),
			steps.Action(m.addResourceProviderVersion),
		},
	}

	err := m.startInstallation(ctx)
	if err != nil {
		return err
	}

	if steps[m.doc.OpenShiftCluster.Properties.Install.Phase] == nil {
		return fmt.Errorf("unrecognised phase %s", m.doc.OpenShiftCluster.Properties.Install.Phase)
	}
	m.log.Printf("starting phase %s", m.doc.OpenShiftCluster.Properties.Install.Phase)
	return m.runSteps(ctx, steps[m.doc.OpenShiftCluster.Properties.Install.Phase])
}

func (m *manager) runSteps(ctx context.Context, s []steps.Step) error {
	err := steps.Run(ctx, m.log, 10*time.Second, s)
	if err != nil {
		m.gatherFailureLogs(ctx)
	}
	return err
}

func (m *manager) startInstallation(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		if doc.OpenShiftCluster.Properties.Install == nil {
			doc.OpenShiftCluster.Properties.Install = &api.Install{}
		}
		return nil
	})
	return err
}

func (m *manager) incrInstallPhase(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.Install.Phase++
		return nil
	})
	return err
}

func (m *manager) finishInstallation(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.Install = nil
		return nil
	})
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

	m.extcli, err = extensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	m.operatorcli, err = operatorclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	m.securitycli, err = securityclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	m.samplescli, err = samplesclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	m.arocli, err = aroclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	m.configcli, err = configclient.NewForConfig(restConfig)
	return err
}

// addResourceProviderVersion sets the deploying resource provider version in
// the cluster document for deployment-tracking purposes.
func (m *manager) addResourceProviderVersion(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ProvisionedBy = version.GitCommit
		return nil
	})
	return err
}
