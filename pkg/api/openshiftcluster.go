package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/go-autorest/autorest/date"
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

	//this property is used in the enrichers. Should not be marshalled
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

	MaintenanceTaskEverything        MaintenanceTask = "Everything"
	MaintenanceTaskOperator          MaintenanceTask = "OperatorUpdate"
	MaintenanceTaskRenewCerts        MaintenanceTask = "CertificatesRenewal"
	MaintenanceTaskSyncClusterObject MaintenanceTask = "SyncClusterObject"

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
	// The desired availability zones for the load balancer frontend IPs/RP-created PIPs
	Zones []string `json:"outboundIpAvailabilityZones,omitempty"`
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

	VMSize              VMSize           `json:"vmSize,omitempty"`
	SubnetID            string           `json:"subnetId,omitempty"`
	EncryptionAtHost    EncryptionAtHost `json:"encryptionAtHost,omitempty"`
	DiskEncryptionSetID string           `json:"diskEncryptionSetId,omitempty"`
}

// VMSize represents a VM size
type VMSize string

func (vmSize VMSize) String() string {
	return string(vmSize)
}

// VMSize constants
// add required resources in pkg/validate/dynamic/quota.go when adding a new VMSize
const (
	VMSizeStandardD2sV3  VMSize = "Standard_D2s_v3"
	VMSizeStandardD4sV3  VMSize = "Standard_D4s_v3"
	VMSizeStandardD8sV3  VMSize = "Standard_D8s_v3"
	VMSizeStandardD16sV3 VMSize = "Standard_D16s_v3"
	VMSizeStandardD32sV3 VMSize = "Standard_D32s_v3"

	VMSizeStandardD2sV4  VMSize = "Standard_D2s_v4"
	VMSizeStandardD4sV4  VMSize = "Standard_D4s_v4"
	VMSizeStandardD8sV4  VMSize = "Standard_D8s_v4"
	VMSizeStandardD16sV4 VMSize = "Standard_D16s_v4"
	VMSizeStandardD32sV4 VMSize = "Standard_D32s_v4"
	VMSizeStandardD64sV4 VMSize = "Standard_D64s_v4"

	VMSizeStandardD2sV5  VMSize = "Standard_D2s_v5"
	VMSizeStandardD4sV5  VMSize = "Standard_D4s_v5"
	VMSizeStandardD8sV5  VMSize = "Standard_D8s_v5"
	VMSizeStandardD16sV5 VMSize = "Standard_D16s_v5"
	VMSizeStandardD32sV5 VMSize = "Standard_D32s_v5"
	VMSizeStandardD64sV5 VMSize = "Standard_D64s_v5"
	VMSizeStandardD96sV5 VMSize = "Standard_D96s_v5"

	VMSizeStandardD4asV4  VMSize = "Standard_D4as_v4"
	VMSizeStandardD8asV4  VMSize = "Standard_D8as_v4"
	VMSizeStandardD16asV4 VMSize = "Standard_D16as_v4"
	VMSizeStandardD32asV4 VMSize = "Standard_D32as_v4"
	VMSizeStandardD64asV4 VMSize = "Standard_D64as_v4"
	VMSizeStandardD96asV4 VMSize = "Standard_D96as_v4"

	VMSizeStandardD4asV5  VMSize = "Standard_D4as_v5"
	VMSizeStandardD8asV5  VMSize = "Standard_D8as_v5"
	VMSizeStandardD16asV5 VMSize = "Standard_D16as_v5"
	VMSizeStandardD32asV5 VMSize = "Standard_D32as_v5"
	VMSizeStandardD64asV5 VMSize = "Standard_D64as_v5"
	VMSizeStandardD96asV5 VMSize = "Standard_D96as_v5"

	VMSizeStandardD4dsV5  VMSize = "Standard_D4ds_v5"
	VMSizeStandardD8dsV5  VMSize = "Standard_D8ds_v5"
	VMSizeStandardD16dsV5 VMSize = "Standard_D16ds_v5"
	VMSizeStandardD32dsV5 VMSize = "Standard_D32ds_v5"
	VMSizeStandardD64dsV5 VMSize = "Standard_D64ds_v5"
	VMSizeStandardD96dsV5 VMSize = "Standard_D96ds_v5"

	VMSizeStandardE4sV3  VMSize = "Standard_E4s_v3"
	VMSizeStandardE8sV3  VMSize = "Standard_E8s_v3"
	VMSizeStandardE16sV3 VMSize = "Standard_E16s_v3"
	VMSizeStandardE32sV3 VMSize = "Standard_E32s_v3"

	VMSizeStandardE2sV4  VMSize = "Standard_E2s_v4"
	VMSizeStandardE4sV4  VMSize = "Standard_E4s_v4"
	VMSizeStandardE8sV4  VMSize = "Standard_E8s_v4"
	VMSizeStandardE16sV4 VMSize = "Standard_E16s_v4"
	VMSizeStandardE20sV4 VMSize = "Standard_E20s_v4"
	VMSizeStandardE32sV4 VMSize = "Standard_E32s_v4"
	VMSizeStandardE48sV4 VMSize = "Standard_E48s_v4"
	VMSizeStandardE64sV4 VMSize = "Standard_E64s_v4"

	VMSizeStandardE2sV5  VMSize = "Standard_E2s_v5"
	VMSizeStandardE4sV5  VMSize = "Standard_E4s_v5"
	VMSizeStandardE8sV5  VMSize = "Standard_E8s_v5"
	VMSizeStandardE16sV5 VMSize = "Standard_E16s_v5"
	VMSizeStandardE20sV5 VMSize = "Standard_E20s_v5"
	VMSizeStandardE32sV5 VMSize = "Standard_E32s_v5"
	VMSizeStandardE48sV5 VMSize = "Standard_E48s_v5"
	VMSizeStandardE64sV5 VMSize = "Standard_E64s_v5"
	VMSizeStandardE96sV5 VMSize = "Standard_E96s_v5"

	VMSizeStandardE4asV4  VMSize = "Standard_E4as_v4"
	VMSizeStandardE8asV4  VMSize = "Standard_E8as_v4"
	VMSizeStandardE16asV4 VMSize = "Standard_E16as_v4"
	VMSizeStandardE20asV4 VMSize = "Standard_E20as_v4"
	VMSizeStandardE32asV4 VMSize = "Standard_E32as_v4"
	VMSizeStandardE48asV4 VMSize = "Standard_E48as_v4"
	VMSizeStandardE64asV4 VMSize = "Standard_E64as_v4"
	VMSizeStandardE96asV4 VMSize = "Standard_E96as_v4"

	VMSizeStandardE8asV5  VMSize = "Standard_E8as_v5"
	VMSizeStandardE16asV5 VMSize = "Standard_E16as_v5"
	VMSizeStandardE20asV5 VMSize = "Standard_E20as_v5"
	VMSizeStandardE32asV5 VMSize = "Standard_E32as_v5"
	VMSizeStandardE48asV5 VMSize = "Standard_E48as_v5"
	VMSizeStandardE64asV5 VMSize = "Standard_E64as_v5"
	VMSizeStandardE96asV5 VMSize = "Standard_E96as_v5"

	VMSizeStandardE64isV3   VMSize = "Standard_E64is_v3"
	VMSizeStandardE80isV4   VMSize = "Standard_E80is_v4"
	VMSizeStandardE80idsV4  VMSize = "Standard_E80ids_v4"
	VMSizeStandardE96dsV5   VMSize = "Standard_E96ds_v5"
	VMSizeStandardE104isV5  VMSize = "Standard_E104is_v5"
	VMSizeStandardE104idsV5 VMSize = "Standard_E104ids_v5"

	VMSizeStandardF4sV2  VMSize = "Standard_F4s_v2"
	VMSizeStandardF8sV2  VMSize = "Standard_F8s_v2"
	VMSizeStandardF16sV2 VMSize = "Standard_F16s_v2"
	VMSizeStandardF32sV2 VMSize = "Standard_F32s_v2"
	VMSizeStandardF72sV2 VMSize = "Standard_F72s_v2"

	VMSizeStandardM128ms VMSize = "Standard_M128ms"

	VMSizeStandardL4s  VMSize = "Standard_L4s"
	VMSizeStandardL8s  VMSize = "Standard_L8s"
	VMSizeStandardL16s VMSize = "Standard_L16s"
	VMSizeStandardL32s VMSize = "Standard_L32s"

	VMSizeStandardL8sV2  VMSize = "Standard_L8s_v2"
	VMSizeStandardL16sV2 VMSize = "Standard_L16s_v2"
	VMSizeStandardL32sV2 VMSize = "Standard_L32s_v2"
	VMSizeStandardL48sV2 VMSize = "Standard_L48s_v2"
	VMSizeStandardL64sV2 VMSize = "Standard_L64s_v2"

	VMSizeStandardL8sV3  VMSize = "Standard_L8s_v3"
	VMSizeStandardL16sV3 VMSize = "Standard_L16s_v3"
	VMSizeStandardL32sV3 VMSize = "Standard_L32s_v3"
	VMSizeStandardL48sV3 VMSize = "Standard_L48s_v3"
	VMSizeStandardL64sV3 VMSize = "Standard_L64s_v3"

	// GPU VMs
	VMSizeStandardNC4asT4V3  VMSize = "Standard_NC4as_T4_v3"
	VMSizeStandardNC8asT4V3  VMSize = "Standard_NC8as_T4_v3"
	VMSizeStandardNC16asT4V3 VMSize = "Standard_NC16as_T4_v3"
	VMSizeStandardNC64asT4V3 VMSize = "Standard_NC64as_T4_v3"

	VMSizeStandardNC6sV3   VMSize = "Standard_NC6s_v3"
	VMSizeStandardNC12sV3  VMSize = "Standard_NC12s_v3"
	VMSizeStandardNC24sV3  VMSize = "Standard_NC24s_v3"
	VMSizeStandardNC24rsV3 VMSize = "Standard_NC24rs_v3"
)

type VMSizeStruct struct {
	CoreCount int    `json:"coreCount,omitempty"`
	Family    string `json:"family,omitempty"`
}

var (
	VMSizeStandardD2sV3Struct  = VMSizeStruct{CoreCount: 2, Family: standardDSv3}
	VMSizeStandardD4sV3Struct  = VMSizeStruct{CoreCount: 4, Family: standardDSv3}
	VMSizeStandardD8sV3Struct  = VMSizeStruct{CoreCount: 8, Family: standardDSv3}
	VMSizeStandardD16sV3Struct = VMSizeStruct{CoreCount: 16, Family: standardDSv3}
	VMSizeStandardD32sV3Struct = VMSizeStruct{CoreCount: 32, Family: standardDSv3}

	VMSizeStandardD2sV4Struct  = VMSizeStruct{CoreCount: 2, Family: standardDSv4}
	VMSizeStandardD4sV4Struct  = VMSizeStruct{CoreCount: 4, Family: standardDSv4}
	VMSizeStandardD8sV4Struct  = VMSizeStruct{CoreCount: 8, Family: standardDSv4}
	VMSizeStandardD16sV4Struct = VMSizeStruct{CoreCount: 16, Family: standardDSv4}
	VMSizeStandardD32sV4Struct = VMSizeStruct{CoreCount: 32, Family: standardDSv4}
	VMSizeStandardD64sV4Struct = VMSizeStruct{CoreCount: 64, Family: standardDSv4}

	VMSizeStandardD2sV5Struct  = VMSizeStruct{CoreCount: 2, Family: standardDSv5}
	VMSizeStandardD4sV5Struct  = VMSizeStruct{CoreCount: 4, Family: standardDSv5}
	VMSizeStandardD8sV5Struct  = VMSizeStruct{CoreCount: 8, Family: standardDSv5}
	VMSizeStandardD16sV5Struct = VMSizeStruct{CoreCount: 16, Family: standardDSv5}
	VMSizeStandardD32sV5Struct = VMSizeStruct{CoreCount: 32, Family: standardDSv5}
	VMSizeStandardD64sV5Struct = VMSizeStruct{CoreCount: 64, Family: standardDSv5}
	VMSizeStandardD96sV5Struct = VMSizeStruct{CoreCount: 96, Family: standardDSv5}

	VMSizeStandardD4asV4Struct  = VMSizeStruct{CoreCount: 4, Family: standardDASv4}
	VMSizeStandardD8asV4Struct  = VMSizeStruct{CoreCount: 8, Family: standardDASv4}
	VMSizeStandardD16asV4Struct = VMSizeStruct{CoreCount: 16, Family: standardDASv4}
	VMSizeStandardD32asV4Struct = VMSizeStruct{CoreCount: 32, Family: standardDASv4}
	VMSizeStandardD64asV4Struct = VMSizeStruct{CoreCount: 64, Family: standardDASv4}
	VMSizeStandardD96asV4Struct = VMSizeStruct{CoreCount: 96, Family: standardDASv4}

	VMSizeStandardD4asV5Struct  = VMSizeStruct{CoreCount: 4, Family: standardDASv5}
	VMSizeStandardD8asV5Struct  = VMSizeStruct{CoreCount: 8, Family: standardDASv5}
	VMSizeStandardD16asV5Struct = VMSizeStruct{CoreCount: 16, Family: standardDASv5}
	VMSizeStandardD32asV5Struct = VMSizeStruct{CoreCount: 32, Family: standardDASv5}
	VMSizeStandardD64asV5Struct = VMSizeStruct{CoreCount: 64, Family: standardDASv5}
	VMSizeStandardD96asV5Struct = VMSizeStruct{CoreCount: 96, Family: standardDASv5}

	VMSizeStandardD4dsV5Struct  = VMSizeStruct{CoreCount: 4, Family: standardDDSv5}
	VMSizeStandardD8dsV5Struct  = VMSizeStruct{CoreCount: 8, Family: standardDDSv5}
	VMSizeStandardD16dsV5Struct = VMSizeStruct{CoreCount: 16, Family: standardDDSv5}
	VMSizeStandardD32dsV5Struct = VMSizeStruct{CoreCount: 32, Family: standardDDSv5}
	VMSizeStandardD64dsV5Struct = VMSizeStruct{CoreCount: 64, Family: standardDDSv5}
	VMSizeStandardD96dsV5Struct = VMSizeStruct{CoreCount: 96, Family: standardDDSv5}

	VMSizeStandardE4sV3Struct  = VMSizeStruct{CoreCount: 4, Family: standardESv3}
	VMSizeStandardE8sV3Struct  = VMSizeStruct{CoreCount: 8, Family: standardESv3}
	VMSizeStandardE16sV3Struct = VMSizeStruct{CoreCount: 16, Family: standardESv3}
	VMSizeStandardE32sV3Struct = VMSizeStruct{CoreCount: 32, Family: standardESv3}

	VMSizeStandardE2sV4Struct  = VMSizeStruct{CoreCount: 2, Family: standardESv4}
	VMSizeStandardE4sV4Struct  = VMSizeStruct{CoreCount: 4, Family: standardESv4}
	VMSizeStandardE8sV4Struct  = VMSizeStruct{CoreCount: 8, Family: standardESv4}
	VMSizeStandardE16sV4Struct = VMSizeStruct{CoreCount: 16, Family: standardESv4}
	VMSizeStandardE20sV4Struct = VMSizeStruct{CoreCount: 20, Family: standardESv4}
	VMSizeStandardE32sV4Struct = VMSizeStruct{CoreCount: 32, Family: standardESv4}
	VMSizeStandardE48sV4Struct = VMSizeStruct{CoreCount: 48, Family: standardESv4}
	VMSizeStandardE64sV4Struct = VMSizeStruct{CoreCount: 64, Family: standardESv4}

	VMSizeStandardE2sV5Struct  = VMSizeStruct{CoreCount: 2, Family: standardESv5}
	VMSizeStandardE4sV5Struct  = VMSizeStruct{CoreCount: 4, Family: standardESv5}
	VMSizeStandardE8sV5Struct  = VMSizeStruct{CoreCount: 8, Family: standardESv5}
	VMSizeStandardE16sV5Struct = VMSizeStruct{CoreCount: 16, Family: standardESv5}
	VMSizeStandardE20sV5Struct = VMSizeStruct{CoreCount: 20, Family: standardESv5}
	VMSizeStandardE32sV5Struct = VMSizeStruct{CoreCount: 32, Family: standardESv5}
	VMSizeStandardE48sV5Struct = VMSizeStruct{CoreCount: 48, Family: standardESv5}
	VMSizeStandardE64sV5Struct = VMSizeStruct{CoreCount: 64, Family: standardESv5}
	VMSizeStandardE96sV5Struct = VMSizeStruct{CoreCount: 96, Family: standardESv5}

	VMSizeStandardE4asV4Struct  = VMSizeStruct{CoreCount: 4, Family: standardEASv4}
	VMSizeStandardE8asV4Struct  = VMSizeStruct{CoreCount: 8, Family: standardEASv4}
	VMSizeStandardE16asV4Struct = VMSizeStruct{CoreCount: 16, Family: standardEASv4}
	VMSizeStandardE20asV4Struct = VMSizeStruct{CoreCount: 20, Family: standardEASv4}
	VMSizeStandardE32asV4Struct = VMSizeStruct{CoreCount: 32, Family: standardEASv4}
	VMSizeStandardE48asV4Struct = VMSizeStruct{CoreCount: 48, Family: standardEASv4}
	VMSizeStandardE64asV4Struct = VMSizeStruct{CoreCount: 64, Family: standardEASv4}
	VMSizeStandardE96asV4Struct = VMSizeStruct{CoreCount: 96, Family: standardEASv4}

	VMSizeStandardE8asV5Struct  = VMSizeStruct{CoreCount: 8, Family: standardEASv5}
	VMSizeStandardE16asV5Struct = VMSizeStruct{CoreCount: 16, Family: standardEASv5}
	VMSizeStandardE20asV5Struct = VMSizeStruct{CoreCount: 20, Family: standardEASv5}
	VMSizeStandardE32asV5Struct = VMSizeStruct{CoreCount: 32, Family: standardEASv5}
	VMSizeStandardE48asV5Struct = VMSizeStruct{CoreCount: 48, Family: standardEASv5}
	VMSizeStandardE64asV5Struct = VMSizeStruct{CoreCount: 64, Family: standardEASv5}
	VMSizeStandardE96asV5Struct = VMSizeStruct{CoreCount: 96, Family: standardEASv5}

	VMSizeStandardE64isV3Struct   = VMSizeStruct{CoreCount: 64, Family: standardESv3}
	VMSizeStandardE80isV4Struct   = VMSizeStruct{CoreCount: 80, Family: standardEISv4}
	VMSizeStandardE80idsV4Struct  = VMSizeStruct{CoreCount: 80, Family: standardEIDSv4}
	VMSizeStandardE96dsV5Struct   = VMSizeStruct{CoreCount: 96, Family: standardEDSv5}
	VMSizeStandardE104isV5Struct  = VMSizeStruct{CoreCount: 104, Family: standardEISv5}
	VMSizeStandardE104idsV5Struct = VMSizeStruct{CoreCount: 104, Family: standardEIDSv5}

	VMSizeStandardF4sV2Struct  = VMSizeStruct{CoreCount: 4, Family: standardFSv2}
	VMSizeStandardF8sV2Struct  = VMSizeStruct{CoreCount: 8, Family: standardFSv2}
	VMSizeStandardF16sV2Struct = VMSizeStruct{CoreCount: 16, Family: standardFSv2}
	VMSizeStandardF32sV2Struct = VMSizeStruct{CoreCount: 32, Family: standardFSv2}
	VMSizeStandardF72sV2Struct = VMSizeStruct{CoreCount: 72, Family: standardFSv2}

	VMSizeStandardM128msStruct = VMSizeStruct{CoreCount: 128, Family: standardMS}

	VMSizeStandardL4sStruct  = VMSizeStruct{CoreCount: 4, Family: standardLSv2}
	VMSizeStandardL8sStruct  = VMSizeStruct{CoreCount: 8, Family: standardLSv2}
	VMSizeStandardL16sStruct = VMSizeStruct{CoreCount: 16, Family: standardLSv2}
	VMSizeStandardL32sStruct = VMSizeStruct{CoreCount: 32, Family: standardLSv2}

	VMSizeStandardL8sV2Struct  = VMSizeStruct{CoreCount: 8, Family: standardLSv2}
	VMSizeStandardL16sV2Struct = VMSizeStruct{CoreCount: 16, Family: standardLSv2}
	VMSizeStandardL32sV2Struct = VMSizeStruct{CoreCount: 32, Family: standardLSv2}
	VMSizeStandardL48sV2Struct = VMSizeStruct{CoreCount: 48, Family: standardLSv2}
	VMSizeStandardL64sV2Struct = VMSizeStruct{CoreCount: 64, Family: standardLSv2}

	VMSizeStandardL8sV3Struct  = VMSizeStruct{CoreCount: 8, Family: standardLSv3}
	VMSizeStandardL16sV3Struct = VMSizeStruct{CoreCount: 16, Family: standardLSv3}
	VMSizeStandardL32sV3Struct = VMSizeStruct{CoreCount: 32, Family: standardLSv3}
	VMSizeStandardL48sV3Struct = VMSizeStruct{CoreCount: 48, Family: standardLSv3}
	VMSizeStandardL64sV3Struct = VMSizeStruct{CoreCount: 64, Family: standardLSv3}

	//Struct GPU nodes
	//Struct the formatting of the ncasv3_t4 family is different.  This can be seen through a
	//Struct az vm list-usage -l eastus
	VMSizeStandardNC4asT4V3Struct  = VMSizeStruct{CoreCount: 4, Family: standardNCAS}
	VMSizeStandardNC8asT4V3Struct  = VMSizeStruct{CoreCount: 8, Family: standardNCAS}
	VMSizeStandardNC16asT4V3Struct = VMSizeStruct{CoreCount: 16, Family: standardNCAS}
	VMSizeStandardNC64asT4V3Struct = VMSizeStruct{CoreCount: 64, Family: standardNCAS}

	VMSizeStandardNC6sV3Struct   = VMSizeStruct{CoreCount: 6, Family: standardNCSv3}
	VMSizeStandardNC12sV3Struct  = VMSizeStruct{CoreCount: 12, Family: standardNCSv3}
	VMSizeStandardNC24sV3Struct  = VMSizeStruct{CoreCount: 24, Family: standardNCSv3}
	VMSizeStandardNC24rsV3Struct = VMSizeStruct{CoreCount: 24, Family: standardNCSv3}
)

const (
	standardDSv3   = "standardDSv3Family"
	standardDSv4   = "standardDSv4Family"
	standardDSv5   = "standardDSv5Family"
	standardDASv4  = "standardDASv4Family"
	standardDASv5  = "standardDASv5Family"
	standardDDSv5  = "standardDDSv5Family"
	standardESv3   = "standardESv3Family"
	standardESv4   = "standardESv4Family"
	standardESv5   = "standardESv5Family"
	standardEASv4  = "standardEASv4Family"
	standardEASv5  = "standardEASv5Family"
	standardEISv4  = "standardEISv4Family"
	standardEIDSv4 = "standardEIDSv4Family"
	standardEISv5  = "standardEISv5Family"
	standardEDSv5  = "standardEDSv5Family"
	standardEIDSv5 = "standardEIDSv5Family"
	standardEIDv5  = "standardEIDv5Family"
	standardFSv2   = "standardFSv2Family"
	standardMS     = "standardMSFamily"
	standardLSv2   = "standardLsv2Family"
	standardLSv3   = "standardLsv3Family"
	standardNCAS   = "Standard NCASv3_T4 Family"
	standardNCSv3  = "Standard NCSv3 Family"
)

// WorkerProfile represents a worker profile
type WorkerProfile struct {
	MissingFields

	Name                string           `json:"name,omitempty"`
	VMSize              VMSize           `json:"vmSize,omitempty"`
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
	IssueDate *date.Time `json:"issueDate,omitempty"`
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
