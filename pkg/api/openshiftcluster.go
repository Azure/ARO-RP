package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"
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

	ServicePrincipalProfile ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`

	NetworkProfile NetworkProfile `json:"networkProfile,omitempty"`

	MasterProfile MasterProfile `json:"masterProfile,omitempty"`

	WorkerProfiles []WorkerProfile `json:"workerProfiles,omitempty"`

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
}

// ProvisioningState represents a provisioning state
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

type MaintenanceTask string

const (
	MaintenanceTaskEverything MaintenanceTask = "Everything"
	MaintenanceTaskOperator   MaintenanceTask = "OperatorUpdate"
	MaintenanceTaskRenewCerts MaintenanceTask = "CertificatesRenewal"
)

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

// ClusterProfile represents a cluster profile.
type ClusterProfile struct {
	MissingFields

	PullSecret           SecureString         `json:"pullSecret,omitempty"`
	Domain               string               `json:"domain,omitempty"`
	Version              string               `json:"version,omitempty"`
	ResourceGroupID      string               `json:"resourceGroupId,omitempty"`
	FipsValidatedModules FipsValidatedModules `json:"fipsValidatedModules,omitempty"`
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

// OutboundType represents the type of routing a cluster is using
type OutboundType string

// OutboundType constants
const (
	OutboundTypeUserDefinedRouting OutboundType = "UserDefinedRouting"
	OutboundTypeLoadbalancer       OutboundType = "Loadbalancer"
)

// NetworkProfile represents a network profile
type NetworkProfile struct {
	MissingFields

	PodCIDR                string                 `json:"podCidr,omitempty"`
	ServiceCIDR            string                 `json:"serviceCidr,omitempty"`
	SoftwareDefinedNetwork SoftwareDefinedNetwork `json:"softwareDefinedNetwork,omitempty"`
	MTUSize                MTUSize                `json:"mtuSize,omitempty"`
	OutboundType           OutboundType           `json:"outboundType,omitempty"`

	APIServerPrivateEndpointIP string `json:"privateEndpointIp,omitempty"`
	GatewayPrivateEndpointIP   string `json:"gatewayPrivateEndpointIp,omitempty"`
	GatewayPrivateLinkID       string `json:"gatewayPrivateLinkId,omitempty"`
}

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

// VMSize constants
// add required resources in pkg/api/validate/dynamic/quota.go when adding a new VMSize
const (
	VMSizeStandardD2sV3 VMSize = "Standard_D2s_v3"

	VMSizeStandardD4asV4  VMSize = "Standard_D4as_v4"
	VMSizeStandardD8asV4  VMSize = "Standard_D8as_v4"
	VMSizeStandardD16asV4 VMSize = "Standard_D16as_v4"
	VMSizeStandardD32asV4 VMSize = "Standard_D32as_v4"

	VMSizeStandardD4sV3  VMSize = "Standard_D4s_v3"
	VMSizeStandardD8sV3  VMSize = "Standard_D8s_v3"
	VMSizeStandardD16sV3 VMSize = "Standard_D16s_v3"
	VMSizeStandardD32sV3 VMSize = "Standard_D32s_v3"

	VMSizeStandardE4sV3     VMSize = "Standard_E4s_v3"
	VMSizeStandardE8sV3     VMSize = "Standard_E8s_v3"
	VMSizeStandardE16sV3    VMSize = "Standard_E16s_v3"
	VMSizeStandardE32sV3    VMSize = "Standard_E32s_v3"
	VMSizeStandardE64isV3   VMSize = "Standard_E64is_v3"
	VMSizeStandardE64iV3    VMSize = "Standard_E64i_v3"
	VMSizeStandardE80isV4   VMSize = "Standard_E80is_v4"
	VMSizeStandardE80idsV4  VMSize = "Standard_E80ids_v4"
	VMSizeStandardE104iV5   VMSize = "Standard_E104i_v5"
	VMSizeStandardE104isV5  VMSize = "Standard_E104is_v5"
	VMSizeStandardE104idV5  VMSize = "Standard_E104id_v5"
	VMSizeStandardE104idsV5 VMSize = "Standard_E104ids_v5"

	VMSizeStandardF4sV2  VMSize = "Standard_F4s_v2"
	VMSizeStandardF8sV2  VMSize = "Standard_F8s_v2"
	VMSizeStandardF16sV2 VMSize = "Standard_F16s_v2"
	VMSizeStandardF32sV2 VMSize = "Standard_F32s_v2"
	VMSizeStandardF72sV2 VMSize = "Standard_F72s_v2"

	VMSizeStandardM128ms VMSize = "Standard_M128ms"
	VMSizeStandardG5     VMSize = "Standard_G5"
	VMSizeStandardGS5    VMSize = "Standard_GS5"

	VMSizeStandardL4s    VMSize = "Standard_L4s"
	VMSizeStandardL8s    VMSize = "Standard_L8s"
	VMSizeStandardL16s   VMSize = "Standard_L16s"
	VMSizeStandardL32s   VMSize = "Standard_L32s"
	VMSizeStandardL8sV2  VMSize = "Standard_L8s_v2"
	VMSizeStandardL16sV2 VMSize = "Standard_L16s_v2"
	VMSizeStandardL32sV2 VMSize = "Standard_L32s_v2"
	VMSizeStandardL48sV2 VMSize = "Standard_L48s_v2"
	VMSizeStandardL64sV2 VMSize = "Standard_L64s_v2"

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
	CoreCount int
	Family    string
}

var (
	VMSizeStandardD2sV3Struct = VMSizeStruct{CoreCount: 2, Family: standardDSv3}

	VMSizeStandardD4asV4Struct  = VMSizeStruct{CoreCount: 4, Family: standardDASv4}
	VMSizeStandardD8asV4Struct  = VMSizeStruct{CoreCount: 8, Family: standardDASv4}
	VMSizeStandardD16asV4Struct = VMSizeStruct{CoreCount: 16, Family: standardDASv4}
	VMSizeStandardD32asV4Struct = VMSizeStruct{CoreCount: 32, Family: standardDASv4}

	VMSizeStandardD4sV3Struct  = VMSizeStruct{CoreCount: 4, Family: standardDSv3}
	VMSizeStandardD8sV3Struct  = VMSizeStruct{CoreCount: 8, Family: standardDSv3}
	VMSizeStandardD16sV3Struct = VMSizeStruct{CoreCount: 16, Family: standardDSv3}
	VMSizeStandardD32sV3Struct = VMSizeStruct{CoreCount: 32, Family: standardDSv3}

	VMSizeStandardE4sV3Struct     = VMSizeStruct{CoreCount: 4, Family: standardESv3}
	VMSizeStandardE8sV3Struct     = VMSizeStruct{CoreCount: 8, Family: standardESv3}
	VMSizeStandardE16sV3Struct    = VMSizeStruct{CoreCount: 16, Family: standardESv3}
	VMSizeStandardE32sV3Struct    = VMSizeStruct{CoreCount: 32, Family: standardESv3}
	VMSizeStandardE64isV3Struct   = VMSizeStruct{CoreCount: 64, Family: standardESv3}
	VMSizeStandardE64iV3Struct    = VMSizeStruct{CoreCount: 64, Family: standardESv3}
	VMSizeStandardE80isV4Struct   = VMSizeStruct{CoreCount: 80, Family: standardEISv4}
	VMSizeStandardE80idsV4Struct  = VMSizeStruct{CoreCount: 80, Family: standardEIDSv4}
	VMSizeStandardE104iV5Struct   = VMSizeStruct{CoreCount: 104, Family: standardEIv5}
	VMSizeStandardE104isV5Struct  = VMSizeStruct{CoreCount: 104, Family: standardEISv5}
	VMSizeStandardE104idV5Struct  = VMSizeStruct{CoreCount: 104, Family: standardEIDv5}
	VMSizeStandardE104idsV5Struct = VMSizeStruct{CoreCount: 104, Family: standardEIDSv5}

	VMSizeStandardF4sV2Struct  = VMSizeStruct{CoreCount: 4, Family: standardFSv2}
	VMSizeStandardF8sV2Struct  = VMSizeStruct{CoreCount: 8, Family: standardFSv2}
	VMSizeStandardF16sV2Struct = VMSizeStruct{CoreCount: 16, Family: standardFSv2}
	VMSizeStandardF32sV2Struct = VMSizeStruct{CoreCount: 32, Family: standardFSv2}
	VMSizeStandardF72sV2Struct = VMSizeStruct{CoreCount: 72, Family: standardFSv2}

	VMSizeStandardM128msStruct = VMSizeStruct{CoreCount: 128, Family: standardMS}
	VMSizeStandardG5Struct     = VMSizeStruct{CoreCount: 32, Family: standardGFamily}
	VMSizeStandardGS5Struct    = VMSizeStruct{CoreCount: 32, Family: standardGFamily}

	VMSizeStandardL4sStruct    = VMSizeStruct{CoreCount: 4, Family: standardLSv2}
	VMSizeStandardL8sStruct    = VMSizeStruct{CoreCount: 8, Family: standardLSv2}
	VMSizeStandardL16sStruct   = VMSizeStruct{CoreCount: 16, Family: standardLSv2}
	VMSizeStandardL32sStruct   = VMSizeStruct{CoreCount: 32, Family: standardLSv2}
	VMSizeStandardL8sV2Struct  = VMSizeStruct{CoreCount: 8, Family: standardLSv2}
	VMSizeStandardL16sV2Struct = VMSizeStruct{CoreCount: 16, Family: standardLSv2}
	VMSizeStandardL32sV2Struct = VMSizeStruct{CoreCount: 32, Family: standardLSv2}
	VMSizeStandardL48sV2Struct = VMSizeStruct{CoreCount: 48, Family: standardLSv2}
	VMSizeStandardL64sV2Struct = VMSizeStruct{CoreCount: 64, Family: standardLSv2}

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
	standardDSv3    = "standardDSv3Family"
	standardDASv4   = "standardDASv4Family"
	standardESv3    = "standardESv3Family"
	standardEISv4   = "standardEISv4Family"
	standardEIDSv4  = "standardEIDSv4Family"
	standardEIv5    = "standardEIv5Family"
	standardEISv5   = "standardEISv5Family"
	standardEIDSv5  = "standardEIDSv5Family"
	standardEIDv5   = "standardEIDv5Family"
	standardFSv2    = "standardFSv2Family"
	standardMS      = "standardMSFamily"
	standardGFamily = "standardGFamily"
	standardLSv2    = "standardLsv2Family"
	standardNCAS    = "Standard NCASv3_T4 Family"
	standardNCSv3   = "Standard NCSv3 Family"
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
	// ArchitectureVersionV1: 4.3, 4.4: 2 load balancers, 2 NSGs
	ArchitectureVersionV1 ArchitectureVersion = iota
	// ArchitectureVersionV2: 4.5: 1 load balancer, 1 NSG
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
