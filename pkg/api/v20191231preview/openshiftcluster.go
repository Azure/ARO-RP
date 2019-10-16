package v20191231preview

// OpenShiftCluster represents an OpenShift cluster
type OpenShiftCluster struct {
	ID         string     `json:"id,omitempty"`       // GET only
	Name       string     `json:"name,omitempty"`     // GET only
	Type       string     `json:"type,omitempty"`     // GET only
	Location   string     `json:"location,omitempty"` // r/o
	Tags       Tags       `json:"tags,omitempty"`
	Properties Properties `json:"properties,omitempty"`
}

// Tags represents an OpenShift cluster's tags
type Tags map[string]string

// Properties represents an OpenShift cluster's properties
type Properties struct {
	ProvisioningState ProvisioningState `json:"provisioningState,omitempty"` // r/o

	PullSecret []byte `json:"pullSecret,omitempty"` // w/o

	NetworkProfile NetworkProfile `json:"networkProfile,omitempty"`

	MasterProfile MasterProfile `json:"masterProfile,omitempty"`

	WorkerProfiles []WorkerProfile `json:"workerProfiles,omitempty"`

	APIServerURL string `json:"apiserverURL,omitempty"` // r/o
	ConsoleURL   string `json:"consoleURL,omitempty"`   // r/o
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

// NetworkProfile represents a network profile
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
