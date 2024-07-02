package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"

	"github.com/go-test/deep"

	"github.com/Azure/ARO-RP/pkg/api"
	utilgenerics "github.com/Azure/ARO-RP/pkg/util/generics"
)

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
		"[Action initializeKubernetesClients]",
		"[Action ensureBillingRecord]",
		"[Action ensureDefaults]",
		"[AuthorizationRetryingAction fixupClusterSPObjectID]",
		"[Action fixInfraID]",
	}

	generalFixesSteps := []string{
		"[Action ensureResourceGroup]",
		"[Action createOrUpdateDenyAssignment]",
		"[Action ensureServiceEndpoints]",
		"[Action populateRegistryStorageAccountName]",
		"[Action migrateStorageAccounts]",
		"[Action fixSSH]",
		"[Action startVMs]",
		"[Condition apiServersReady, timeout 30m0s]",
		"[Action fixSREKubeconfig]",
		"[Action fixUserAdminKubeconfig]",
		"[Action createOrUpdateRouterIPFromCluster]",
		"[Action ensureGatewayUpgrade]",
		"[Action rotateACRTokenPassword]",
		"[Action populateRegistryStorageAccountName]",
		"[Action ensureMTUSize]",
	}

	certificateRenewalSteps := []string{
		"[Action startVMs]",
		"[Condition apiServersReady, timeout 30m0s]",
		"[Action populateDatabaseIntIP]",
		"[Action fixMCSCert]",
		"[Action fixMCSUserData]",
		"[Action configureAPIServerCertificate]",
		"[Action configureIngressCertificate]",
		"[Action initializeOperatorDeployer]",
		"[Action renewMDSDCertificate]",
	}

	operatorUpdateSteps := []string{
		"[Action startVMs]",
		"[Condition apiServersReady, timeout 30m0s]",
		"[Action initializeOperatorDeployer]",
		"[Action ensureAROOperator]",
		"[Condition aroDeploymentReady, timeout 20m0s]",
		"[Condition ensureAROOperatorRunningDesiredVersion, timeout 5m0s]",
		"[Action syncClusterObject]",
	}

	syncClusterObjectSteps := []string{
		"[Action startVMs]",
		"[Condition apiServersReady, timeout 30m0s]",
		"[Action initializeOperatorDeployer]",
		"[Action syncClusterObject]",
	}

	hiveSteps := []string{
		"[Action hiveCreateNamespace]",
		"[Action hiveEnsureResources]",
		"[Condition hiveClusterDeploymentReady, timeout 5m0s]",
		"[Action hiveResetCorrelationData]",
	}

	updateProvisionedBySteps := []string{"[Action updateProvisionedBy]"}

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
			shouldRunSteps: utilgenerics.ConcatMultipleSlices(zerothSteps, operatorUpdateSteps),
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
			shouldRunSteps: utilgenerics.ConcatMultipleSlices(zerothSteps),
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
			shouldRunSteps: utilgenerics.ConcatMultipleSlices(zerothSteps, operatorUpdateSteps),
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
			shouldRunSteps: utilgenerics.ConcatMultipleSlices(
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
			shouldRunSteps: utilgenerics.ConcatMultipleSlices(
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
			shouldRunSteps: utilgenerics.ConcatMultipleSlices(
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
			shouldRunSteps: utilgenerics.ConcatMultipleSlices(zerothSteps, certificateRenewalSteps),
		},
		{
			name: "SyncClusterObject steps",
			fixture: func() (*api.OpenShiftClusterDocument, bool) {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskSyncClusterObject
				return doc, true
			},
			shouldRunSteps: utilgenerics.ConcatMultipleSlices(zerothSteps, syncClusterObjectSteps),
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
			shouldRunSteps: utilgenerics.ConcatMultipleSlices(
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
				o := strings.Replace(s.String(), "pkg/cluster.(*manager).", "", -1)
				stepsToRun = append(stepsToRun, o)
			}

			diff := deep.Equal(stepsToRun, tt.shouldRunSteps)
			for _, d := range diff {
				t.Error(d)
			}
		})
	}
}
