package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	imageregistryclient "github.com/openshift/client-go/imageregistry/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	samplesclient "github.com/openshift/client-go/samples/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// AdminUpdate performs an admin update of an ARO cluster
func (m *manager) AdminUpdate(ctx context.Context) error {
	toRun := m.adminUpdate()
	return m.runSteps(ctx, toRun)
}

func (m *manager) adminUpdate() []steps.Step {
	task := m.doc.OpenShiftCluster.Properties.MaintenanceTask
	isEverything := task == api.MaintenanceTaskEverything || task == ""
	isOperator := task == api.MaintenanceTaskOperator

	// Generic fix-up or setup actions that are fairly safe to always take, and
	// don't require a running cluster
	toRun := []steps.Step{
		steps.Action(m.initializeKubernetesClients), // must be first
		steps.Action(m.ensureBillingRecord),         // belt and braces
		steps.Action(m.ensureDefaults),
		steps.Action(m.fixupClusterSPObjectID),
		steps.Action(m.fixInfraID), // Old clusters lacks infraID in the database. Which makes code prone to errors.
	}

	if isEverything {
		toRun = append(toRun,
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.ensureResourceGroup)), // re-create RP RBAC if needed after tenant migration
			steps.Action(m.createOrUpdateDenyAssignment),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.enableServiceEndpoints)),
			steps.Action(m.populateRegistryStorageAccountName), // must go before migrateStorageAccounts
			steps.Action(m.migrateStorageAccounts),
			steps.Action(m.fixSSH),
			steps.Action(m.populateDatabaseIntIP),
			//steps.Action(m.removePrivateDNSZone), // TODO(mj): re-enable once we communicate this out
		)
	}

	// Make sure the VMs are switched on and we have an APIServer
	toRun = append(toRun,
		steps.Action(m.startVMs),
		steps.Condition(m.apiServersReady, 30*time.Minute, true),
	)

	// Requires Kubernetes clients
	if isEverything {
		toRun = append(toRun,
			steps.Action(m.fixSREKubeconfig),
			steps.Action(m.fixUserAdminKubeconfig),
			steps.Action(m.createOrUpdateRouterIPFromCluster),
			steps.Action(m.fixMCSCert),
			steps.Action(m.fixMCSUserData),
			steps.Action(m.ensureGatewayUpgrade),
			steps.Action(m.configureAPIServerCertificate),
			steps.Action(m.configureIngressCertificate),
			steps.Action(m.populateRegistryStorageAccountName),
			steps.Action(m.populateCreatedAt), // TODO(mikalai): Remove after a round of admin updates

		)
	}

	// Update the ARO Operator
	if isEverything || isOperator {
		toRun = append(toRun,
			steps.Action(m.ensureAROOperator),
			steps.Condition(m.aroDeploymentReady, 20*time.Minute, true),
		)
	}

	// We don't run this on an operator-only deploy as PUCM scripts then cannot
	// determine if the cluster has been fully admin-updated
	if isEverything {
		toRun = append(toRun,
			steps.Action(m.updateProvisionedBy), // Run this last so we capture the resource provider only once the upgrade has been fully performed
		)
	}

	return toRun
}

func (m *manager) Update(ctx context.Context) error {
	steps := []steps.Step{
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.validateResources)),
		steps.Action(m.initializeKubernetesClients), // All init steps are first
		steps.Action(m.initializeClusterSPClients),
		steps.Action(m.clusterSPObjectID),
		// credentials rotation flow steps
		steps.Action(m.createOrUpdateClusterServicePrincipalRBAC),
		steps.Action(m.createOrUpdateDenyAssignment),
		steps.Action(m.updateOpenShiftSecret),
		steps.Action(m.updateAROSecret),
	}

	return m.runSteps(ctx, steps)
}

// Install installs an ARO cluster
func (m *manager) Install(ctx context.Context) error {

	var (
		installConfig *installconfig.InstallConfig
		image         *releaseimage.Image
	)

	steps := map[api.InstallPhase][]steps.Step{
		api.InstallPhaseBootstrap: {
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.validateResources)),
			steps.Action(m.ensureACRToken),
			steps.Action(m.generateSSHKey),
			steps.Action(m.generateFIPSMode),
			steps.Action(func(ctx context.Context) error {
				var err error
				installConfig, image, err = m.generateInstallConfig(ctx)
				return err
			}),
			steps.Action(m.createDNS),
			steps.Action(m.initializeClusterSPClients), // must run before clusterSPObjectID
			steps.Action(m.clusterSPObjectID),
			steps.Action(func(ctx context.Context) error {
				return m.ensureInfraID(ctx, installConfig)
			}),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.ensureResourceGroup)),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.enableServiceEndpoints)),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.setMasterSubnetPolicies)),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(func(ctx context.Context) error {
				return m.deployStorageTemplate(ctx, installConfig)
			})),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.updateAPIIPEarly)),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.createOrUpdateRouterIPEarly)),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.ensureGatewayCreate)),
			steps.Action(func(ctx context.Context) error {
				return m.ensureGraph(ctx, installConfig, image)
			}),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.attachNSGs)),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.generateKubeconfigs)),
			steps.Action(m.ensureBillingRecord),
			steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.deployResourceTemplate)),
			steps.Action(m.createAPIServerPrivateEndpoint),
			steps.Action(m.createCertificates),
			steps.Action(m.initializeKubernetesClients),
			steps.Condition(m.bootstrapConfigMapReady, 30*time.Minute, true),
			steps.Action(m.ensureAROOperator),
			steps.Action(m.incrInstallPhase),
		},
		api.InstallPhaseRemoveBootstrap: {
			steps.Action(m.initializeKubernetesClients),
			steps.Action(m.removeBootstrap),
			steps.Action(m.removeBootstrapIgnition),
			steps.Action(m.configureAPIServerCertificate),
			steps.Condition(m.apiServersReady, 30*time.Minute, true),
			steps.Condition(m.minimumWorkerNodesReady, 30*time.Minute, true),
			steps.Condition(m.operatorConsoleExists, 30*time.Minute, true),
			steps.Action(m.updateConsoleBranding),
			steps.Condition(m.operatorConsoleReady, 20*time.Minute, true),
			steps.Condition(m.clusterVersionReady, 30*time.Minute, true),
			steps.Condition(m.aroDeploymentReady, 20*time.Minute, true),
			steps.Action(m.disableUpdates),
			steps.Action(m.disableSamples),
			steps.Action(m.disableOperatorHubSources),
			steps.Action(m.updateClusterData),
			steps.Action(m.configureIngressCertificate),
			steps.Condition(m.ingressControllerReady, 30*time.Minute, true),
			steps.Action(m.configureDefaultStorageClass),
			steps.Action(m.finishInstallation),
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

	m.extensionscli, err = extensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	m.maocli, err = maoclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	m.mcocli, err = mcoclient.NewForConfig(restConfig)
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
	if err != nil {
		return err
	}

	m.imageregistrycli, err = imageregistryclient.NewForConfig(restConfig)
	return err
}

// updateProvisionedBy sets the deploying resource provider version in
// the cluster document for deployment-tracking purposes.
func (m *manager) updateProvisionedBy(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ProvisionedBy = version.GitCommit
		return nil
	})
	return err
}
