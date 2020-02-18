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
}

// OpenShiftCluster represents an Azure Red Hat OpenShift cluster.
type OpenShiftCluster struct {
	ID         string            `json:"id,omitempty" mutable:"case"`
	Name       string            `json:"name,omitempty" mutable:"case"`
	Type       string            `json:"type,omitempty" mutable:"case"`
	Location   string            `json:"location,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties Properties        `json:"properties,omitempty"`
}

// Properties represents an OpenShift cluster's properties.
type Properties struct {
	ProvisioningState       ProvisioningState       `json:"provisioningState,omitempty"`
	FailedProvisioningState ProvisioningState       `json:"failedProvisioningState,omitempty"`
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
	TenantID string `json:"tenantId,omitempty"`
	ClientID string `json:"clientId,omitempty"`
}

// NetworkProfile represents a network profile.
type NetworkProfile struct {
	PodCIDR     string `json:"podCidr,omitempty"`
	ServiceCIDR string `json:"serviceCidr,omitempty"`

	PrivateEndpointIP string `json:"privateEndpointIp,omitempty"`
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
	VMSizeStandardD4sV3 VMSize = "Standard_D4s_v3"
	VMSizeStandardD8sV3 VMSize = "Standard_D8s_v3"
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
	InstallPhaseDeployStorage InstallPhase = iota
	InstallPhaseDeployResources
	InstallPhaseRemoveBootstrap
)
