package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"

	"github.com/go-test/deep"

	"github.com/Azure/ARO-RP/pkg/api"
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
			},
		}
	}

	for _, tt := range []struct {
		name           string
		fixture        func() *api.OpenShiftClusterDocument
		shouldRunSteps []string
	}{
		{
			name: "ARO Operator Update",
			fixture: func() *api.OpenShiftClusterDocument {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskOperator
				return doc
			},
			shouldRunSteps: []string{
				"[Action initializeKubernetesClients-fm]",
				"[Action ensureBillingRecord-fm]",
				"[Action ensureDefaults-fm]",
				"[Action fixupClusterSPObjectID-fm]",
				"[Action fixInfraID-fm]",
				"[Action startVMs-fm]",
				"[Condition apiServersReady-fm, timeout 30m0s]",
				"[Action ensureAROOperator-fm]",
				"[Condition aroDeploymentReady-fm, timeout 20m0s]",
			},
		},
		{
			name: "Everything update",
			fixture: func() *api.OpenShiftClusterDocument {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				return doc
			},
			shouldRunSteps: []string{
				"[Action initializeKubernetesClients-fm]",
				"[Action ensureBillingRecord-fm]",
				"[Action ensureDefaults-fm]",
				"[Action fixupClusterSPObjectID-fm]",
				"[Action fixInfraID-fm]",
				"[AuthorizationRefreshingAction [Action ensureResourceGroup-fm]]",
				"[Action createOrUpdateDenyAssignment-fm]",
				"[Action fixSSH-fm]",
				"[Action populateDatabaseIntIP-fm]",
				"[Action startVMs-fm]",
				"[Condition apiServersReady-fm, timeout 30m0s]",
				"[Action fixSREKubeconfig-fm]",
				"[Action fixUserAdminKubeconfig-fm]",
				"[Action createOrUpdateRouterIPFromCluster-fm]",
				"[Action fixMCSCert-fm]",
				"[Action fixMCSUserData-fm]",
				"[Action ensureGatewayUpgrade-fm]",
				"[Action configureAPIServerCertificate-fm]",
				"[Action configureIngressCertificate-fm]",
				"[Action populateRegistryStorageAccountName-fm]",
				"[Action populateCreatedAt-fm]",
				"[Action ensureAROOperator-fm]",
				"[Condition aroDeploymentReady-fm, timeout 20m0s]",
				"[Action updateProvisionedBy-fm]",
			},
		},
		{
			name: "Blank (should perform everything)",
			fixture: func() *api.OpenShiftClusterDocument {
				doc := baseClusterDoc()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				return doc
			},
			shouldRunSteps: []string{
				"[Action initializeKubernetesClients-fm]",
				"[Action ensureBillingRecord-fm]",
				"[Action ensureDefaults-fm]",
				"[Action fixupClusterSPObjectID-fm]",
				"[Action fixInfraID-fm]",
				"[AuthorizationRefreshingAction [Action ensureResourceGroup-fm]]",
				"[Action createOrUpdateDenyAssignment-fm]",
				"[Action fixSSH-fm]",
				"[Action populateDatabaseIntIP-fm]",
				"[Action startVMs-fm]",
				"[Condition apiServersReady-fm, timeout 30m0s]",
				"[Action fixSREKubeconfig-fm]",
				"[Action fixUserAdminKubeconfig-fm]",
				"[Action createOrUpdateRouterIPFromCluster-fm]",
				"[Action fixMCSCert-fm]",
				"[Action fixMCSUserData-fm]",
				"[Action ensureGatewayUpgrade-fm]",
				"[Action configureAPIServerCertificate-fm]",
				"[Action configureIngressCertificate-fm]",
				"[Action populateRegistryStorageAccountName-fm]",
				"[Action populateCreatedAt-fm]",
				"[Action ensureAROOperator-fm]",
				"[Condition aroDeploymentReady-fm, timeout 20m0s]",
				"[Action updateProvisionedBy-fm]",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				doc: tt.fixture(),
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
