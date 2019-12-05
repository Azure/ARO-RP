package v20191231preview

// OpenShiftClusterList represents a list of OpenShift clusters.
type OpenShiftClusterList struct {
	// The list of OpenShift clusters.
	OpenShiftClusters []*OpenShiftCluster `json:"value"`
}

// OpenShiftCluster represents an Azure Red Hat OpenShift cluster.
type OpenShiftCluster struct {
	// The resource ID (immutable).
	ID string `json:"id,omitempty" mutable:"case"`

	// The resource name (immutable).
	Name string `json:"name,omitempty" mutable:"case"`

	// The resource type (immutable).
	Type string `json:"type,omitempty" mutable:"case"`

	// The resource location (immutable).
	Location string `json:"location,omitempty"`

	// The resource tags.
	Tags Tags `json:"tags,omitempty" mutable:"true"`

	// The cluster properties.
	Properties Properties `json:"properties,omitempty"`
}

// Tags represents an OpenShift cluster's tags.
type Tags map[string]string

// Properties represents an OpenShift cluster's properties.
type Properties struct {
	// The cluster provisioning state (immutable).
	ProvisioningState ProvisioningState `json:"provisioningState,omitempty"`

	// The cluster service principal profile.
	ServicePrincipalProfile ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`

	// The cluster network profile.
	NetworkProfile NetworkProfile `json:"networkProfile,omitempty"`

	// The cluster master profile.
	MasterProfile MasterProfile `json:"masterProfile,omitempty"`

	// The cluster worker profiles.
	WorkerProfiles []WorkerProfile `json:"workerProfiles,omitempty"`

	// The URL to access the cluster API server (immutable).
	APIServerURL string `json:"apiserverUrl,omitempty"`

	// The URL to access the cluster console (immutable).
	ConsoleURL string `json:"consoleUrl,omitempty"`
}

// ProvisioningState represents a provisioning state.
type ProvisioningState string

// ProvisioningState constants
const (
	ProvisioningStateCreating  ProvisioningState = "Creating"
	ProvisioningStateUpdating  ProvisioningState = "Updating"
	ProvisioningStateDeleting  ProvisioningState = "Deleting"
	ProvisioningStateSucceeded ProvisioningState = "Succeeded"
	ProvisioningStateFailed    ProvisioningState = "Failed"
)

// ServicePrincipalProfile represents a service principal profile.
type ServicePrincipalProfile struct {
	// The client ID used for the cluster
	ClientID string `json:"clientId,omitempty"`

	// The client secret used for the cluster
	ClientSecret string `json:"clientSecret,omitempty"`
}

// NetworkProfile represents a network profile.
type NetworkProfile struct {
	// The CIDR used for OpenShift/Kubernetes Pods (immutable).
	PodCIDR string `json:"podCidr,omitempty"`

	// The CIDR used for OpenShift/Kubernetes Services (immutable).
	ServiceCIDR string `json:"serviceCidr,omitempty"`
}

// MasterProfile represents a master profile.
type MasterProfile struct {
	// The size of the master VMs (immutable).
	VMSize VMSize `json:"vmSize,omitempty"`

	// The Azure resource ID of the worker subnet (immutable).
	SubnetID string `json:"subnetId,omitempty"`
}

// VMSize represents a VM size.
type VMSize string

// VMSize constants
const (
	VMSizeStandardD2sV3 VMSize = "Standard_D2s_v3"
	VMSizeStandardD4sV3 VMSize = "Standard_D4s_v3"
	VMSizeStandardD8sV3 VMSize = "Standard_D8s_v3"
)

// WorkerProfile represents a worker profile.
type WorkerProfile struct {
	// The worker profile name.  Must be "worker" (immutable).
	Name string `json:"name,omitempty"`

	// The size of the worker VMs (immutable).
	VMSize VMSize `json:"vmSize,omitempty"`

	// The disk size of the worker VMs.  Must be 128 or greater (immutable).
	DiskSizeGB int `json:"diskSizeGB,omitempty"`

	// The Azure resource ID of the worker subnet (immutable).
	SubnetID string `json:"subnetId,omitempty"`

	// The number of worker VMs.  Must be between 3 and 20.
	Count int `json:"count,omitempty" mutable:"true"`
}
