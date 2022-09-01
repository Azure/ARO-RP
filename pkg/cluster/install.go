package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"time"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	imageregistryclient "github.com/openshift/client-go/imageregistry/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	samplesclient "github.com/openshift/client-go/samples/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/installer"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/deploy"
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
			steps.Action(m.ensureMTUSize),
		)
	}

	// Update the ARO Operator
	if isEverything || isOperator {
		toRun = append(toRun,
			steps.Action(m.initializeOperatorDeployer), // depends on kube clients
			steps.Action(m.ensureAROOperator),
			steps.Condition(m.aroDeploymentReady, 20*time.Minute, true),
			steps.Condition(m.ensureAROOperatorRunningDesiredVersion, 5*time.Minute, true),
		)
	}

	// Hive cluster adoption and reconciliation
	if isEverything {
		toRun = append(toRun,
			steps.Action(m.hiveCreateNamespace),
			steps.Action(m.hiveEnsureResources),
			steps.Condition(m.hiveClusterDeploymentReady, 5*time.Minute, false),
			steps.Action(m.hiveResetCorrelationData),
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
		steps.Action(m.initializeOperatorDeployer),  // depends on kube clients
		steps.Action(m.initializeClusterSPClients),
		steps.Action(m.clusterSPObjectID),
		// credentials rotation flow steps
		steps.Action(m.createOrUpdateClusterServicePrincipalRBAC),
		steps.Action(m.createOrUpdateDenyAssignment),
		steps.Action(m.startVMs),
		steps.Condition(m.apiServersReady, 30*time.Minute, true),
		steps.Action(m.configureAPIServerCertificate),
		steps.Action(m.configureIngressCertificate),
		steps.Action(m.updateOpenShiftSecret),
		steps.Action(m.updateAROSecret),
		// Hive reconciliation: we mostly need it to make sure that
		// hive has the latest credentials after rotation.
		steps.Action(m.hiveCreateNamespace),
		steps.Action(m.hiveEnsureResources),
		steps.Condition(m.hiveClusterDeploymentReady, 5*time.Minute, true),
		steps.Action(m.hiveResetCorrelationData),
	}

	return m.runSteps(ctx, steps)
}

func (m *manager) runIntegratedInstaller(ctx context.Context) error {
	i := installer.NewInstaller(m.log, m.env, m.doc.ID, m.doc.OpenShiftCluster, m.subscriptionDoc.Subscription, m.fpAuthorizer, m.deployments, m.graph)
	return i.Install(ctx)
}

func (m *manager) runHiveInstaller(ctx context.Context) error {
	// TODO: Load from M6 database
	installerPullspec, err := m.env.LiveConfig().DefaultInstallerPullSpec(ctx)
	if err != nil {
		return err
	}

	v := &api.OpenShiftVersion{
		Version:           version.InstallStream.Version.String(),
		OpenShiftPullspec: version.InstallStream.PullSpec,
		InstallerPullspec: installerPullspec,
	}

	// Run installer. For M5/M6 we will persist the graph inside the installer
	// code since it's easier, but in the future, this data should be collected
	// from Hive's outputs where needed.
	return m.hiveClusterManager.Install(ctx, m.subscriptionDoc, m.doc, v)
}

func (m *manager) bootstrap() []steps.Step {
	s := []steps.Step{
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.validateResources)),
		steps.Action(m.ensureACRToken),
		steps.Action(m.ensureInfraID),
		steps.Action(m.ensureSSHKey),
		steps.Action(m.ensureStorageSuffix),
		steps.Action(m.populateMTUSize),

		steps.Action(m.createDNS),
		steps.Action(m.initializeClusterSPClients), // must run before clusterSPObjectID
		steps.Action(m.clusterSPObjectID),
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.ensureResourceGroup)),
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.enableServiceEndpoints)),
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.setMasterSubnetPolicies)),
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.deployStorageTemplate)),
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.attachNSGs)),
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.updateAPIIPEarly)),
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.createOrUpdateRouterIPEarly)),
		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.ensureGatewayCreate)),
		steps.Action(m.createAPIServerPrivateEndpoint),
		steps.Action(m.createCertificates),

		// We will always need a Hive namespace, whether we are installing
		// via Hive or adopting
		steps.Action(m.hiveCreateNamespace),
	}

	if m.installViaHive {
		s = append(s,
			steps.Action(m.hiveCreateNamespace),
			steps.Action(m.runHiveInstaller),
			// Give Hive 60 minutes to install the cluster, since this includes
			// all of bootstrapping being complete
			steps.Condition(m.hiveClusterInstallationComplete, 60*time.Minute, true),
			steps.Condition(m.hiveClusterDeploymentReady, 5*time.Minute, true),
		)
	} else {
		s = append(s,
			steps.Action(m.runIntegratedInstaller),
			steps.Action(m.hiveCreateNamespace),
			steps.Action(m.hiveEnsureResources),
			steps.Condition(m.hiveClusterDeploymentReady, 5*time.Minute, true),
		)
	}

	s = append(s,
		// Reset correlation data whether adopting or installing via Hive
		steps.Action(m.hiveResetCorrelationData),

		steps.AuthorizationRefreshingAction(m.fpAuthorizer, steps.Action(m.generateKubeconfigs)),
		steps.Action(m.ensureBillingRecord),
		steps.Action(m.initializeKubernetesClients),
		steps.Action(m.initializeOperatorDeployer), // depends on kube clients
		steps.Condition(m.apiServersReady, 30*time.Minute, true),
		steps.Action(m.ensureAROOperator),
		steps.Action(m.incrInstallPhase),
	)

	return s
}

// Install installs an ARO cluster
func (m *manager) Install(ctx context.Context) error {
	if m.installViaHive && m.hiveClusterManager == nil {
		return errors.New("installViaHive was requested but hiveClusterManager is unavailable")
	}

	steps := map[api.InstallPhase][]steps.Step{
		api.InstallPhaseBootstrap: m.bootstrap(),
		api.InstallPhaseRemoveBootstrap: {
			steps.Action(m.initializeKubernetesClients, "initialize_kubernetes_clients"),
			steps.Action(m.initializeOperatorDeployer, "finishing_phase_initialize_operator_deployer"), // depends on kube clients
			steps.Action(m.removeBootstrap, "remove_bootstrap"),
			steps.Action(m.removeBootstrapIgnition, "remove_bootstrap_ignition"),
			steps.Action(m.configureAPIServerCertificate, "configure_api_server_certificate"),
			steps.Condition(m.apiServersReady, 30*time.Minute, true, "finishing_phase_check_api_server"),
			steps.Condition(m.minimumWorkerNodesReady, 30*time.Minute, true, "check_minimum_worker_nodes"),
			steps.Condition(m.operatorConsoleExists, 30*time.Minute, true, "check_operator_console"),
			steps.Action(m.updateConsoleBranding, "update_console_branding"),
			steps.Condition(m.operatorConsoleReady, 20*time.Minute, true, "check_operator_console"),
			steps.Condition(m.clusterVersionReady, 30*time.Minute, true, "check_cluster_version"),
			steps.Condition(m.aroDeploymentReady, 20*time.Minute, true, "check_aro_deployment"),
			steps.Action(m.disableUpdates, "disable_updates"),
			steps.Action(m.disableSamples, "disable_samples"),
			steps.Action(m.disableOperatorHubSources, "disable_operator_hub_sources"),
			steps.Action(m.updateClusterData, "update_cluster_data"),
			steps.Action(m.configureIngressCertificate, "configure_ingress_certificate"),
			steps.Condition(m.ingressControllerReady, 30*time.Minute, true, "check_ingress_controller"),
			steps.Action(m.configureDefaultStorageClass, "configure_default_storage_class"),
			steps.Action(m.finishInstallation, "finish_installation"),
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
	stepsTimeRun, err := steps.Run(ctx, m.log, 10*time.Second, s, m.now)
	if err != nil {
		m.gatherFailureLogs(ctx)
	} else {
		var totalInstallTime int64
		for topic, duration := range stepsTimeRun {
			m.metricsEmitter.EmitGauge(fmt.Sprintf("backend.openshiftcluster.installtime.%s", topic), duration, nil)
			totalInstallTime += duration
		}
		m.metricsEmitter.EmitGauge("backend.openshiftcluster.installtime.total", totalInstallTime, nil)
	}
	return err
}

func (m *manager) startInstallation(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		if doc.OpenShiftCluster.Properties.Install == nil {
			// set the install time which is used for the SAS token with which
			// the bootstrap node retrieves its ignition payload
			doc.OpenShiftCluster.Properties.Install = &api.Install{
				Now: time.Now().UTC(),
			}
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

	m.maocli, err = machineclient.NewForConfig(restConfig)
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

// initializeKubernetesClients initializes clients which are used
// once the cluster is up later on in the install process.
func (m *manager) initializeOperatorDeployer(ctx context.Context) (err error) {
	m.aroOperatorDeployer, err = deploy.New(m.log, m.env, m.doc.OpenShiftCluster, m.arocli, m.extensionscli, m.kubernetescli)
	return
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
