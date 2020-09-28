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
func (i *manager) AdminUpgrade(ctx context.Context) error {
	steps := []steps.Step{
		steps.Action(i.initializeKubernetesClients), // must be first
		steps.Action(i.deploySnapshotUpgradeTemplate),
		steps.Action(i.startVMs),
		steps.Condition(i.apiServersReady, 30*time.Minute),
		steps.Action(i.ensureBillingRecord), // belt and braces
		steps.Action(i.fixLBProbes),
		steps.Action(i.fixNSG),
		steps.Action(i.fixPullSecret), // TODO(mj): Remove when operator deployed
		steps.Action(i.ensureRouteFix),
		steps.Action(i.ensureAROOperator),
		steps.Condition(i.aroDeploymentReady, 10*time.Minute),
		steps.Action(i.upgradeCertificates),
		steps.Action(i.configureAPIServerCertificate),
		steps.Action(i.configureIngressCertificate),
		steps.Action(i.addResourceProviderVersion), // Run this last so we capture the resource provider only once the upgrade has been fully performed
	}

	return i.runSteps(ctx, steps)
}

// Install installs an ARO cluster
func (i *manager) Install(ctx context.Context, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds, image *releaseimage.Image, bootstrapLoggingConfig *bootstraplogging.Config) error {
	steps := map[api.InstallPhase][]steps.Step{
		api.InstallPhaseBootstrap: {
			steps.Action(i.createDNS),
			steps.AuthorizationRefreshingAction(i.fpAuthorizer, steps.Action(func(ctx context.Context) error {
				return i.deployStorageTemplate(ctx, installConfig, platformCreds, image, bootstrapLoggingConfig)
			})),
			steps.AuthorizationRefreshingAction(i.fpAuthorizer, steps.Action(i.attachNSGsAndPatch)),
			steps.Action(i.ensureBillingRecord),
			steps.AuthorizationRefreshingAction(i.fpAuthorizer, steps.Action(i.deployResourceTemplate)),
			steps.Action(i.deployResourceTemplate),
			steps.Action(i.createPrivateEndpoint),
			steps.Action(i.updateAPIIP),
			steps.Action(i.createCertificates),
			steps.Action(i.initializeKubernetesClients),
			steps.Condition(i.bootstrapConfigMapReady, 30*time.Minute),
			steps.Action(i.ensureRouteFix),
			steps.Action(i.ensureAROOperator),
			steps.Action(i.incrInstallPhase),
		},
		api.InstallPhaseRemoveBootstrap: {
			steps.Action(i.initializeKubernetesClients),
			steps.Action(i.removeBootstrap),
			steps.Action(i.removeBootstrapIgnition),
			steps.Action(i.configureAPIServerCertificate),
			steps.Condition(i.apiServersReady, 30*time.Minute),
			steps.Condition(i.operatorConsoleExists, 30*time.Minute),
			steps.Action(i.updateConsoleBranding),
			steps.Condition(i.operatorConsoleReady, 30*time.Minute),
			steps.Condition(i.clusterVersionReady, 30*time.Minute),
			steps.Condition(i.aroDeploymentReady, 10*time.Minute),
			steps.Action(i.disableUpdates),
			steps.Action(i.disableSamples),
			steps.Action(i.disableOperatorHubSources),
			steps.Action(i.updateRouterIP),
			steps.Action(i.configureIngressCertificate),
			steps.Condition(i.ingressControllerReady, 30*time.Minute),
			steps.Action(i.finishInstallation),
			steps.Action(i.addResourceProviderVersion),
		},
	}

	err := i.startInstallation(ctx)
	if err != nil {
		return err
	}

	if steps[i.doc.OpenShiftCluster.Properties.Install.Phase] == nil {
		return fmt.Errorf("unrecognised phase %s", i.doc.OpenShiftCluster.Properties.Install.Phase)
	}
	i.log.Printf("starting phase %s", i.doc.OpenShiftCluster.Properties.Install.Phase)
	return i.runSteps(ctx, steps[i.doc.OpenShiftCluster.Properties.Install.Phase])
}

func (i *manager) runSteps(ctx context.Context, s []steps.Step) error {
	err := steps.Run(ctx, i.log, 10*time.Second, s)
	if err != nil {
		i.gatherFailureLogs(ctx)
	}
	return err
}

func (i *manager) startInstallation(ctx context.Context) error {
	var err error
	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		if doc.OpenShiftCluster.Properties.Install == nil {
			doc.OpenShiftCluster.Properties.Install = &api.Install{}
		}
		return nil
	})
	return err
}

func (i *manager) incrInstallPhase(ctx context.Context) error {
	var err error
	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.Install.Phase++
		return nil
	})
	return err
}

func (i *manager) finishInstallation(ctx context.Context) error {
	var err error
	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.Install = nil
		return nil
	})
	return err
}

// initializeKubernetesClients initializes clients which are used
// once the cluster is up later on in the install process.
func (i *manager) initializeKubernetesClients(ctx context.Context) error {
	restConfig, err := restconfig.RestConfig(i.env, i.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	i.kubernetescli, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.extcli, err = extensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.operatorcli, err = operatorclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.securitycli, err = securityclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.samplescli, err = samplesclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.arocli, err = aroclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.configcli, err = configclient.NewForConfig(restConfig)
	return err
}

// addResourceProviderVersion sets the deploying resource provider version in
// the cluster document for deployment-tracking purposes.
func (i *manager) addResourceProviderVersion(ctx context.Context) error {
	var err error
	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ProvisionedBy = version.GitCommit
		return nil
	})
	return err
}
