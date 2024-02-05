package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"

	"github.com/go-test/deep"

	"github.com/Azure/ARO-RP/pkg/api"
)

func concatMultipleSlices[T any](slices ...[]T) []T {
	result := []T{}

	for _, s := range slices {
		result = append(result, s...)
	}

	return result
}

func TestAdminUpdateSteps(t *testing.T) {
	const (
		key = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
	)

	baseClusterDoc := func() *api.OpenShiftClusterDocument {
		return &api.OpenShiftClusterDocument{
			Key: strings.ToLower(key),
			OpenShiftCluster: &api.OpenShiftCluster{
				ID: key,
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Version: "4.10.0",
					},
				},
			},
		}
	}

	zerothSteps := []string{
		"[Action initializeKubernetesClients-fm]",
		"[Action ensureBillingRecord-fm]",
		"[Action ensureDefaults-fm]",
		"[AuthorizationRetryingAction fixupClusterSPObjectID-fm]",
		"[Action startVMs-fm]",
		"[Condition apiServersReady-fm, timeout 30m0s]",
		"[Action fixInfraID-fm]",
	}

	generalFixesSteps := []string{
		"[Action ensureResourceGroup-fm]",
		"[Action createOrUpdateDenyAssignment-fm]",
		"[Action ensureServiceEndpoints-fm]",
		"[Action populateRegistryStorageAccountName-fm]",
		"[Action migrateStorageAccounts-fm]",
		"[Action fixSSH-fm]",
		"[Action fixSREKubeconfig-fm]",
		"[Action fixUserAdminKubeconfig-fm]",
		"[Action createOrUpdateRouterIPFromCluster-fm]",
		"[Action ensureGatewayUpgrade-fm]",
		"[Action rotateACRTokenPassword-fm]",
		"[Action populateRegistryStorageAccountName-fm]",
		"[Action ensureMTUSize-fm]",
	}

	certificateRenewalSteps := []string{
		"[Action populateDatabaseIntIP-fm]",
		"[Action fixMCSCert-fm]",
		"[Action fixMCSUserData-fm]",
		"[Action configureAPIServerCertificate-fm]",
		"[Action configureIngressCertificate-fm]",
		"[Action initializeOperatorDeployer-fm]",
		"[Action renewMDSDCertificate-fm]",
	}

	operatorUpdateSteps := []string{
		"[Action initializeOperatorDeployer-fm]",
		"[Action ensureAROOperator-fm]",
		"[Condition aroDeploymentReady-fm, timeout 20m0s]",
		"[Condition ensureAROOperatorRunningDesiredVersion-fm, timeout 5m0s]",
	}

	hiveSteps := []string{
		"[Action hiveCreateNamespace-fm]",
		"[Action hiveEnsureResources-fm]",
		"[Condition hiveClusterDeploymentReady-fm, timeout 5m0s]",
		"[Action hiveResetCorrelationData-fm]",
	}

	updateProvisionedBySteps := []string{"[Action updateProvisionedBy-fm]"}

	for _, tt := range []struct {
		name           string
		fixture        func() (doc *api.OpenShiftClusterDocument, adoptHive bool)
		shouldRunSteps []string
	}{
		{
			name: "ARO Operator Update",
			fixture: func() (*api.OpenShiftClusterDocument, bool) {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskOperator
				return doc, true
			},
			shouldRunSteps: concatMultipleSlices(zerothSteps, operatorUpdateSteps),
		},
		{
			name: "ARO Operator Update on <= 4.6 cluster does not update operator",
			fixture: func() (*api.OpenShiftClusterDocument, bool) {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskOperator
				doc.OpenShiftCluster.Properties.ClusterProfile.Version = "4.6.62"
				return doc, true
			},
			shouldRunSteps: zerothSteps,
		},
		{
			name: "ARO Operator Update on 4.7.0 cluster does update operator",
			fixture: func() (*api.OpenShiftClusterDocument, bool) {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskOperator
				doc.OpenShiftCluster.Properties.ClusterProfile.Version = "4.7.0"
				return doc, true
			},
			shouldRunSteps: concatMultipleSlices(zerothSteps, operatorUpdateSteps),
		},
		{
			name: "Everything update and adopt Hive.",
			fixture: func() (*api.OpenShiftClusterDocument, bool) {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				return doc, true
			},
			shouldRunSteps: concatMultipleSlices(
				zerothSteps, generalFixesSteps, certificateRenewalSteps,
				operatorUpdateSteps, hiveSteps, updateProvisionedBySteps,
			),
		},
		{
			name: "Everything update on <= 4.6 cluster does not update operator",
			fixture: func() (*api.OpenShiftClusterDocument, bool) {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.ClusterProfile.Version = "4.6.62"
				return doc, true
			},
			shouldRunSteps: concatMultipleSlices(
				zerothSteps, generalFixesSteps, certificateRenewalSteps,
				hiveSteps, updateProvisionedBySteps,
			),
		},
		{
			name: "Blank, Hive not adopting (should perform everything but Hive)",
			fixture: func() (*api.OpenShiftClusterDocument, bool) {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				return doc, false
			},
			shouldRunSteps: concatMultipleSlices(
				zerothSteps, generalFixesSteps, certificateRenewalSteps,
				operatorUpdateSteps, updateProvisionedBySteps,
			),
		},
		{
			name: "Blank, Hive adopting (should perform everything)",
			fixture: func() (*api.OpenShiftClusterDocument, bool) {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				return doc, true
			},
			shouldRunSteps: concatMultipleSlices(
				zerothSteps, generalFixesSteps, certificateRenewalSteps,
				operatorUpdateSteps, hiveSteps, updateProvisionedBySteps,
			),
		},
		{
			name: "Rotate in-cluster MDSD/Ingress/API certs",
			fixture: func() (*api.OpenShiftClusterDocument, bool) {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskRenewCerts
				return doc, true
			},
			shouldRunSteps: concatMultipleSlices(zerothSteps, certificateRenewalSteps),
		},
		{
			name: "adminUpdate() does not adopt Hive-created clusters",
			fixture: func() (*api.OpenShiftClusterDocument, bool) {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.HiveProfile.Namespace = "aro-00000000-0000-0000-0000-000000000000"
				doc.OpenShiftCluster.Properties.HiveProfile.CreatedByHive = true
				return doc, true
			},
			shouldRunSteps: concatMultipleSlices(
				zerothSteps, generalFixesSteps, certificateRenewalSteps,
				operatorUpdateSteps, updateProvisionedBySteps,
			),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc, adoptViaHive := tt.fixture()
			m := &manager{
				doc:          doc,
				adoptViaHive: adoptViaHive,
			}
			toRun := m.adminUpdate()

			var stepsToRun []string
			for _, s := range toRun {
				// make it a little nicer when defining the steps that should run, since they're all methods
				o := strings.Replace(s.String(), "github.com/Azure/ARO-RP/pkg/cluster.(*manager).", "", -1)
				stepsToRun = append(stepsToRun, o)
			}

			diff := deep.Equal(stepsToRun, tt.shouldRunSteps)
			for _, d := range diff {
				t.Error(d)
			}
		})
	}
}
