import "@typespec/rest";
import "@typespec/http";
import "@azure-tools/typespec-azure-core";
import "@azure-tools/typespec-azure-resource-manager";

using TypeSpec.Rest;
using TypeSpec.Http;
using Azure.Core;
using Azure.ResourceManager;

@armProviderNamespace("Microsoft.RedHatOpenShift")
namespace Microsoft.RedHatOpenShift;

/** OpenShiftCluster represents an Azure Red Hat OpenShift cluster. */
model OpenShiftCluster is TrackedResource<OpenShiftClusterProperties> {
    ...ResourceNameParameter<OpenShiftCluster, KeyName = "resourceName", NamePattern = "">;
    ...ManagedServiceIdentityProperty;
}
@@doc(OpenShiftCluster.name, "The name of the OpenShift cluster resource.");

/** OpenShiftClusterProperties represents an OpenShift cluster's properties. */
model OpenShiftClusterProperties {
    /** The cluster provisioning state. */
    @visibility(Lifecycle.Read)
    provisioningState?: ProvisioningState;

    /** The cluster profile. */
    clusterProfile: ClusterProfile;

    /** The console profile. */
    consoleProfile: ConsoleProfile;

    /** The cluster service principal profile. */
    servicePrincipalProfile: ServicePrincipalProfile;

    /** The workload identity profile. */
    platformWorkloadIdentityProfile: PlatformWorkloadIdentityProfile;

    /** The cluster network profile. */
    networkProfile: NetworkProfile;

    /** The cluster master profile. */
    masterProfile: MasterProfile;

    /** The cluster worker profiles. */
    workerProfiles: WorkerProfile[];

    /** The cluster worker profiles status. */
    workerProfilesStatus: WorkerProfile[];

    /** The cluster API server profile. */
    apiserverProfile: ApiServerProfile;

    /** The cluster ingress profiles. */
    ingressProfiles: IngressProfile[];
}

/** ProvisioningState represents a provisioning state. */
@lroStatus
union ProvisioningState {
    ResourceProvisioningState,

    /** Non-terminal state indicating the resource is creating */
    "Creating",

    /** Non-terminal state indicating the resource is updating */
    "Updating",

    /** Non-terminal state indicating the resource is admin updating */
    "AdminUpdating",

    /** Non-terminal state indicating the resource is deleting */
    "Deleting",
}

/** ClusterProfile represents a cluster profile. */
model ClusterProfile {
    /** The pull secret for the cluster. */
    pullSecret: string;

    /** The domain for the cluster. */
    domain: string;

    /** The version of the cluster. */
    version: string;

    /** The ID of the cluster resource group. */
    resourceGroupId: string;

    /** If FIPS validated crypto modules are used */
    fipsValidatedModules: FipsValidatedModules;

    /** The URL of the managed OIDC issuer in a workload identity cluster */
    oidcIssuer: OIDCIssuer;
}

/** FIPSValidatedModules determines if FIPS is used. */
union FipsValidatedModules {
    string,
    /** FIPS validated modules are used */
    "Enabled",

    /** FIPS validated modules are not used */
    "Disabled"
}

scalar OIDCIssuer extends string {}

/** ConsoleProfile represents a console profile. */
model ConsoleProfile {
    /** The URL to access the cluster console. */
    url: string;
}

/** ServicePrincipalProfile represents a service principal profile. */
model ServicePrincipalProfile {
    /** The client ID used for the cluster. */
    clientId: string;

    /** The client secret used for the cluster. */
    clientSecret: string;
}

/** PlatformWorkloadIdentityProfile encapsulates all information that is specific to workload identity clusters. */
model PlatformWorkloadIdentityProfile {
    /** The cluster version that platform workload identities should be compatible with. */
    upgradeableTo: UpgradeableTo;

    /** The platform workload identities associated with the cluster. */
    #suppress "@azure-tools/typespec-azure-resource-manager/arm-no-record" "We require additionalProperties (map) because the keys (platformWorkloadIdentity) can change according to OpenShift version, and we don’t want to bind the key to a specific API version."
    platformWorkloadIdentities: Record<PlatformWorkloadIdentity>;
}

/** UpgradeableTo stores a single OpenShift version a workload identity cluster can be upgraded */
scalar UpgradeableTo extends string {}

/** PlatformWorkloadIdentity stores information representing a single workload identity. */
model PlatformWorkloadIdentity {
    /** The resource ID of the PlatformWorkloadIdentity resource */
    resourceId: string;

    /** The ClientID of the PlatformWorkloadIdentity resource */
    clientId: string;

    /** The ObjectID of the PlatformWorkloadIdentity resource */
    objectId: string;
}

/** NetworkProfile represents a network profile. */
model NetworkProfile {
    /** The CIDR used for OpenShift/Kubernetes Pods. */
    podCidr: string;

    /** The CIDR used for OpenShift/Kubernetes Services. */
    serviceCidr: string;

    /** The OutboundType used for egress traffic. */
    outboundType: OutboundType;

    /** The cluster load balancer profile. */
    loadBalancerProfile: LoadBalancerProfile;
    
    /** Specifies whether subnets are pre-attached with an NSG */
    #suppress "@azure-tools/typespec-azure-core/casing-style" "This field name is already live in the API and we cannot change it."
    preconfiguredNSG: PreconfiguredNSG;
}

/** The outbound routing strategy used to provide your cluster egress to the internet. */
union OutboundType {
    string,
    
    /** The cluster uses the standard Azure LoadBalancer for egress traffic. */
    "LoadBalancer",

    /** The cluster uses the UserDefinedRouting for egress traffic. */
    "UserDefinedRouting"
}

/** LoadBalancerProfile represents the profile of the cluster public load balancer. */
model LoadBalancerProfile {
    /** The desired managed outbound IPs for the cluster public load balancer. */
    managedOutboundIps: ManagedOutboundIps;

    /** The list of effective outbound IP addresses of the public load balancer. */
    effectiveOutboundIps: EffectiveOutboundIP[];
}

/** ManagedOutboundIPs represents the desired managed outbound IPs for the cluster public load balancer. */
model ManagedOutboundIps {
    /** Count represents the desired number of IPv4 outbound IPs created and managed by Azure for the cluster public load balancer.  Allowed values are in the range of 1 - 20.  The default value is 1. */
    count: int32;
}

/** EffectiveOutboundIP represents an effective outbound IP addresses of the cluster public load balancer. */
model EffectiveOutboundIP {
    /** The fully qualified Azure resource id of an IP address resource. */
    id: string;
}

/** PreconfiguredNSG represents whether customers want to use their own NSG attached to the subnets */
union PreconfiguredNSG {
    string,
    /** The cluster uses an existing NSG for egress traffic. */
    "Enabled",

    /** The cluster uses the service-provided NSG for egress traffic. */
    "Disabled"
}

/** MasterProfile represents a master profile. */
model MasterProfile {
    /** The size of the master VMs. */
    vmSize: VMSize;

    /** The Azure resource ID of the master subnet. */
    subnetId: string;

    /** Whether master virtual machines are encrypted at host. */
    encryptionAtHost: EncryptionAtHost;

    /** The resource ID of an associated DiskEncryptionSet, if applicable. */
    diskEncryptionSetId: string;
}

/**
 * VM size availability varies by region.
 * If a node contains insufficient compute resources (memory, cpu, etc.), pods might fail to run correctly.
 * For more details on restricted VM sizes, see: https://docs.microsoft.com/en-us/azure/openshift/support-policies-v4#supported-virtual-machine-sizes
 */
scalar VMSize extends string {}

/** EncryptionAtHost represents encryption at host state */
union EncryptionAtHost {
    string,

    /** Encryption at host is enabled */
    "Enabled",

    /** Encryption at host is disabled */
    "Disabled"
}

/** WorkerProfile represents a worker profile. */
model WorkerProfile {
    /** The worker profile name. */
    name: string;

    /** The size of the worker VMs. */
    vmSize: VMSize;

    /** The disk size of the worker VMs. */
    #suppress "@azure-tools/typespec-azure-core/casing-style" "This field name is already live in the API and we cannot change it."
    diskSizeGB: int32;

    /** The Azure resource ID of the worker subnet. */
    subnetId: string;

    /** The number of worker VMs. */
    count: int32;

    /** Whether worker virtual machines are encrypted at host. */
    encryptionAtHost: EncryptionAtHost;

    /** The resource ID of an associated DiskEncryptionSet, if applicable. */
    diskEncryptionSetId: string;
}

/** APIServerProfile represents an API server profile. */
model ApiServerProfile {
    /** API server visibility. */
    visibility: Visibility;

    /** The URL to access the cluster API server. */
    url: string;

    /** The IP of the cluster API server. */
    ip: string;
}

/** IngressProfile represents an ingress profile. */
model IngressProfile {
    /** The ingress profile name. */
    name: string;

    /** Ingress visibility. */
    visibility: Visibility;

    /** The IP of the cluster ingress. */
    ip: string;
}

/** Visibility represents visibility. */
union Visibility {
    string,

    /** The ingress or apiserver is public. */
    "Public",

    /** The ingress or apiserver is private. */
    "Private"
}

/** OpenShiftClusterCredentials represents an OpenShift cluster's credentials. */
model OpenShiftClusterCredentials {
    /** The username for the kubeadmin user. */
    kubeadminUsername: string;

    /** The password for the kubeadmin user. */
    @secret
    kubeadminPassword: string;
}

/** OpenShiftClusterAdminKubeconfig represents an OpenShift cluster's admin kubeconfig. */
model OpenShiftClusterAdminKubeconfig {
    /** The base64-encoded kubeconfig file. */
    @secret
    kubeconfig: string;
}
