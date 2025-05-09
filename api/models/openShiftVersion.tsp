import "@typespec/rest";
import "@typespec/http";
import "@azure-tools/typespec-azure-core";
import "@azure-tools/typespec-azure-resource-manager";

using TypeSpec.Rest;
using Azure.ResourceManager;

@armProviderNamespace("Microsoft.RedHatOpenShift")
namespace Microsoft.RedHatOpenShift;

@parentResource(ArmLocationResource<"Subscription">)
model OpenShiftVersion is ProxyResource<OpenShiftVersionProperties> {
  ...ResourceNameParameter<OpenShiftVersion, KeyName = "openShiftVersion", NamePattern = "^(\\d+)\\.(\\d+)\\.(\\d+)(.*)">;
}
@@doc(OpenShiftVersion.name, "The desired version value of the OpenShiftVersion resource.");
@@minLength(OpenShiftVersion.name, 1);
@@maxLength(OpenShiftVersion.name, 63);

/** OpenShiftVersionProperties represents the properties of an OpenShiftVersion. */
#suppress "@azure-tools/typespec-azure-resource-manager/arm-resource-provisioning-state" "Proxy resource does not have provisioning state"
model OpenShiftVersionProperties {
    /** Version represents the version to create the cluster at. */
    version: string;
}
