package v20240812preview

import "time"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftClusterList represents a list of OpenShift clusters.
type OpenShiftClusterList struct {
	// The list of OpenShift clusters.
	OpenShiftClusters []*OpenShiftCluster `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

// OpenShiftCluster represents an Azure Red Hat OpenShift cluster.
type OpenShiftCluster struct {
	// The resource ID.
	ID string `json:"id,omitempty" mutable:"case"`

	// The resource name.
	Name string `json:"name,omitempty" mutable:"case"`

	// The resource type.
	Type string `json:"type,omitempty" mutable:"case"`

	// The resource location.
	Location string `json:"location,omitempty"`

	// SystemData - The system metadata relating to this resource
	SystemData *SystemData `json:"systemData,omitempty" swagger:"readOnly"`

	// The resource tags.
	Tags Tags `json:"tags,omitempty" mutable:"true"`

	// The cluster properties.
	Properties OpenShiftClusterProperties `json:"properties,omitempty"`

	// Identity stores information about the cluster MSI(s) in a workload identity cluster.
	Identity *ManagedServiceIdentity `json:"identity,omitempty"`
}

// UsesWorkloadIdentity checks whether a cluster is a Workload Identity cluster or a Service Principal cluster
func (oc *OpenShiftCluster) UsesWorkloadIdentity() bool {
	return oc.Properties.PlatformWorkloadIdentityProfile != nil && oc.Properties.ServicePrincipalProfile == nil
}

// Tags represents an OpenShift cluster's tags.
type Tags map[string]string

// OpenShiftClusterProperties represents an OpenShift cluster's properties.
type OpenShiftClusterProperties struct {
	// The cluster provisioning state.
	ProvisioningState ProvisioningState `json:"provisioningState,omitempty"`

	// The cluster profile.
	ClusterProfile ClusterProfile `json:"clusterProfile,omitempty"`

	// The console profile.
	ConsoleProfile ConsoleProfile `json:"consoleProfile,omitempty"`

	// The cluster service principal profile.
	ServicePrincipalProfile *ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`

	// The workload identity profile.
	PlatformWorkloadIdentityProfile *PlatformWorkloadIdentityProfile `json:"platformWorkloadIdentityProfile,omitempty"`

	// The cluster network profile.
	NetworkProfile NetworkProfile `json:"networkProfile,omitempty"`

	// The cluster master profile.
	MasterProfile MasterProfile `json:"masterProfile,omitempty"`

	// The cluster worker profiles.
	WorkerProfiles []WorkerProfile `json:"workerProfiles,omitempty"`

	// The cluster worker profiles status.
	WorkerProfilesStatus []WorkerProfile `json:"workerProfilesStatus,omitempty" swagger:"readOnly"`

	// The cluster API server profile.
	APIServerProfile APIServerProfile `json:"apiserverProfile,omitempty"`

	// The cluster ingress profiles.
	IngressProfiles []IngressProfile `json:"ingressProfiles,omitempty"`
}

// ProvisioningState represents a provisioning state.
type ProvisioningState string

// ProvisioningState constants.
// TODO: ProvisioningStateCanceled is included to pass upstream CI. It is currently unused in ARO.
const (
	ProvisioningStateCreating      ProvisioningState = "Creating"
	ProvisioningStateUpdating      ProvisioningState = "Updating"
	ProvisioningStateCanceled      ProvisioningState = "Canceled"
	ProvisioningStateAdminUpdating ProvisioningState = "AdminUpdating"
	ProvisioningStateDeleting      ProvisioningState = "Deleting"
	ProvisioningStateSucceeded     ProvisioningState = "Succeeded"
	ProvisioningStateFailed        ProvisioningState = "Failed"
)

// FipsValidatedModules determines if FIPS is used.
type FipsValidatedModules string

// OIDCIssuer represents the URL of the managed OIDC issuer in a workload identity cluster.
type OIDCIssuer string

// FipsValidatedModules constants.
const (
	FipsValidatedModulesEnabled  FipsValidatedModules = "Enabled"
	FipsValidatedModulesDisabled FipsValidatedModules = "Disabled"
)

// ClusterProfile represents a cluster profile.
type ClusterProfile struct {
	// The pull secret for the cluster.
	PullSecret string `json:"pullSecret,omitempty"`

	// The domain for the cluster.
	Domain string `json:"domain,omitempty"`

	// The version of the cluster.
	Version string `json:"version,omitempty"`

	// The ID of the cluster resource group.
	ResourceGroupID string `json:"resourceGroupId,omitempty"`

	// If FIPS validated crypto modules are used
	FipsValidatedModules FipsValidatedModules `json:"fipsValidatedModules,omitempty"`

	// The URL of the managed OIDC issuer in a workload identity cluster.
	OIDCIssuer *OIDCIssuer `json:"oidcIssuer,omitempty"`
}

// ConsoleProfile represents a console profile.
type ConsoleProfile struct {
	// The URL to access the cluster console.
	URL string `json:"url,omitempty" swagger:"readOnly"`
}

// ServicePrincipalProfile represents a service principal profile.
type ServicePrincipalProfile struct {
	// The client ID used for the cluster.
	ClientID string `json:"clientId,omitempty" mutable:"true"`

	// The client secret used for the cluster.
	ClientSecret string `json:"clientSecret,omitempty" mutable:"true"`
}

// The outbound routing strategy used to provide your cluster egress to the internet.
type OutboundType string

// OutboundType constants.
const (
	OutboundTypeUserDefinedRouting OutboundType = "UserDefinedRouting"
	OutboundTypeLoadbalancer       OutboundType = "Loadbalancer"
)

// ResourceReference represents a reference to an Azure resource.
type ResourceReference struct {
	// The fully qualified Azure resource id of an IP address resource.
	ID string `json:"id,omitempty"`
}

// LoadBalancerProfile represents the profile of the cluster public load balancer.
type LoadBalancerProfile struct {
	// The desired managed outbound IPs for the cluster public load balancer.
	ManagedOutboundIPs *ManagedOutboundIPs `json:"managedOutboundIps,omitempty" mutable:"true"`
	// The list of effective outbound IP addresses of the public load balancer.
	EffectiveOutboundIPs []EffectiveOutboundIP `json:"effectiveOutboundIps,omitempty" swagger:"readOnly"`
}

// EffectiveOutboundIP represents an effective outbound IP resource of the cluster public load balancer.
type EffectiveOutboundIP ResourceReference

// ManagedOutboundIPs represents the desired managed outbound IPs for the cluster public load balancer.
type ManagedOutboundIPs struct {
	// Count represents the desired number of IPv4 outbound IPs created and managed by Azure for the cluster public load balancer.  Allowed values are in the range of 1 - 20.  The default value is 1.
	Count int `json:"count,omitempty"`
}

// NetworkProfile represents a network profile.
type NetworkProfile struct {
	// The CIDR used for OpenShift/Kubernetes Pods.
	PodCIDR string `json:"podCidr,omitempty"`

	// The CIDR used for OpenShift/Kubernetes Services.
	ServiceCIDR string `json:"serviceCidr,omitempty"`

	// The OutboundType used for egress traffic.
	OutboundType OutboundType `json:"outboundType,omitempty"`

	// The cluster load balancer profile.
	LoadBalancerProfile *LoadBalancerProfile `json:"loadBalancerProfile,omitempty"`

	// Specifies whether subnets are pre-attached with an NSG
	PreconfiguredNSG PreconfiguredNSG `json:"preconfiguredNSG,omitempty"`
}

// PreconfiguredNSG represents whether customers want to use their own NSG attached to the subnets
type PreconfiguredNSG string

// PreconfiguredNSG constants
const (
	PreconfiguredNSGEnabled  PreconfiguredNSG = "Enabled"
	PreconfiguredNSGDisabled PreconfiguredNSG = "Disabled"
)

// EncryptionAtHost represents encryption at host state
type EncryptionAtHost string

// EncryptionAtHost constants
const (
	EncryptionAtHostEnabled  EncryptionAtHost = "Enabled"
	EncryptionAtHostDisabled EncryptionAtHost = "Disabled"
)

// MasterProfile represents a master profile.
type MasterProfile struct {
	// The size of the master VMs.
	VMSize VMSize `json:"vmSize,omitempty"`

	// The Azure resource ID of the master subnet.
	SubnetID string `json:"subnetId,omitempty"`

	// Whether master virtual machines are encrypted at host.
	EncryptionAtHost EncryptionAtHost `json:"encryptionAtHost,omitempty"`

	// The resource ID of an associated DiskEncryptionSet, if applicable.
	DiskEncryptionSetID string `json:"diskEncryptionSetId,omitempty"`
}

// VM size availability varies by region.
// If a node contains insufficient compute resources (memory, cpu, etc.), pods might fail to run correctly.
// For more details on restricted VM sizes, see: https://docs.microsoft.com/en-us/azure/openshift/support-policies-v4#supported-virtual-machine-sizes
type VMSize string

// WorkerProfile represents a worker profile.
type WorkerProfile struct {
	// The worker profile name.
	Name string `json:"name,omitempty"`

	// The size of the worker VMs.
	VMSize VMSize `json:"vmSize,omitempty"`

	// The disk size of the worker VMs.
	DiskSizeGB int `json:"diskSizeGB,omitempty"`

	// The Azure resource ID of the worker subnet.
	SubnetID string `json:"subnetId,omitempty"`

	// The number of worker VMs.
	Count int `json:"count,omitempty"`

	// Whether master virtual machines are encrypted at host.
	EncryptionAtHost EncryptionAtHost `json:"encryptionAtHost,omitempty"`

	// The resource ID of an associated DiskEncryptionSet, if applicable.
	DiskEncryptionSetID string `json:"diskEncryptionSetId,omitempty"`
}

// APIServerProfile represents an API server profile.
type APIServerProfile struct {
	// API server visibility.
	Visibility Visibility `json:"visibility,omitempty"`

	// The URL to access the cluster API server.
	URL string `json:"url,omitempty" swagger:"readOnly"`

	// The IP of the cluster API server.
	IP string `json:"ip,omitempty" swagger:"readOnly"`
}

// Visibility represents visibility.
type Visibility string

// Visibility constants
const (
	VisibilityPublic  Visibility = "Public"
	VisibilityPrivate Visibility = "Private"
)

// IngressProfile represents an ingress profile.
type IngressProfile struct {
	// The ingress profile name.
	Name string `json:"name,omitempty"`

	// Ingress visibility.
	Visibility Visibility `json:"visibility,omitempty"`

	// The IP of the ingress.
	IP string `json:"ip,omitempty" swagger:"readOnly"`
}

// PlatformWorkloadIdentityProfile encapsulates all information that is specific to workload identity clusters.
type PlatformWorkloadIdentityProfile struct {
	UpgradeableTo              *UpgradeableTo                      `json:"upgradeableTo,omitempty" mutable:"true"`
	PlatformWorkloadIdentities map[string]PlatformWorkloadIdentity `json:"platformWorkloadIdentities,omitempty" mutable:"true"`
}

// UpgradeableTo stores a single OpenShift version a workload identity cluster can be upgraded to
type UpgradeableTo string

// PlatformWorkloadIdentity stores information representing a single workload identity.
type PlatformWorkloadIdentity struct {
	// The resource ID of the PlatformWorkloadIdentity resource
	ResourceID string `json:"resourceId,omitempty" mutable:"true"`

	// The ClientID of the PlatformWorkloadIdentity resource
	ClientID string `json:"clientId,omitempty" swagger:"readOnly" mutable:"true"`

	// The ObjectID of the PlatformWorkloadIdentity resource
	ObjectID string `json:"objectId,omitempty" swagger:"readOnly" mutable:"true"`
}

// UserAssignedIdentity stores information about a user-assigned managed identity in a predefined format required by Microsoft's Managed Identity team.
type UserAssignedIdentity struct {
	// The ClientID of the UserAssignedIdentity resource
	ClientID string `json:"clientId,omitempty" swagger:"readOnly"`

	// The PrincipalID of the UserAssignedIdentity resource
	PrincipalID string `json:"principalId,omitempty" swagger:"readOnly"`
}

// The ManagedServiceIdentity type.
type ManagedServiceIdentityType string

// ManagedServiceIdentityType constants
const (
	ManagedServiceIdentityNone                       ManagedServiceIdentityType = "None"
	ManagedServiceIdentitySystemAssigned             ManagedServiceIdentityType = "SystemAssigned"
	ManagedServiceIdentityUserAssigned               ManagedServiceIdentityType = "UserAssigned"
	ManagedServiceIdentitySystemAssignedUserAssigned ManagedServiceIdentityType = "SystemAssigned,UserAssigned"
)

// ManagedServiceIdentity stores information about the cluster MSI(s) in a workload identity cluster.
type ManagedServiceIdentity struct {
	// The type of the ManagedServiceIdentity resource.
	Type ManagedServiceIdentityType `json:"type,omitempty"`

	// The PrincipalID of the Identity resource.
	PrincipalID string `json:"principalId,omitempty" swagger:"readOnly"`

	// The TenantID provided by the MSI RP
	TenantID string `json:"tenantId,omitempty" swagger:"readOnly"`

	// A map of user assigned identities attached to the cluster, specified in a type required by Microsoft's Managed Identity team.
	UserAssignedIdentities map[string]UserAssignedIdentity `json:"userAssignedIdentities,omitempty"`
}

// CreatedByType by defines user type, which executed the request
type CreatedByType string

const (
	CreatedByTypeApplication     CreatedByType = "Application"
	CreatedByTypeKey             CreatedByType = "Key"
	CreatedByTypeManagedIdentity CreatedByType = "ManagedIdentity"
	CreatedByTypeUser            CreatedByType = "User"
)

// SystemData metadata pertaining to creation and last modification of the resource.
type SystemData struct {
	// The identity that created the resource.
	CreatedBy string `json:"createdBy,omitempty"`
	// The type of identity that created the resource. Possible values include: 'User', 'Application', 'ManagedIdentity', 'Key'
	CreatedByType CreatedByType `json:"createdByType,omitempty"`
	// The timestamp of resource creation (UTC).
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	// The identity that last modified the resource.
	LastModifiedBy string `json:"lastModifiedBy,omitempty"`
	// The type of identity that last modified the resource. Possible values include: 'User', 'Application', 'ManagedIdentity', 'Key'
	LastModifiedByType CreatedByType `json:"lastModifiedByType,omitempty"`
	// The type of identity that last modified the resource.
	LastModifiedAt *time.Time `json:"lastModifiedAt,omitempty"`
}
