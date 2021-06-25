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
	SystemData SystemData                 `json:"systemData,omitempty"`
}

// OpenShiftClusterProperties represents an OpenShift cluster's properties.
type OpenShiftClusterProperties struct {
	ArchitectureVersion     ArchitectureVersion     `json:"architectureVersion"` // ArchitectureVersion is int so 0 is valid value to be returned
	ProvisioningState       ProvisioningState       `json:"provisioningState,omitempty"`
	LastProvisioningState   ProvisioningState       `json:"lastProvisioningState,omitempty"`
	FailedProvisioningState ProvisioningState       `json:"failedProvisioningState,omitempty"`
	LastAdminUpdateError    string                  `json:"lastAdminUpdateError,omitempty"`
	CreatedAt               time.Time               `json:"createdAt,omitempty"`
	CreatedBy               string                  `json:"createdBy,omitempty"`
	ProvisionedBy           string                  `json:"provisionedBy,omitempty"`
	ClusterProfile          ClusterProfile          `json:"clusterProfile,omitempty"`
	ConsoleProfile          ConsoleProfile          `json:"consoleProfile,omitempty"`
	ServicePrincipalProfile ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`
	NetworkProfile          NetworkProfile          `json:"networkProfile,omitempty"`
	MasterProfile           MasterProfile           `json:"masterProfile,omitempty"`
	WorkerProfiles          []WorkerProfile         `json:"workerProfiles,omitempty"`
	APIServerProfile        APIServerProfile        `json:"apiserverProfile,omitempty"`
	IngressProfiles         []IngressProfile        `json:"ingressProfiles,omitempty"`
	Install                 *Install                `json:"install,omitempty"`
	StorageSuffix           string                  `json:"storageSuffix,omitempty"`
	RegistryProfiles        []RegistryProfile       `json:"registryProfiles,omitempty"`
	InfraID                 string                  `json:"infraId,omitempty"`
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

// ClusterProfile represents a cluster profile.
type ClusterProfile struct {
	Domain          string `json:"domain,omitempty"`
	Version         string `json:"version,omitempty"`
	ResourceGroupID string `json:"resourceGroupId,omitempty"`
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

// NetworkProfile represents a network profile.
type NetworkProfile struct {
	PodCIDR     string `json:"podCidr,omitempty"`
	ServiceCIDR string `json:"serviceCidr,omitempty"`

	APIServerPrivateEndpointIP string `json:"privateEndpointIp,omitempty"`
}

// MasterProfile represents a master profile.
type MasterProfile struct {
	VMSize   VMSize `json:"vmSize,omitempty"`
	SubnetID string `json:"subnetId,omitempty"`
}

// VMSize represents a VM size.
type VMSize string

// VMSize constants.
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

	VMSizeStandardE4sV3   VMSize = "Standard_E4s_v3"
	VMSizeStandardE8sV3   VMSize = "Standard_E8s_v3"
	VMSizeStandardE16sV3  VMSize = "Standard_E16s_v3"
	VMSizeStandardE32sV3  VMSize = "Standard_E32s_v3"
	VMSizeStandardE64isV3 VMSize = "Standard_E64is_v3"
	VMSizeStandardE64iV3  VMSize = "Standard_E64i_v3"

	VMSizeStandardF4sV2  VMSize = "Standard_F4s_v2"
	VMSizeStandardF8sV2  VMSize = "Standard_F8s_v2"
	VMSizeStandardF16sV2 VMSize = "Standard_F16s_v2"
	VMSizeStandardF32sV2 VMSize = "Standard_F32s_v2"
	VMSizeStandardF72sV2 VMSize = "Standard_F72s_v2"

	VMSizeStandardM128ms VMSize = "Standard_M128ms"
	VMSizeStandardG5     VMSize = "Standard_G5"
	VMSizeStandardGS5    VMSize = "Standard_GS5"
)

// WorkerProfile represents a worker profile.
type WorkerProfile struct {
	Name       string `json:"name,omitempty"`
	VMSize     VMSize `json:"vmSize,omitempty"`
	DiskSizeGB int    `json:"diskSizeGB,omitempty"`
	SubnetID   string `json:"subnetId,omitempty"`
	Count      int    `json:"count,omitempty"`
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
