package rolesets

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

var DefaultPlatformWorkloadIdentityRoleSet = api.PlatformWorkloadIdentityRoleSet{
	Properties: api.PlatformWorkloadIdentityRoleSetProperties{
		OpenShiftVersion: "4.14",
		PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
			{
				OperatorName:       "CloudControllerManager",
				RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
				RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
				ServiceAccounts: []string{
					"openshift-cloud-controller-manager:cloud-controller-manager",
				},
			},
			{
				OperatorName:       "ClusterIngressOperator",
				RoleDefinitionName: "Azure RedHat OpenShift Cluster Ingress Operator Role",
				RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c",
				ServiceAccounts: []string{
					"openshift-ingress-operator:ingress-operator",
				},
			},
			{
				OperatorName:       "MachineApiOperator",
				RoleDefinitionName: "Azure RedHat OpenShift Machine API Operator Role",
				RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/0358943c-7e01-48ba-8889-02cc51d78637",
				ServiceAccounts: []string{
					"openshift-machine-api:machine-api-operator",
				},
			},
			{
				OperatorName:       "StorageOperator",
				RoleDefinitionName: "Azure RedHat OpenShift Storage Operator Role",
				RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/5b7237c5-45e1-49d6-bc18-a1f62f400748",
				ServiceAccounts: []string{
					"openshift-cluster-csi-drivers:azure-disk-csi-driver-operator",
					"openshift-cluster-csi-drivers:azure-disk-csi-driver-controller-sa",
				},
			},
			{
				OperatorName:       "NetworkOperator",
				RoleDefinitionName: "Azure RedHat OpenShift Network Operator Role",
				RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/be7a6435-15ae-4171-8f30-4a343eff9e8f",
				ServiceAccounts: []string{
					"openshift-cloud-network-config-controller:cloud-network-config-controller",
				},
			},
			{
				OperatorName:       "ImageRegistryOperator",
				RoleDefinitionName: "Azure RedHat OpenShift Image Registry Operator Role",
				RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/8b32b316-c2f5-4ddf-b05b-83dacd2d08b5",
				ServiceAccounts: []string{
					"openshift-image-registry:cluster-image-registry-operator",
					"openshift-image-registry:registry",
				},
			},
			{
				OperatorName:       "AzureFilesStorageOperator",
				RoleDefinitionName: "Azure RedHat OpenShift Azure Files Storage Operator Role",
				RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/0d7aedc0-15fd-4a67-a412-efad370c947e",
				ServiceAccounts: []string{
					"openshift-cluster-csi-drivers:azure-file-csi-driver-operator",
					"openshift-cluster-csi-drivers:azure-file-csi-driver-controller-sa",
					"openshift-cluster-csi-drivers:azure-file-csi-driver-node-sa",
				},
			},
			{
				OperatorName:       "ServiceOperator",
				RoleDefinitionName: "Azure RedHat OpenShift Service Operator",
				RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/4436bae4-7702-4c84-919b-c4069ff25ee2",
				ServiceAccounts: []string{
					"openshift-azure-operator:aro-operator-master",
				},
			},
		},
	},
}
