import "@typespec/rest";
import "@typespec/http";
import "@azure-tools/typespec-azure-core";
import "@azure-tools/typespec-azure-resource-manager";

using TypeSpec.Rest;
using TypeSpec.Http;
using Azure.ResourceManager;

@armProviderNamespace("Microsoft.RedHatOpenShift")
namespace Microsoft.RedHatOpenShift;

@parentResource(ArmLocationResource<"Subscription">)
model PlatformWorkloadIdentityRoleSet is ProxyResource<PlatformWorkloadIdentityRoleSetProperties> {
  ...ResourceNameParameter<PlatformWorkloadIdentityRoleSet, KeyName = "openShiftMinorVersion", NamePattern = "^(\\d+)\\.(\\d+)">;
}
@@doc(PlatformWorkloadIdentityRoleSet.name, "The desired version value of the PlatformWorkloadIdentityRoleSet resource.");
@@minLength(PlatformWorkloadIdentityRoleSet.name, 1);
@@maxLength(PlatformWorkloadIdentityRoleSet.name, 63);

/** PlatformWorkloadIdentityRoleSetProperties represents the properties of an PlatformWorkloadIdentityRoleSet. */
#suppress "@azure-tools/typespec-azure-resource-manager/arm-resource-provisioning-state" "Proxy resource does not have provisioning state"
model PlatformWorkloadIdentityRoleSetProperties {
    /** OpenShiftVersion represents the version associated with this set of roles. */
    openShiftVersion: string;

    /** PlatformWorkloadIdentityRoles represents the set of roles associated with this version. */
    @OpenAPI.extension("x-ms-identifiers", #[])
    platformWorkloadIdentityRoles: PlatformWorkloadIdentityRole[];
}

/** PlatformWorkloadIdentityRole represents a mapping from a particular OCP operator to the built-in role that should be assigned to that operator's corresponding managed identity. */
model PlatformWorkloadIdentityRole {
    /** OperatorName represents the name of the operator that this role is for. */
    operatorName: string;

    /** RoleDefinitionName represents the name of the role. */
    roleDefinitionName: string;

    /** RoleDefinitionID represents the resource ID of the role definition. */
    roleDefinitionId: string;
}
