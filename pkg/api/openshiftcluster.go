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
	ID         string            `json:"id,omitempty"`
	Name       string            `json:"name,omitempty"`
	Type       string            `json:"type,omitempty"`
	Location   string            `json:"location,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties Properties        `json:"properties,omitempty"`
}

// SecureBytes represents an encrypted []byte
type SecureBytes []byte

// SecureString represents an encrypted string
type SecureString string

// Properties represents an OpenShift cluster's properties
type Properties struct {
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

	ProvisioningState       ProvisioningState `json:"provisioningState,omitempty"`
	FailedProvisioningState ProvisioningState `json:"failedProvisioningState,omitempty"`

	ClusterProfile ClusterProfile `json:"clusterProfile,omitempty"`

	ConsoleProfile ConsoleProfile `json:"consoleProfile,omitempty"`

	ServicePrincipalProfile ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`

	NetworkProfile NetworkProfile `json:"networkProfile,omitempty"`

	MasterProfile MasterProfile `json:"masterProfile,omitempty"`

	WorkerProfiles []WorkerProfile `json:"workerProfiles,omitempty"`

	APIServerProfile APIServerProfile `json:"apiserverProfile,omitempty"`

	IngressProfiles []IngressProfile `json:"ingressProfiles,omitempty"`

	// Install is non-nil only when an install is in progress
	Install *Install `json:"install,omitempty"`

	StorageSuffix string `json:"storageSuffix,omitempty"`

	SSHKey            SecureBytes  `json:"sshKey,omitempty"`
	AdminKubeconfig   SecureBytes  `json:"adminKubeconfig,omitempty"`
	KubeadminPassword SecureString `json:"kubeadminPassword,omitempty"`
}

// ProvisioningState represents a provisioning state
type ProvisioningState string

// ProvisioningState constants
const (
	ProvisioningStateCreating  ProvisioningState = "Creating"
	ProvisioningStateUpdating  ProvisioningState = "Updating"
	ProvisioningStateDeleting  ProvisioningState = "Deleting"
	ProvisioningStateSucceeded ProvisioningState = "Succeeded"
	ProvisioningStateFailed    ProvisioningState = "Failed"
)

// ClusterProfile represents a cluster profile.
type ClusterProfile struct {
	MissingFields

	Domain          string `json:"domain,omitempty"`
	Version         string `json:"version,omitempty"`
	ResourceGroupID string `json:"resourceGroupId,omitempty"`
}

// ConsoleProfile represents a console profile.
type ConsoleProfile struct {
	MissingFields

	URL string `json:"url,omitempty"`
}

// ServicePrincipalProfile represents a service principal profile.
type ServicePrincipalProfile struct {
	MissingFields

	TenantID     string       `json:"tenantId,omitempty"`
	ClientID     string       `json:"clientId,omitempty"`
	ClientSecret SecureString `json:"clientSecret,omitempty"`
}

// NetworkProfile represents a network profile
type NetworkProfile struct {
	MissingFields

	PodCIDR     string `json:"podCidr,omitempty"`
	ServiceCIDR string `json:"serviceCidr,omitempty"`

	PrivateEndpointIP string `json:"privateEndpointIp,omitempty"`
}

// MasterProfile represents a master profile
type MasterProfile struct {
	MissingFields

	VMSize   VMSize `json:"vmSize,omitempty"`
	SubnetID string `json:"subnetId,omitempty"`
}

// VMSize represents a VM size
type VMSize string

// VMSize constants
const (
	VMSizeStandardD2sV3 VMSize = "Standard_D2s_v3"
	VMSizeStandardD4sV3 VMSize = "Standard_D4s_v3"
	VMSizeStandardD8sV3 VMSize = "Standard_D8s_v3"
)

// WorkerProfile represents a worker profile
type WorkerProfile struct {
	MissingFields

	Name       string `json:"name,omitempty"`
	VMSize     VMSize `json:"vmSize,omitempty"`
	DiskSizeGB int    `json:"diskSizeGB,omitempty"`
	SubnetID   string `json:"subnetId,omitempty"`
	Count      int    `json:"count,omitempty"`
}

// APIServerProfile represents an API server profile
type APIServerProfile struct {
	MissingFields

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

// IngressProfile represents an ingress profile
type IngressProfile struct {
	MissingFields

	Name       string     `json:"name,omitempty"`
	Visibility Visibility `json:"visibility,omitempty"`
	IP         string     `json:"ip,omitempty"`
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
	InstallPhaseDeployStorage InstallPhase = iota
	InstallPhaseDeployResources
	InstallPhaseRemoveBootstrap
)
