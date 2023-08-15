package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"
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
	ID         string                     `json:"id,omitempty" mutable:"case"`
	Name       string                     `json:"name,omitempty" mutable:"case"`
	Type       string                     `json:"type,omitempty" mutable:"case"`
	Location   string                     `json:"location,omitempty"`
	Tags       map[string]string          `json:"tags,omitempty"`
	Properties OpenShiftClusterProperties `json:"properties,omitempty"`
}

// OpenShiftClusterProperties represents an OpenShift cluster's properties.
type OpenShiftClusterProperties struct {
	ArchitectureVersion             ArchitectureVersion     `json:"architectureVersion"` // ArchitectureVersion is int so 0 is valid value to be returned
	ProvisioningState               ProvisioningState       `json:"provisioningState,omitempty"`
	LastProvisioningState           ProvisioningState       `json:"lastProvisioningState,omitempty"`
	FailedProvisioningState         ProvisioningState       `json:"failedProvisioningState,omitempty"`
	LastAdminUpdateError            string                  `json:"lastAdminUpdateError,omitempty"`
	MaintenanceTask                 MaintenanceTask         `json:"maintenanceTask,omitempty" mutable:"true"`
	OperatorFlags                   OperatorFlags           `json:"operatorFlags,omitempty" mutable:"true"`
	OperatorVersion                 string                  `json:"operatorVersion,omitempty" mutable:"true"`
	CreatedAt                       time.Time               `json:"createdAt,omitempty"`
	CreatedBy                       string                  `json:"createdBy,omitempty"`
	ProvisionedBy                   string                  `json:"provisionedBy,omitempty"`
	ClusterProfile                  ClusterProfile          `json:"clusterProfile,omitempty"`
	FeatureProfile                  FeatureProfile          `json:"featureProfile,omitempty"`
	ConsoleProfile                  ConsoleProfile          `json:"consoleProfile,omitempty"`
	ServicePrincipalProfile         ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`
	NetworkProfile                  NetworkProfile          `json:"networkProfile,omitempty"`
	MasterProfile                   MasterProfile           `json:"masterProfile,omitempty"`
	WorkerProfiles                  []WorkerProfile         `json:"workerProfiles,omitempty"`
	APIServerProfile                APIServerProfile        `json:"apiserverProfile,omitempty"`
	IngressProfiles                 []IngressProfile        `json:"ingressProfiles,omitempty"`
	Install                         *Install                `json:"install,omitempty"`
	StorageSuffix                   string                  `json:"storageSuffix,omitempty"`
	RegistryProfiles                []RegistryProfile       `json:"registryProfiles,omitempty"`
	ImageRegistryStorageAccountName string                  `json:"imageRegistryStorageAccountName,omitempty"`
	InfraID                         string                  `json:"infraId,omitempty"`
	HiveProfile                     HiveProfile             `json:"hiveProfile,omitempty"`
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

// FipsValidatedModules constants.
const (
	FipsValidatedModulesEnabled  FipsValidatedModules = "Enabled"
	FipsValidatedModulesDisabled FipsValidatedModules = "Disabled"
)

type MaintenanceTask string

const (
	MaintenanceTaskEverything MaintenanceTask = "Everything"
	MaintenanceTaskOperator   MaintenanceTask = "OperatorUpdate"
	MaintenanceTaskRenewCerts MaintenanceTask = "CertificatesRenewal"
)

// Operator feature flags
type OperatorFlags map[string]string

// ClusterProfile represents a cluster profile.
type ClusterProfile struct {
	Domain               string               `json:"domain,omitempty"`
	Version              string               `json:"version,omitempty"`
	ResourceGroupID      string               `json:"resourceGroupId,omitempty"`
	FipsValidatedModules FipsValidatedModules `json:"fipsValidatedModules,omitempty"`
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
	ClientID   string `json:"clientId,omitempty"`
	SPObjectID string `json:"spObjectId,omitempty"`
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

// NetworkProfile represents a network profile.
type NetworkProfile struct {
	// The software defined network (SDN) to use when installing the cluster.
	SoftwareDefinedNetwork SoftwareDefinedNetwork `json:"softwareDefinedNetwork,omitempty"`

	PodCIDR      string       `json:"podCidr,omitempty"`
	ServiceCIDR  string       `json:"serviceCidr,omitempty"`
	MTUSize      MTUSize      `json:"mtuSize,omitempty"`
	OutboundType OutboundType `json:"outboundType,omitempty" mutable:"true"`

	APIServerPrivateEndpointIP string           `json:"privateEndpointIp,omitempty"`
	GatewayPrivateEndpointIP   string           `json:"gatewayPrivateEndpointIp,omitempty"`
	GatewayPrivateLinkID       string           `json:"gatewayPrivateLinkId,omitempty"`
	PreconfiguredNSG           PreconfiguredNSG `json:"preconfigureNSG,omitempty"`
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
	VMSize              VMSize           `json:"vmSize,omitempty"`
	SubnetID            string           `json:"subnetId,omitempty"`
	EncryptionAtHost    EncryptionAtHost `json:"encryptionAtHost,omitempty"`
	DiskEncryptionSetID string           `json:"diskEncryptionSetId,omitempty"`
}

// VMSize represents a VM size.
type VMSize string

// VMSize constants.
const (
	VMSizeStandardD2sV3  VMSize = "Standard_D2s_v3"
	VMSizeStandardD4sV3  VMSize = "Standard_D4s_v3"
	VMSizeStandardD8sV3  VMSize = "Standard_D8s_v3"
	VMSizeStandardD16sV3 VMSize = "Standard_D16s_v3"
	VMSizeStandardD32sV3 VMSize = "Standard_D32s_v3"

	VMSizeStandardD4sV4  VMSize = "Standard_D4s_v4"
	VMSizeStandardD8sV4  VMSize = "Standard_D8s_v4"
	VMSizeStandardD16sV4 VMSize = "Standard_D16s_v4"
	VMSizeStandardD32sV4 VMSize = "Standard_D32s_v4"
	VMSizeStandardD64sV4 VMSize = "Standard_D64s_v4"

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

// WorkerProfile represents a worker profile.
type WorkerProfile struct {
	Name                string           `json:"name,omitempty"`
	VMSize              VMSize           `json:"vmSize,omitempty"`
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
