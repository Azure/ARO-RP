package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"sync"
	"time"

	"github.com/Azure/ARO-RP/pkg/util/vms"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
)

// OpenShiftCluster represents an OpenShift cluster
type OpenShiftCluster struct {
	MissingFields

	// ID, Name and Type are cased as the user provided them at create time.
	// ID, Name, Type and Location are immutable.
	ID         string                     `json:"id,omitempty"`
	Name       string                     `json:"name,omitempty"`
	Type       string                     `json:"type,omitempty"`
	Location   string                     `json:"location,omitempty"`
	SystemData SystemData                 `json:"systemData,omitempty"`
	Tags       map[string]string          `json:"tags,omitempty"`
	Properties OpenShiftClusterProperties `json:"properties,omitempty"`
	Identity   *ManagedServiceIdentity    `json:"managedServiceIdentity,omitempty"`

	// this property is used in the enrichers. Should not be marshalled
	Lock sync.Mutex `json:"-"`
}

// UsesWorkloadIdentity checks whether a cluster is a Workload Identity cluster or a Service Principal cluster
func (oc *OpenShiftCluster) UsesWorkloadIdentity() bool {
	return oc.Properties.PlatformWorkloadIdentityProfile != nil && oc.Properties.ServicePrincipalProfile == nil
}

// ClusterMsiResourceId returns the resource ID of the cluster MSI or an error
// if it encounters an issue while grabbing the resource ID from the cluster
// doc. It is written under the assumption that there is only one cluster MSI
// and will have to be refactored if we ever use more than one.
func (oc *OpenShiftCluster) ClusterMsiResourceId() (*arm.ResourceID, error) {
	if !oc.HasUserAssignedIdentities() {
		return nil, errors.New("could not find cluster MSI in cluster doc")
	} else if len(oc.Identity.UserAssignedIdentities) > 1 {
		return nil, errors.New("unexpectedly found more than one cluster MSI in cluster doc")
	}

	var msiResourceId string
	for resourceId := range oc.Identity.UserAssignedIdentities {
		msiResourceId = resourceId
	}

	return arm.ParseResourceID(msiResourceId)
}

// HasUserAssignedIdentities returns true if and only if the cluster doc's
// Identity.UserAssignedIdentities is non-nil and non-empty.
func (oc *OpenShiftCluster) HasUserAssignedIdentities() bool {
	return oc.Identity != nil && oc.Identity.UserAssignedIdentities != nil && len(oc.Identity.UserAssignedIdentities) > 0
}

// CreatedByType by defines user type, which executed the request
// This field should match common-types field names for swagger and sdk generation
type CreatedByType string

const (
	CreatedByTypeApplication     CreatedByType = "Application"
	CreatedByTypeKey             CreatedByType = "Key"
	CreatedByTypeManagedIdentity CreatedByType = "ManagedIdentity"
	CreatedByTypeUser            CreatedByType = "User"
)

// SystemData represets metadata provided by arm. Time fields inside the struct are pointers
// so we could better verify which fields are provided to use by ARM or not. Time package
// does not comply with omitempty. More details about requirements:
// https://github.com/Azure/azure-resource-manager-rpc/blob/master/v1.0/common-api-contracts.md#system-metadata-for-all-azure-resources
type SystemData struct {
	CreatedBy          string        `json:"createdBy,omitempty"`
	CreatedByType      CreatedByType `json:"createdByType,omitempty"`
	CreatedAt          *time.Time    `json:"createdAt,omitempty"`
	LastModifiedBy     string        `json:"lastModifiedBy,omitempty"`
	LastModifiedByType CreatedByType `json:"lastModifiedByType,omitempty"`
	LastModifiedAt     *time.Time    `json:"lastModifiedAt,omitempty"`
}

// SecureBytes represents an encrypted []byte
type SecureBytes []byte

// SecureString represents an encrypted string
type SecureString string

// OpenShiftClusterProperties represents an OpenShift cluster's properties
type OpenShiftClusterProperties struct {
	MissingFields

	// Provisioning state machine:
	//
	// From ARM's perspective, Succeeded and Failed are the only two terminal
	// provisioning states for asynchronous operations.  Clients will poll PUT,
	// PATCH or DELETE operations until the resource gets to one of those
	// provisioning states.
	//
	// ARO uses Creating, Updating and Deleting as non-terminal provisioning
	// states to signal asynchronous operations from the front end to the back
	// end.
	//
	// In case of failures, the back end sets failedProvisioningState to the
	// provisioning state at the time of the failure.
	//
	// The ARO front end gates provisioning state machine transitions as
	// follows:
	//
	// * no PUT, PATCH or DELETE is accepted unless the cluster is currently in
	//   a terminal provisioning state.
	//
	// * DELETE is always allowed regardless of the terminal provisioning state
	//   of the cluster.
	//
	// * PUT and PATCH are allowed as long as the cluster is in Succeeded
	//   provisioning state, or in a Failed provisioning state with the failed
	//   provisioning state to Updating.
	//
	// i.e. if a cluster creation or deletion fails, there is no remedy but to
	// delete the cluster.

	// LastProvisioningState allows the backend to see the last terminal
	// ProvisioningState.  When they complete, regardless of success, admin
	// updates always reset the ProvisioningState to LastProvisioningState.

	ArchitectureVersion     ArchitectureVersion `json:"architectureVersion,omitempty"`
	ProvisioningState       ProvisioningState   `json:"provisioningState,omitempty"`
	LastProvisioningState   ProvisioningState   `json:"lastProvisioningState,omitempty"`
	FailedProvisioningState ProvisioningState   `json:"failedProvisioningState,omitempty"`
	LastAdminUpdateError    string              `json:"lastAdminUpdateError,omitempty"`
	MaintenanceTask         MaintenanceTask     `json:"maintenanceTask,omitempty"`

	// Operator feature/option flags
	OperatorFlags   OperatorFlags `json:"operatorFlags,omitempty"`
	OperatorVersion string        `json:"operatorVersion,omitempty"`

	// Zones are the availability zones this cluster occupies
	// Only new clusters have this set
	Zones []string `json:"zones,omitempty"`

	CreatedAt time.Time `json:"createdAt,omitempty"`

	// CreatedBy is the RP version (Git commit hash) that created this cluster
	CreatedBy string `json:"createdBy,omitempty"`

	// ProvisionedBy is the RP version (Git commit hash) that last successfully
	// admin updated this cluster
	ProvisionedBy string `json:"provisionedBy,omitempty"`

	ClusterProfile ClusterProfile `json:"clusterProfile,omitempty"`

	FeatureProfile FeatureProfile `json:"featureProfile,omitempty"`

	ConsoleProfile ConsoleProfile `json:"consoleProfile,omitempty"`

	ServicePrincipalProfile *ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`

	PlatformWorkloadIdentityProfile *PlatformWorkloadIdentityProfile `json:"platformWorkloadIdentityProfile,omitempty"`

	NetworkProfile NetworkProfile `json:"networkProfile,omitempty"`

	MasterProfile MasterProfile `json:"masterProfile,omitempty"`

	// WorkerProfiles is used to store the worker profile data that was sent in the api request
	WorkerProfiles []WorkerProfile `json:"workerProfiles,omitempty"`

	// WorkerProfilesStatus is used to store the enriched worker profile data
	WorkerProfilesStatus []WorkerProfile `json:"workerProfilesStatus,omitempty" swagger:"readOnly"`

	APIServerProfile APIServerProfile `json:"apiserverProfile,omitempty"`

	IngressProfiles []IngressProfile `json:"ingressProfiles,omitempty"`

	// Install is non-nil only when an install is in progress
	Install *Install `json:"install,omitempty"`

	StorageSuffix                   string `json:"storageSuffix,omitempty"`
	ImageRegistryStorageAccountName string `json:"imageRegistryStorageAccountName,omitempty"`

	InfraID string      `json:"infraId,omitempty"`
	SSHKey  SecureBytes `json:"sshKey,omitempty"`

	// AdminKubeconfig is installer generated kubeconfig. It is 10 year config,
	// and should never be returned to the user.
	AdminKubeconfig SecureBytes `json:"adminKubeconfig,omitempty"`
	// AROServiceKubeconfig is used by ARO services. In example monitor
	AROServiceKubeconfig SecureBytes `json:"aroServiceKubeconfig,omitempty"`
	// AROSREKubeconfig is used by portal when proxying request from SRE
	AROSREKubeconfig SecureBytes `json:"aroSREKubeconfig,omitempty"`
	// KubeadminPassword installer generated kube-admin password
	KubeadminPassword SecureString `json:"kubeadminPassword,omitempty"`

	// UserAdminKubeconfig is derived admin kubeConfig with shorter live span
	UserAdminKubeconfig SecureBytes `json:"userAdminKubeconfig,omitempty"`

	RegistryProfiles []*RegistryProfile `json:"registryProfiles,omitempty"`

	HiveProfile HiveProfile `json:"hiveProfile,omitempty"`

	MaintenanceState MaintenanceState `json:"maintenanceState,omitempty"`
}

// ProvisioningState represents a provisioning state
type ProvisioningState string

// ProvisioningState constants
// TODO: ProvisioningStateCanceled is included to pass upstream CI. It is currently unused in ARO.
const (
	ProvisioningStateCreating      ProvisioningState = "Creating"
	ProvisioningStateUpdating      ProvisioningState = "Updating"
	ProvisioningStateAdminUpdating ProvisioningState = "AdminUpdating"
	ProvisioningStateCanceled      ProvisioningState = "Canceled"
	ProvisioningStateMaintenance   ProvisioningState = "Maintenance"
	ProvisioningStateDeleting      ProvisioningState = "Deleting"
	ProvisioningStateSucceeded     ProvisioningState = "Succeeded"
	ProvisioningStateFailed        ProvisioningState = "Failed"
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

// IsMaintenanceOngoingTask returns true if the maintenance task should change state to maintenance ongoing (planned/unplanned)
func (t MaintenanceTask) IsMaintenanceOngoingTask() bool {
	result := (t == MaintenanceTaskEverything) ||
		(t == MaintenanceTaskOperator) ||
		(t == MaintenanceTaskRenewCerts) ||
		(t == MaintenanceTaskSyncClusterObject) ||
		(t == MaintenanceTaskMigrateLoadBalancer) ||
		(t == "")
	return result
}

// Cluster-scoped flags
type OperatorFlags map[string]string

// IsTerminal returns true if state is Terminal
func (t ProvisioningState) IsTerminal() bool {
	return ProvisioningStateFailed == t || ProvisioningStateSucceeded == t
}

func (t ProvisioningState) String() string {
	return string(t)
}

// FipsValidatedModules determines if FIPS is used.
type FipsValidatedModules string

// FipsValidatedModules constants.
const (
	FipsValidatedModulesEnabled  FipsValidatedModules = "Enabled"
	FipsValidatedModulesDisabled FipsValidatedModules = "Disabled"
)

// OIDCIssuer represents the URL of the managed OIDC issuer in a workload identity cluster.
type OIDCIssuer string

// ClusterProfile represents a cluster profile.
type ClusterProfile struct {
	MissingFields

	PullSecret                    SecureString         `json:"pullSecret,omitempty"`
	Domain                        string               `json:"domain,omitempty"`
	Version                       string               `json:"version,omitempty"`
	ResourceGroupID               string               `json:"resourceGroupId,omitempty"`
	FipsValidatedModules          FipsValidatedModules `json:"fipsValidatedModules,omitempty"`
	OIDCIssuer                    *OIDCIssuer          `json:"oidcIssuer,omitempty"`
	BoundServiceAccountSigningKey *SecureString        `json:"boundServiceAccountSigningKey,omitempty"`
}

// FeatureProfile represents a feature profile.
type FeatureProfile struct {
	MissingFields

	GatewayEnabled bool `json:"gatewayEnabled,omitempty"`
}

// ConsoleProfile represents a console profile.
type ConsoleProfile struct {
	MissingFields

	URL string `json:"url,omitempty"`
}

// ServicePrincipalProfile represents a service principal profile.
type ServicePrincipalProfile struct {
	MissingFields

	ClientID     string       `json:"clientId,omitempty"`
	ClientSecret SecureString `json:"clientSecret,omitempty"`
	SPObjectID   string       `json:"spObjectId,omitempty"`
}

// SoftwareDefinedNetwork
type SoftwareDefinedNetwork string

const (
	SoftwareDefinedNetworkOVNKubernetes SoftwareDefinedNetwork = "OVNKubernetes"
	SoftwareDefinedNetworkOpenShiftSDN  SoftwareDefinedNetwork = "OpenShiftSDN"
)

// MTUSize represents the MTU size of a cluster
type MTUSize int

// MTUSize constants
const (
	MTU1500 MTUSize = 1500
	MTU3900 MTUSize = 3900
)

// The outbound routing strategy used to provide your cluster egress to the internet.
type OutboundType string

// OutboundType constants
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

// NetworkProfile represents a network profile
type NetworkProfile struct {
	MissingFields

	PodCIDR                string                 `json:"podCidr,omitempty"`
	ServiceCIDR            string                 `json:"serviceCidr,omitempty"`
	SoftwareDefinedNetwork SoftwareDefinedNetwork `json:"softwareDefinedNetwork,omitempty"`
	MTUSize                MTUSize                `json:"mtuSize,omitempty"`
	OutboundType           OutboundType           `json:"outboundType,omitempty"`

	APIServerPrivateEndpointIP string               `json:"privateEndpointIp,omitempty"`
	GatewayPrivateEndpointIP   string               `json:"gatewayPrivateEndpointIp,omitempty"`
	GatewayPrivateLinkID       string               `json:"gatewayPrivateLinkId,omitempty"`
	PreconfiguredNSG           PreconfiguredNSG     `json:"preconfiguredNSG,omitempty"`
	LoadBalancerProfile        *LoadBalancerProfile `json:"loadBalancerProfile,omitempty"`
}

// IP address ranges internally used by ARO
var (
	JoinCIDRRange []string = []string{
		"100.64.0.0/16",
		"169.254.169.0/29",
		"100.88.0.0/16",
	}
)

// PreconfiguredNSG represents whether customers want to use their own NSG attached to the subnets
type PreconfiguredNSG string

// PreconfiguredNSG constants
const (
	PreconfiguredNSGEnabled  PreconfiguredNSG = "Enabled"
	PreconfiguredNSGDisabled PreconfiguredNSG = "Disabled"
)

// EncryptionAtHost represents encryption at host.
type EncryptionAtHost string

// EncryptionAtHost constants
const (
	EncryptionAtHostEnabled  EncryptionAtHost = "Enabled"
	EncryptionAtHostDisabled EncryptionAtHost = "Disabled"
)

// MasterProfile represents a master profile
type MasterProfile struct {
	MissingFields

	VMSize              vms.VMSize       `json:"vmSize,omitempty"`
	SubnetID            string           `json:"subnetId,omitempty"`
	EncryptionAtHost    EncryptionAtHost `json:"encryptionAtHost,omitempty"`
	DiskEncryptionSetID string           `json:"diskEncryptionSetId,omitempty"`
}

// WorkerProfile represents a worker profile
type WorkerProfile struct {
	MissingFields

	Name                string           `json:"name,omitempty"`
	VMSize              vms.VMSize       `json:"vmSize,omitempty"`
	DiskSizeGB          int              `json:"diskSizeGB,omitempty"`
	SubnetID            string           `json:"subnetId,omitempty"`
	Count               int              `json:"count,omitempty"`
	EncryptionAtHost    EncryptionAtHost `json:"encryptionAtHost,omitempty"`
	DiskEncryptionSetID string           `json:"diskEncryptionSetId,omitempty"`
}

// GetEnrichedWorkerProfiles returns WorkerProfilesStatus if not nil, otherwise WorkerProfiles
// with their respective json property name
func GetEnrichedWorkerProfiles(ocp OpenShiftClusterProperties) ([]WorkerProfile, string) {
	if ocp.WorkerProfilesStatus != nil {
		return ocp.WorkerProfilesStatus, "workerProfilesStatus"
	}
	return ocp.WorkerProfiles, "workerProfiles"
}

// APIServerProfile represents an API server profile
type APIServerProfile struct {
	MissingFields

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

// IngressProfile represents an ingress profile
type IngressProfile struct {
	MissingFields

	Name       string     `json:"name,omitempty"`
	Visibility Visibility `json:"visibility,omitempty"`
	IP         string     `json:"ip,omitempty"`
}

// RegistryProfile represents a registry's login
type RegistryProfile struct {
	MissingFields

	Name     string       `json:"name,omitempty"`
	Username string       `json:"username,omitempty"`
	Password SecureString `json:"password,omitempty"`
	// IssueDate is when the username/password for the registry was last updated.
	IssueDate *time.Time `json:"issueDate,omitempty"`
}

// Install represents an install process
type Install struct {
	MissingFields

	Now   time.Time    `json:"now,omitempty"`
	Phase InstallPhase `json:"phase"`
}

// InstallPhase represents an install phase
type InstallPhase int

// InstallPhase constants
const (
	InstallPhaseBootstrap InstallPhase = iota
	InstallPhaseRemoveBootstrap
)

// ArchitectureVersion represents an architecture version
type ArchitectureVersion int

// ArchitectureVersion constants
const (
	// ArchitectureVersionV1: 4.3, 4.4: 3 load balancers (internal, control-plane egress, public), 2 NSGs
	ArchitectureVersionV1 ArchitectureVersion = iota
	// ArchitectureVersionV2: 4.5: 2 load balancers (internal, public), 1 NSG.
	ArchitectureVersionV2
)

// HiveProfile represents the hive related data of a cluster
type HiveProfile struct {
	MissingFields

	Namespace string `json:"namespace,omitempty"`

	// CreatedByHive is used during PUCM to skip adoption and reconciliation
	// of clusters that were created by Hive to avoid deleting existing
	// ClusterDeployments.
	CreatedByHive bool `json:"createdByHive,omitempty"`
}

// PlatformWorkloadIdentityProfile encapsulates all information that is specific to workload identity clusters.
type PlatformWorkloadIdentityProfile struct {
	MissingFields

	UpgradeableTo              *UpgradeableTo                      `json:"upgradeableTo,omitempty"`
	PlatformWorkloadIdentities map[string]PlatformWorkloadIdentity `json:"platformWorkloadIdentities,omitempty"`
}

// UpgradeableTo stores a single OpenShift version a workload identity cluster can be upgraded to
type UpgradeableTo string

// PlatformWorkloadIdentity stores information representing a single workload identity.
type PlatformWorkloadIdentity struct {
	MissingFields

	// The resource ID of the PlatformWorkloadIdentity resource
	ResourceID string `json:"resourceId,omitempty"`

	// The ClientID of the PlatformWorkloadIdentity resource
	ClientID string `json:"clientId,omitempty" swagger:"readOnly"`

	// The ObjectID of the PlatformWorkloadIdentity resource
	ObjectID string `json:"objectId,omitempty" swagger:"readOnly"`
}

// UserAssignedIdentity stores information about a user-assigned managed identity in a predefined format required by Microsoft's Managed Identity team.
type UserAssignedIdentity struct {
	MissingFields

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
	MissingFields

	// The type of the ManagedServiceIdentity resource.
	Type ManagedServiceIdentityType `json:"type,omitempty"`

	// The PrincipalID of the Identity resource.
	PrincipalID string `json:"principalId,omitempty" swagger:"readOnly"`

	// A map of user assigned identities attached to the cluster, specified in a type required by Microsoft's Managed Identity team.
	UserAssignedIdentities map[string]UserAssignedIdentity `json:"userAssignedIdentities,omitempty"`

	// The IdentityURL provided by the MSI RP
	IdentityURL string `json:"identityURL,omitempty" mutable:"true"`

	// The TenantID provided by the MSI RP
	TenantID string `json:"tenantId,omitempty" swagger:"readOnly"`
}
