package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"github.com/Azure/ARO-RP/pkg/api/util/vms"
)

// OpenShiftClusterList represents a list of OpenShift clusters.
type OpenShiftClusterList struct {
	// The list of OpenShift clusters.
	OpenShiftClusters []*OpenShiftCluster `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

// OpenShiftCluster represents an Azure Red Hat OpenShift cluster.
type OpenShiftCluster struct {
	ID                         string                     `json:"id,omitempty" mutable:"case"`
	Name                       string                     `json:"name,omitempty" mutable:"case"`
	Type                       string                     `json:"type,omitempty" mutable:"case"`
	Location                   string                     `json:"location,omitempty"`
	Tags                       map[string]string          `json:"tags,omitempty"`
	Properties                 OpenShiftClusterProperties `json:"properties,omitempty"`
	Identity                   *ManagedServiceIdentity    `json:"identity,omitempty"`
	OperatorFlagsMergeStrategy string                     `json:"operatorFlagsMergeStrategy,omitempty" mutable:"true"`
}

// OpenShiftClusterProperties represents an OpenShift cluster's properties.
type OpenShiftClusterProperties struct {
	ArchitectureVersion             ArchitectureVersion              `json:"architectureVersion"` // ArchitectureVersion is int so 0 is valid value to be returned
	ProvisioningState               ProvisioningState                `json:"provisioningState,omitempty"`
	LastProvisioningState           ProvisioningState                `json:"lastProvisioningState,omitempty"`
	FailedProvisioningState         ProvisioningState                `json:"failedProvisioningState,omitempty"`
	LastAdminUpdateError            string                           `json:"lastAdminUpdateError,omitempty"`
	MaintenanceTask                 MaintenanceTask                  `json:"maintenanceTask,omitempty" mutable:"true"`
	OperatorFlags                   OperatorFlags                    `json:"operatorFlags,omitempty" mutable:"true"`
	OperatorVersion                 string                           `json:"operatorVersion,omitempty" mutable:"true"`
	CreatedAt                       time.Time                        `json:"createdAt,omitempty"`
	CreatedBy                       string                           `json:"createdBy,omitempty"`
	ProvisionedBy                   string                           `json:"provisionedBy,omitempty"`
	ClusterProfile                  ClusterProfile                   `json:"clusterProfile,omitempty"`
	FeatureProfile                  FeatureProfile                   `json:"featureProfile,omitempty"`
	ConsoleProfile                  ConsoleProfile                   `json:"consoleProfile,omitempty"`
	ServicePrincipalProfile         *ServicePrincipalProfile         `json:"servicePrincipalProfile,omitempty"`
	PlatformWorkloadIdentityProfile *PlatformWorkloadIdentityProfile `json:"platformWorkloadIdentityProfile,omitempty"`
	NetworkProfile                  NetworkProfile                   `json:"networkProfile,omitempty"`
	MasterProfile                   MasterProfile                    `json:"masterProfile,omitempty"`
	// WorkerProfiles is used to store the worker profile data that was sent in the api request
	WorkerProfiles []WorkerProfile `json:"workerProfiles,omitempty"`
	// WorkerProfilesStatus is used to store the enriched worker profile data
	WorkerProfilesStatus            []WorkerProfile   `json:"workerProfilesStatus,omitempty" swagger:"readOnly"`
	APIServerProfile                APIServerProfile  `json:"apiserverProfile,omitempty"`
	IngressProfiles                 []IngressProfile  `json:"ingressProfiles,omitempty"`
	Install                         *Install          `json:"install,omitempty"`
	StorageSuffix                   string            `json:"storageSuffix,omitempty"`
	RegistryProfiles                []RegistryProfile `json:"registryProfiles,omitempty"`
	ImageRegistryStorageAccountName string            `json:"imageRegistryStorageAccountName,omitempty"`
	InfraID                         string            `json:"infraId,omitempty"`
	HiveProfile                     HiveProfile       `json:"hiveProfile,omitempty"`
	MaintenanceState                MaintenanceState  `json:"maintenanceState,omitempty"`
}

// ProvisioningState represents a provisioning state.
type ProvisioningState string

// ProvisioningState constants
const (
	ProvisioningStateCreating      ProvisioningState = "Creating"
	ProvisioningStateUpdating      ProvisioningState = "Updating"
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

// MaintenanceState represents the maintenance state of a cluster.
// This is used by cluster monitornig stack to emit maintenance signals to customers.
type MaintenanceState string

const (
	MaintenanceStateNone                 MaintenanceState = "None"
	MaintenanceStatePending              MaintenanceState = "Pending"
	MaintenanceStatePlanned              MaintenanceState = "Planned"
	MaintenanceStateUnplanned            MaintenanceState = "Unplanned"
	MaintenanceStateCustomerActionNeeded MaintenanceState = "CustomerActionNeeded"
)

type MaintenanceTask string

const (
	//
	// Maintenance tasks that perform work on the cluster
	//

	MaintenanceTaskEverything          MaintenanceTask = "Everything"
	MaintenanceTaskOperator            MaintenanceTask = "OperatorUpdate"
	MaintenanceTaskRenewCerts          MaintenanceTask = "CertificatesRenewal"
	MaintenanceTaskSyncClusterObject   MaintenanceTask = "SyncClusterObject"
	MaintenanceTaskMigrateLoadBalancer MaintenanceTask = "MigrateLoadBalancer"

	//
	// Maintenance tasks for updating customer maintenance signals
	//

	MaintenanceTaskPending MaintenanceTask = "Pending"

	// None signal should only be used when (1) admin update fails and (2) SRE fixes the failed admin update without running another admin updates
	// Admin update success should automatically set the cluster into None state
	MaintenanceTaskNone MaintenanceTask = "None"

	// Customer action needed signal should only be used when (1) admin update fails and (2) customer needs to take action to resolve the failure
	// To remove the signal after customer takes action, use maintenance task None
	MaintenanceTaskCustomerActionNeeded MaintenanceTask = "CustomerActionNeeded"
)

var validMaintenanceTasks = []MaintenanceTask{
	MaintenanceTaskEverything,
	MaintenanceTaskOperator,
	MaintenanceTaskRenewCerts,
	MaintenanceTaskSyncClusterObject,
	MaintenanceTaskMigrateLoadBalancer,
	// internal maintenance state signals
	MaintenanceTaskPending,
	MaintenanceTaskNone,
	MaintenanceTaskCustomerActionNeeded,
}

// Operator feature flags
type OperatorFlags map[string]string

// ClusterProfile represents a cluster profile.
type ClusterProfile struct {
	Domain               string               `json:"domain,omitempty"`
	Version              string               `json:"version,omitempty"`
	ResourceGroupID      string               `json:"resourceGroupId,omitempty"`
	FipsValidatedModules FipsValidatedModules `json:"fipsValidatedModules,omitempty"`
	OIDCIssuer           *OIDCIssuer          `json:"oidcIssuer,omitempty"`
}

// FeatureProfile represents a feature profile.
type FeatureProfile struct {
	GatewayEnabled bool `json:"gatewayEnabled,omitempty" mutable:"true"`
}

// ConsoleProfile represents a console profile.
type ConsoleProfile struct {
	URL string `json:"url,omitempty"`
}

// ServicePrincipalProfile represents a service principal profile.
type ServicePrincipalProfile struct {
	ClientID     string `json:"clientId,omitempty"`
	SPObjectID   string `json:"spObjectId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
}

// SoftwareDefinedNetwork constants.
type SoftwareDefinedNetwork string

const (
	SoftwareDefinedNetworkOVNKubernetes SoftwareDefinedNetwork = "OVNKubernetes"
	SoftwareDefinedNetworkOpenShiftSDN  SoftwareDefinedNetwork = "OpenShiftSDN"
)

// MTUSize represents the MTU size of a cluster (Maximum transmission unit)
type MTUSize int

// MTUSize constants
const (
	MTU1500 MTUSize = 1500
	MTU3900 MTUSize = 3900
)

// The outbound routing strategy used to provide your cluster egress to the internet.
type OutboundType string

// OutboundType constants.
const (
	OutboundTypeUserDefinedRouting OutboundType = "UserDefinedRouting"
	OutboundTypeLoadbalancer       OutboundType = "Loadbalancer"
)

// ResourceReference represents a reference to an Azure resource.
type ResourceReference struct {
	// The fully qualified Azure resource id.
	ID string `json:"id,omitempty"`
}

// LoadBalancerProfile represents the profile of the cluster public load balancer.
type LoadBalancerProfile struct {
	// The desired managed outbound IPs for the cluster public load balancer.
	ManagedOutboundIPs *ManagedOutboundIPs `json:"managedOutboundIps,omitempty"`
	// The list of effective outbound IP addresses of the public load balancer.
	EffectiveOutboundIPs []EffectiveOutboundIP `json:"effectiveOutboundIps,omitempty" swagger:"readOnly"`
	// The desired outbound IP resources for the cluster load balancer.
	OutboundIPs []OutboundIP `json:"outboundIps,omitempty"`
	// The desired outbound IP Prefix resources for the cluster load balancer.
	OutboundIPPrefixes []OutboundIPPrefix `json:"outboundIpPrefixes,omitempty"`
	// The desired number of allocated SNAT ports per VM. Allowed values are in the range of 0 to 64000 (inclusive). The default value is 1024.
	AllocatedOutboundPorts *int `json:"allocatedOutboundPorts,omitempty"`
}

// EffectiveOutboundIP represents an effective outbound IP resource of the cluster public load balancer.
type EffectiveOutboundIP ResourceReference

// ManagedOutboundIPs represents the desired managed outbound IPs for the cluster public load balancer.
type ManagedOutboundIPs struct {
	// Count represents the desired number of IPv4 outbound IPs created and managed by Azure for the cluster public load balancer.  Allowed values are in the range of 1 - 20.  The default value is 1.
	Count int `json:"count,omitempty"`
}

// OutboundIP represents a desired outbound IP resource for the cluster load balancer.
type OutboundIP ResourceReference

// OutboundIPPrefix represents a desired outbound IP Prefix resource for the cluster load balancer.
type OutboundIPPrefix ResourceReference

// NetworkProfile represents a network profile.
type NetworkProfile struct {
	// The software defined network (SDN) to use when installing the cluster.
	SoftwareDefinedNetwork SoftwareDefinedNetwork `json:"softwareDefinedNetwork,omitempty"`

	PodCIDR      string       `json:"podCidr,omitempty"`
	ServiceCIDR  string       `json:"serviceCidr,omitempty"`
	MTUSize      MTUSize      `json:"mtuSize,omitempty"`
	OutboundType OutboundType `json:"outboundType,omitempty" mutable:"true"`

	APIServerPrivateEndpointIP string               `json:"privateEndpointIp,omitempty"`
	GatewayPrivateEndpointIP   string               `json:"gatewayPrivateEndpointIp,omitempty"`
	GatewayPrivateLinkID       string               `json:"gatewayPrivateLinkId,omitempty"`
	PreconfiguredNSG           PreconfiguredNSG     `json:"preconfiguredNSG,omitempty"`
	LoadBalancerProfile        *LoadBalancerProfile `json:"loadBalancerProfile,omitempty"`
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
	VMSize              vms.VMSize       `json:"vmSize,omitempty"`
	SubnetID            string           `json:"subnetId,omitempty"`
	EncryptionAtHost    EncryptionAtHost `json:"encryptionAtHost,omitempty"`
	DiskEncryptionSetID string           `json:"diskEncryptionSetId,omitempty"`
}

// WorkerProfile represents a worker profile.
type WorkerProfile struct {
	Name                string           `json:"name,omitempty"`
	VMSize              vms.VMSize       `json:"vmSize,omitempty"`
	DiskSizeGB          int              `json:"diskSizeGB,omitempty"`
	SubnetID            string           `json:"subnetId,omitempty"`
	Count               int              `json:"count,omitempty"`
	EncryptionAtHost    EncryptionAtHost `json:"encryptionAtHost,omitempty"`
	DiskEncryptionSetID string           `json:"diskEncryptionSetId,omitempty"`
}

// APIServerProfile represents an API server profile.
type APIServerProfile struct {
	Visibility Visibility `json:"visibility,omitempty"`
	URL        string     `json:"url,omitempty"`
	IP         string     `json:"ip,omitempty"`
	IntIP      string     `json:"intIp,omitempty"`
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
	Name       string     `json:"name,omitempty"`
	Visibility Visibility `json:"visibility,omitempty"`
	IP         string     `json:"ip,omitempty"`
}

// PlatformWorkloadIdentityProfile encapsulates all information that is specific to workload identity clusters.
type PlatformWorkloadIdentityProfile struct {
	UpgradeableTo              *UpgradeableTo                      `json:"upgradeableTo,omitempty"`
	PlatformWorkloadIdentities map[string]PlatformWorkloadIdentity `json:"platformWorkloadIdentities,omitempty"`
}

// UpgradeableTo stores a single OpenShift version a workload identity cluster can be upgraded to
type UpgradeableTo string

// PlatformWorkloadIdentity stores information representing a single workload identity.
type PlatformWorkloadIdentity struct {
	// The resource ID of the PlatformWorkloadIdentity resource
	ResourceID string `json:"resourceId,omitempty"`

	// The ClientID of the PlatformWorkloadIdentity resource
	ClientID string `json:"clientId,omitempty" swagger:"readOnly"`

	// The ObjectID of the PlatformWorkloadIdentity resource
	ObjectID string `json:"objectId,omitempty" swagger:"readOnly"`
}

// UserAssignedIdentity stores information about a user-assigned managed identity in a predefined format required by Microsoft's Managed Identity team.
type UserAssignedIdentity struct {
	// The ClientID of the ClusterUserAssignedIdentity resource
	ClientID string `json:"clientId,omitempty" swagger:"readOnly"`

	// The PrincipalID of the ClusterUserAssignedIdentity resource
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

// Install represents an install process.
type Install struct {
	Now   time.Time    `json:"now,omitempty"`
	Phase InstallPhase `json:"phase"`
}

// InstallPhase represents an install phase.
type InstallPhase int

// InstallPhase constants.
const (
	InstallPhaseBootstrap InstallPhase = iota
	InstallPhaseRemoveBootstrap
)

// RegistryProfile represents a registry profile
type RegistryProfile struct {
	Name     string `json:"name,omitempty"`
	Username string `json:"username,omitempty"`
	// IssueDate is when the username/password for the registry was last updated.
	IssueDate *time.Time `json:"issueDate,omitempty"`
}

// ArchitectureVersion represents an architecture version
type ArchitectureVersion int

// ArchitectureVersion constants
const (
	ArchitectureVersionV1 ArchitectureVersion = iota
	ArchitectureVersionV2
)

// CreatedByType defines user type, which executed the request
type CreatedByType string

const (
	CreatedByTypeApplication     CreatedByType = "Application"
	CreatedByTypeKey             CreatedByType = "Key"
	CreatedByTypeManagedIdentity CreatedByType = "ManagedIdentity"
	CreatedByTypeUser            CreatedByType = "User"
)

// SystemData metadata pertaining to creation and last modification of the resource.
type SystemData struct {
	CreatedBy          string        `json:"createdBy,omitempty"`
	CreatedByType      CreatedByType `json:"createdByType,omitempty"`
	CreatedAt          *time.Time    `json:"createdAt,omitempty"`
	LastModifiedBy     string        `json:"lastModifiedBy,omitempty"`
	LastModifiedByType CreatedByType `json:"lastModifiedByType,omitempty"`
	LastModifiedAt     *time.Time    `json:"lastModifiedAt,omitempty"`
}

type HiveProfile struct {
	Namespace string `json:"namespace,omitempty"`

	// CreatedByHive is used during PUCM to skip adoption and reconciliation
	// of clusters that were created by Hive to avoid deleting existing
	// ClusterDeployments.
	CreatedByHive bool `json:"createdByHive,omitempty"`
}
