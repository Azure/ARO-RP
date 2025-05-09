import "@typespec/versioning";
import "@azure-tools/typespec-azure-resource-manager";

import "./models/openShiftCluster.tsp";
import "./models/openShiftVersion.tsp";
import "./models/platformWorkloadIdentityRoleSet.tsp";

using TypeSpec.Http;
using TypeSpec.Versioning;
using Azure.ResourceManager;

@armProviderNamespace("Microsoft.RedHatOpenShift")
@service(#{ title: "Azure Red Hat OpenShift" })
@versioned(Microsoft.RedHatOpenShift.Versions)
namespace Microsoft.RedHatOpenShift;

@summary("Azure Red Hat OpenShift")
interface Operations extends Azure.ResourceManager.Operations {}

enum Versions {
  // v2020_04_30: "2020-04-30",
  // v2021_09_01_preview: "2021-09-01-preview",
  // v2022_04_01: "2022-04-01",
  // v2022_09_04: "2022-09-04",
  // v2023_04_01: "2023-04-01",
  // v2023_07_01_preview: "2023-07-01-preview",
  // v2023_09_04: "2023-09-04",
  // v2023_11_22: "2023-11-22",
  // v2024_08_12_preview: "2024-08-12-preview",

  @useDependency(Azure.ResourceManager.Versions.v1_0_Preview_1)
  @useDependency(Azure.Core.Versions.v1_0_Preview_2)
  @armCommonTypesVersion(Azure.ResourceManager.CommonTypes.Versions.v6)
  v2025_07_25: "2025-07-25",
}

@armResourceOperations(OpenShiftCluster)
interface OpenShiftClusters {
    @summary("Lists OpenShift clusters in the specified subscription.")
    @doc("The operation returns properties of each OpenShift cluster.")
    listBySubscription is ArmListBySubscription<OpenShiftCluster>;

    @summary("Lists OpenShift clusters in the specified subscription and resource group.")
    @doc("The operation returns properties of each OpenShift cluster.")
    listByResourceGroup is ArmResourceListByParent<OpenShiftCluster>;

    @summary("Gets a OpenShift cluster with the specified subscription, resource group and resource name.")
    @doc("The operation returns properties of a OpenShift cluster.")
    get is ArmResourceRead<OpenShiftCluster>;


    @summary("Creates or updates a OpenShift cluster with the specified subscription, resource group and resource name.")
    @doc("The operation returns properties of a OpenShift cluster.")
    createOrUpdate is ArmResourceCreateOrReplaceAsync<OpenShiftCluster>;

    @summary("Creates or updates a OpenShift cluster with the specified subscription, resource group and resource name.")
    @doc("The operation returns properties of a OpenShift cluster.")
    update is ArmResourcePatchAsync<OpenShiftCluster, OpenShiftClusterProperties>;

    @summary("Deletes a OpenShift cluster with the specified subscription, resource group and resource name.")
    @doc("The operation returns nothing.")
    delete is ArmResourceDeleteWithoutOkAsync<OpenShiftCluster>;

    @summary("Lists credentials of an OpenShift cluster with the specified subscription, resource group and resource name.")
    @doc("The operation returns the credentials.")
    listCredentials is ArmResourceActionSync<OpenShiftCluster, void, OpenShiftClusterCredentials>;

    @summary("Lists admin kubeconfig of an OpenShift cluster with the specified subscription, resource group and resource name.")
    @doc("The operation returns the admin kubeconfig.")
    listAdminCredentials is ArmResourceActionSync<OpenShiftCluster, void, OpenShiftClusterAdminKubeconfig>;
}

@armResourceOperations(OpenShiftVersion)
interface OpenShiftVersions {
    @summary("Lists all OpenShift versions available to install in the specified location.")
    @doc("The operation returns the installable OpenShift versions as a string.")
    list is ArmListBySubscription<OpenShiftVersion>;


    @summary("Gets an available OpenShift version to install in the specified location.")
    @doc("This operation returns installable OpenShift version as a string.")
    get is ArmResourceRead<OpenShiftVersion>;
}

@armResourceOperations(PlatformWorkloadIdentityRoleSet)
interface PlatformWorkloadIdentityRoleSets {
    @summary("Lists a mapping of OpenShift versions to identity requirements, which include operatorName, roleDefinitionName, roleDefinitionId, and serviceAccounts.")
    @doc("This operation returns a list of Platform Workload Identity Role Sets as a string")
    list is ArmListBySubscription<PlatformWorkloadIdentityRoleSet>;


    @summary("Gets a mapping of an OpenShift version to identity requirements, which includes operatorName, roleDefinitionName, roleDefinitionId, and serviceAccounts.")
    @doc("This operation returns Platform Workload Identity Role Set as a string")
    get is ArmResourceRead<PlatformWorkloadIdentityRoleSet>;
}
