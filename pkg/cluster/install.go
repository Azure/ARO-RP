package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/go-semver/semver"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	imageregistryclient "github.com/openshift/client-go/imageregistry/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	samplesclient "github.com/openshift/client-go/samples/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/containerinstall"
	"github.com/Azure/ARO-RP/pkg/database"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/deploy"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	utilgenerics "github.com/Azure/ARO-RP/pkg/util/generics"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	operatorCutOffVersion = "4.7.0" // OCP versions older than this will not receive ARO operator updates
)

// AdminUpdate performs an admin update of an ARO cluster
func (m *manager) AdminUpdate(ctx context.Context) error {
	toRun := m.adminUpdate()
	return m.runSteps(ctx, toRun, "adminUpdate")
}

func (m *manager) adminUpdate() []steps.Step {
	task := m.doc.OpenShiftCluster.Properties.MaintenanceTask
	isEverything := task == api.MaintenanceTaskEverything || task == ""
	isOperator := task == api.MaintenanceTaskOperator
	isRenewCerts := task == api.MaintenanceTaskRenewCerts
	isSyncClusterObject := task == api.MaintenanceTaskSyncClusterObject

	stepsToRun := m.getZerothSteps()
	if isEverything {
		stepsToRun = utilgenerics.ConcatMultipleSlices(
			stepsToRun, m.getGeneralFixesSteps(), m.getCertificateRenewalSteps(),
		)
		if m.shouldUpdateOperator() {
			stepsToRun = append(stepsToRun, m.getOperatorUpdateSteps()...)
		}
		if m.adoptViaHive && !m.clusterWasCreatedByHive() {
			stepsToRun = append(stepsToRun, m.getHiveAdoptionAndReconciliationSteps()...)
		}
	} else if isOperator {
		if m.shouldUpdateOperator() {
			stepsToRun = append(stepsToRun, m.getOperatorUpdateSteps()...)
		}
	} else if isRenewCerts {
		stepsToRun = append(stepsToRun, m.getCertificateRenewalSteps()...)
	} else if isSyncClusterObject {
		stepsToRun = append(stepsToRun, m.getSyncClusterObjectSteps()...)
	}

	// We don't run this on an operator-only deploy as PUCM scripts then cannot
	// determine if the cluster has been fully admin-updated
	// Run this last so we capture the resource provider only once the upgrade has been fully performed
	if isEverything {
		stepsToRun = append(stepsToRun, steps.Action(m.updateProvisionedBy))
	}

	return stepsToRun
}

func (m *manager) getZerothSteps() []steps.Step {
	bootstrap := []steps.Step{
		steps.Action(m.initializeKubernetesClients), // must be first
		steps.Action(m.ensureBillingRecord),         // belt and braces
		steps.Action(m.ensureDefaults),
		steps.Action(m.fixupClusterSPObjectID),
	}

	// Generic fix-up actions that are fairly safe to always take, and don't require a running cluster
	step0 := []steps.Step{
		steps.Action(m.fixInfraID), // Old clusters lacks infraID in the database. Which makes code prone to errors.
	}

	return append(bootstrap, step0...)
}

func (m *manager) getEnsureAPIServerReadySteps() []steps.Step {
	return []steps.Step{
		steps.Action(m.startVMs),
		steps.Condition(m.apiServersReady, 30*time.Minute, true),
	}
}

func (m *manager) getGeneralFixesSteps() []steps.Step {
	stepsThatDontNeedAPIServer := []steps.Step{
		steps.Action(m.ensureResourceGroup), // re-create RP RBAC if needed after tenant migration
		steps.Action(m.createOrUpdateDenyAssignment),
		steps.Action(m.ensureServiceEndpoints),
		steps.Action(m.populateRegistryStorageAccountName), // must go before migrateStorageAccounts
		steps.Action(m.migrateStorageAccounts),
		steps.Action(m.fixSSH),
		// steps.Action(m.removePrivateDNSZone), // TODO(mj): re-enable once we communicate this out

	}
	stepsThatNeedAPIServer := []steps.Step{
		steps.Action(m.fixSREKubeconfig),
		steps.Action(m.fixUserAdminKubeconfig),
		steps.Action(m.createOrUpdateRouterIPFromCluster),

		steps.Action(m.ensureGatewayUpgrade),
		steps.Action(m.rotateACRTokenPassword),

		steps.Action(m.populateRegistryStorageAccountName),
		steps.Action(m.ensureMTUSize),
	}
	return utilgenerics.ConcatMultipleSlices(
		stepsThatDontNeedAPIServer,
		m.getEnsureAPIServerReadySteps(),
		stepsThatNeedAPIServer,
	)
}

func (m *manager) getCertificateRenewalSteps() []steps.Step {
	steps := []steps.Step{
		steps.Action(m.populateDatabaseIntIP),
		steps.Action(m.correctCertificateIssuer),
		steps.Action(m.fixMCSCert),
		steps.Action(m.fixMCSUserData),
		steps.Action(m.configureAPIServerCertificate),
		steps.Action(m.configureIngressCertificate),

		steps.Action(m.initializeOperatorDeployer),

		steps.Action(m.renewMDSDCertificate), // Dependent on initializeOperatorDeployer.
	}
	return utilgenerics.ConcatMultipleSlices(m.getEnsureAPIServerReadySteps(), steps)
}

func (m *manager) getOperatorUpdateSteps() []steps.Step {
	steps := []steps.Step{
		steps.Action(m.initializeOperatorDeployer),

		steps.Action(m.ensureAROOperator),

		// The following are dependent on initializeOperatorDeployer.
		steps.Condition(m.aroDeploymentReady, 20*time.Minute, true),
		steps.Condition(m.ensureAROOperatorRunningDesiredVersion, 5*time.Minute, true),

		// Once the ARO Operator is updated, synchronize the Cluster object. This is
		// done after the ARO Operator is potentially updated so that any flag
		// changes that happen in the same request only apply on the new Operator.
		// Otherwise, it is possible for a flag change to occur on the old Operator
		// version, then require reconciling to a new version a second time (e.g.
		// DNSMasq changes) with the associated node cyclings for the resource
		// updates.
		steps.Action(m.syncClusterObject),
	}

	return utilgenerics.ConcatMultipleSlices(m.getEnsureAPIServerReadySteps(), steps)
}

func (m *manager) getSyncClusterObjectSteps() []steps.Step {
	steps := []steps.Step{
		steps.Action(m.initializeOperatorDeployer),
		steps.Action(m.syncClusterObject),
	}
	return utilgenerics.ConcatMultipleSlices(m.getEnsureAPIServerReadySteps(), steps)
}

func (m *manager) getHiveAdoptionAndReconciliationSteps() []steps.Step {
	return []steps.Step{
		steps.Action(m.hiveCreateNamespace),
		steps.Action(m.hiveEnsureResources),
		steps.Condition(m.hiveClusterDeploymentReady, 5*time.Minute, false),
		steps.Action(m.hiveResetCorrelationData),
	}
}

func (m *manager) shouldUpdateOperator() bool {
	runningVersion, err := semver.NewVersion(m.doc.OpenShiftCluster.Properties.ClusterProfile.Version)
	if err != nil {
		return false
	}

	cutOffVersion := semver.New(operatorCutOffVersion)
	return cutOffVersion.Compare(*runningVersion) <= 0
}

func (m *manager) clusterWasCreatedByHive() bool {
	if m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace == "" {
		return false
	}

	return m.doc.OpenShiftCluster.Properties.HiveProfile.CreatedByHive
}

func (m *manager) Update(ctx context.Context) error {
	s := []steps.Step{
		steps.AuthorizationRetryingAction(m.fpAuthorizer, m.validateResources),
		steps.Action(m.initializeKubernetesClients), // All init steps are first
		steps.Action(m.initializeOperatorDeployer),  // depends on kube clients
		// Since ServicePrincipalProfile is now a pointer and our converters re-build the struct,
		// our update path needs to enrich the doc with SPObjectID since it was overwritten by our API on put/patch.
		steps.AuthorizationRetryingAction(m.fpAuthorizer, m.fixupClusterSPObjectID),

		// credentials rotation flow steps
		steps.Action(m.createOrUpdateClusterServicePrincipalRBAC),
		steps.Action(m.createOrUpdateDenyAssignment),
		steps.Action(m.startVMs),
		steps.Condition(m.apiServersReady, 30*time.Minute, true),
		steps.Action(m.rotateACRTokenPassword),
		steps.Action(m.correctCertificateIssuer),
		steps.Action(m.configureAPIServerCertificate),
		steps.Action(m.configureIngressCertificate),
		steps.Action(m.renewMDSDCertificate),
		steps.Action(m.ensureCredentialsRequest),
		steps.Action(m.updateOpenShiftSecret),
		steps.Condition(m.aroCredentialsRequestReconciled, 3*time.Minute, true),
		steps.Action(m.updateAROSecret),
		steps.Action(m.restartAROOperatorMaster), // depends on m.updateOpenShiftSecret; the point of restarting is to pick up any changes made to the secret
		steps.Condition(m.aroDeploymentReady, 5*time.Minute, true),
		steps.Action(m.ensureUpgradeAnnotation),
		steps.Action(m.reconcileLoadBalancerProfile),
	}

	if m.adoptViaHive {
		s = append(s,
			// Hive reconciliation: we mostly need it to make sure that
			// hive has the latest credentials after rotation.
			steps.Action(m.hiveCreateNamespace),
			steps.Action(m.hiveEnsureResources),
			steps.Condition(m.hiveClusterDeploymentReady, 5*time.Minute, true),
			steps.Action(m.hiveResetCorrelationData),
		)
	}

	return m.runSteps(ctx, s, "update")
}

func (m *manager) runPodmanInstaller(ctx context.Context) error {
	version, err := m.openShiftVersionFromVersion(ctx)
	if err != nil {
		return err
	}

	i, err := containerinstall.New(ctx, m.log, m.env, m.doc.ID)
	if err != nil {
		return err
	}

	return i.Install(ctx, m.subscriptionDoc, m.doc, version)
}

func (m *manager) runHiveInstaller(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, setFieldCreatedByHive(true))
	if err != nil {
		return err
	}

	version, err := m.openShiftVersionFromVersion(ctx)
	if err != nil {
		return err
	}

	var customManifests map[string]kruntime.Object
	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		customManifests, err = m.generateWorkloadIdentityResources()
		if err != nil {
			return err
		}
	}

	// Run installer. For M5/M6 we will persist the graph inside the installer
	// code since it's easier, but in the future, this data should be collected
	// from Hive's outputs where needed.
	return m.hiveClusterManager.Install(ctx, m.subscriptionDoc, m.doc, version, customManifests)
}

func setFieldCreatedByHive(createdByHive bool) database.OpenShiftClusterDocumentMutator {
	return func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.HiveProfile.CreatedByHive = createdByHive
		return nil
	}
}

func (m *manager) bootstrap() []steps.Step {
	s := []steps.Step{}

	// initialize required clients to manage cluster credentials and populate
	// clientIDs/objectIDs for them (both CSP and WI)
	// TODO: ensuring credential IDs relies on authorizers that aren't exposed in the
	// manager struct, so we'll rebuild the fpAuthorizer and use the error catching
	// to advance
	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		s = append(s,
			steps.Action(m.ensureClusterMsiCertificate),
			steps.Action(m.initializeClusterMsiClients),
			steps.AuthorizationRetryingAction(m.fpAuthorizer, m.clusterIdentityIDs),
			steps.AuthorizationRetryingAction(m.fpAuthorizer, m.platformWorkloadIdentityIDs),
		)
	} else {
		s = append(s,
			steps.Action(m.initializeClusterSPClients),
			steps.AuthorizationRetryingAction(m.fpAuthorizer, m.clusterSPObjectID),
		)
	}

	s = append(s,
		steps.AuthorizationRetryingAction(m.fpAuthorizer, m.validateResources),
		steps.Action(m.ensurePreconfiguredNSG),
		steps.Action(m.ensureACRToken),
		steps.Action(m.ensureInfraID),
		steps.Action(m.ensureSSHKey),
		steps.Action(m.ensureStorageSuffix),
		steps.Action(m.populateMTUSize),
		steps.Action(m.createDNS),
		steps.Action(m.createOIDC),
		steps.Action(m.ensureResourceGroup),
		steps.Action(m.ensureServiceEndpoints),
		steps.Action(m.setMasterSubnetPolicies),
		steps.AuthorizationRetryingAction(m.fpAuthorizer, m.deployBaseResourceTemplate),
		steps.AuthorizationRetryingAction(m.fpAuthorizer, m.attachNSGs),
		steps.Action(m.updateAPIIPEarly),
		steps.Action(m.createOrUpdateRouterIPEarly),
		steps.Action(m.ensureGatewayCreate),
		steps.Action(m.createAPIServerPrivateEndpoint),
		steps.Action(m.createCertificates),
	)

	if m.adoptViaHive || m.installViaHive {
		// We will always need a Hive namespace, whether we are installing
		// via Hive or adopting
		s = append(s, steps.Action(m.hiveCreateNamespace))
	}

	if m.installViaHive {
		s = append(s,
			steps.Action(m.runHiveInstaller),
			// Give Hive 60 minutes to install the cluster, since this includes
			// all of bootstrapping being complete
			steps.Condition(m.hiveClusterInstallationComplete, 60*time.Minute, true),
			steps.Condition(m.hiveClusterDeploymentReady, 5*time.Minute, true),
			steps.Action(m.generateKubeconfigs),
		)
	} else {
		s = append(s,
			steps.Action(m.runPodmanInstaller),
			steps.Action(m.generateKubeconfigs),
		)

		if m.adoptViaHive {
			s = append(s,
				steps.Action(m.hiveEnsureResources),
				steps.Condition(m.hiveClusterDeploymentReady, 5*time.Minute, true),
			)
		}
	}

	if m.adoptViaHive || m.installViaHive {
		s = append(s,
			// Reset correlation data whether adopting or installing via Hive
			steps.Action(m.hiveResetCorrelationData),
		)
	}

	s = append(s,
		steps.Action(m.ensureBillingRecord),
		steps.Action(m.initializeKubernetesClients),
		steps.Action(m.initializeOperatorDeployer), // depends on kube clients
		steps.Condition(m.apiServersReady, 30*time.Minute, true),
		steps.Action(m.installAROOperator),
		steps.Action(m.enableOperatorReconciliation),
		steps.Action(m.incrInstallPhase),
	)

	return s
}

// Install installs an ARO cluster
func (m *manager) Install(ctx context.Context) error {
	steps := map[api.InstallPhase][]steps.Step{
		api.InstallPhaseBootstrap: m.bootstrap(),
		api.InstallPhaseRemoveBootstrap: {
			steps.Action(m.initializeKubernetesClients),
			steps.Action(m.initializeOperatorDeployer), // depends on kube clients
			steps.Action(m.removeBootstrap),
			steps.Action(m.removeBootstrapIgnition),
			// Occasionally, the apiserver experiences disruptions, causing the certificate configuration step to fail.
			// This issue is currently under investigation.
			steps.Condition(m.apiServersReady, 30*time.Minute, true),
			steps.Action(m.configureAPIServerCertificate),
			steps.Condition(m.apiServersReady, 30*time.Minute, true),
			steps.Condition(m.minimumWorkerNodesReady, 30*time.Minute, true),
			steps.Condition(m.operatorConsoleExists, 30*time.Minute, true),
			steps.Action(m.updateConsoleBranding),
			steps.Condition(m.operatorConsoleReady, 20*time.Minute, true),
			steps.Action(m.disableSamples),
			steps.Action(m.disableOperatorHubSources),
			steps.Action(m.disableUpdates),
			steps.Condition(m.clusterVersionReady, 30*time.Minute, true),
			steps.Condition(m.aroDeploymentReady, 20*time.Minute, true),
			steps.Action(m.updateClusterData),
			steps.Action(m.configureIngressCertificate),
			steps.Condition(m.ingressControllerReady, 30*time.Minute, true),
			steps.Action(m.configureDefaultStorageClass),
			steps.Action(m.disableOperatorReconciliation),
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
	return m.runSteps(ctx, steps[m.doc.OpenShiftCluster.Properties.Install.Phase], "install")
}

func (m *manager) runSteps(ctx context.Context, s []steps.Step, metricsTopic string) error {
	var err error
	if metricsTopic != "" {
		var stepsTimeRun map[string]int64
		stepsTimeRun, err = steps.Run(ctx, m.log, 10*time.Second, s, m.now)
		if err == nil {
			var totalInstallTime int64
			for stepName, duration := range stepsTimeRun {
				metricName := fmt.Sprintf("backend.openshiftcluster.%s.%s.duration.seconds", metricsTopic, stepName)
				m.metricsEmitter.EmitGauge(metricName, duration, nil)
				totalInstallTime += duration
			}

			metricName := fmt.Sprintf("backend.openshiftcluster.%s.duration.total.seconds", metricsTopic)
			m.metricsEmitter.EmitGauge(metricName, totalInstallTime, nil)
		}
	} else {
		_, err = steps.Run(ctx, m.log, 10*time.Second, s, nil)
	}
	if err != nil {
		m.gatherFailureLogs(ctx)
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

	m.dynamiccli, err = dynamic.NewForConfig(restConfig)
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
	if err != nil {
		return err
	}

	mapper, err := apiutil.NewDynamicRESTMapper(restConfig, apiutil.WithLazyDiscovery)
	if err != nil {
		return err
	}

	client, err := client.New(restConfig, client.Options{
		Mapper: mapper,
	})

	m.ch = clienthelper.NewWithClient(m.log, client)
	return err
}

// initializeKubernetesClients initializes clients which are used
// once the cluster is up later on in the install process.
func (m *manager) initializeOperatorDeployer(ctx context.Context) (err error) {
	m.aroOperatorDeployer, err = deploy.New(m.log, m.env, m.doc.OpenShiftCluster, m.subscriptionDoc, m.arocli, m.ch, m.extensionscli, m.kubernetescli, m.operatorcli)
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
