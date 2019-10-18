package api

import (
	"crypto/rsa"
	"time"
)

// OpenShiftCluster represents an OpenShift cluster
type OpenShiftCluster struct {
	MissingFields

	ID         string            `json:"id,omitempty"`
	Name       string            `json:"name,omitempty"`
	Type       string            `json:"type,omitempty"`
	Location   string            `json:"location,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties Properties        `json:"properties,omitempty"`
}

// Properties represents an OpenShift cluster's properties
type Properties struct {
	MissingFields

	ProvisioningState ProvisioningState `json:"provisioningState,omitempty"`

	PullSecret []byte `json:"pullSecret,omitempty"` // w/o

	NetworkProfile NetworkProfile `json:"networkProfile,omitempty"`

	MasterProfile MasterProfile `json:"masterProfile,omitempty"`

	WorkerProfiles []WorkerProfile `json:"workerProfiles,omitempty"`

	APIServerURL string `json:"apiserverURL,omitempty"` // r/o
	ConsoleURL   string `json:"consoleURL,omitempty"`   // r/o

	Installation *Installation `json:"installation,omitempty"`

	ResourceGroup string `json:"resourceGroup,omitempty"`
	StorageSuffix string `json:"storageSuffix,omitempty"`
	ClusterID     string `json:"clusterId,omitempty"`

	SSHKey            *rsa.PrivateKey `json:"sshKey,omitempty"`
	AdminKubeconfig   []byte          `json:"adminKubeconfig,omitempty"`
	KubeadminPassword string          `json:"kubeadminPassword,omitempty"`
}

// ProvisioningState represents a provisioning state
type ProvisioningState string

// ProvisioningState constants
const (
	ProvisioningStateUpdating  ProvisioningState = "Updating"
	ProvisioningStateDeleting  ProvisioningState = "Deleting"
	ProvisioningStateSucceeded ProvisioningState = "Succeeded"
	ProvisioningStateFailed    ProvisioningState = "Failed"
)

type NetworkProfile struct {
	VNetCIDR    string `json:"vnetCidr,omitempty"`
	PodCIDR     string `json:"podCidr,omitempty"`
	ServiceCIDR string `json:"serviceCidr,omitempty"`
}

// MasterProfile represents a master profile
type MasterProfile struct {
	VMSize VMSize `json:"vmSize,omitempty"`
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
	Name       string `json:"name,omitempty"`
	VMSize     VMSize `json:"vmSize,omitempty"`
	DiskSizeGB int    `json:"diskSizeGB,omitempty"`
	Count      int    `json:"count,omitempty"`
}

// Installation represents an installation process
type Installation struct {
	MissingFields

	Now   time.Time         `json:"now,omitempty"`
	Phase InstallationPhase `json:"phase"`
}

// InstallationPhase represents an installation phase
type InstallationPhase int

// InstallationPhase constants
const (
	InstallationPhaseDeployStorage InstallationPhase = iota
	InstallationPhaseDeployResources
	InstallationPhaseRemoveBootstrap
)
