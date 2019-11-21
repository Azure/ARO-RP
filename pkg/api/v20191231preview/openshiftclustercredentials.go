package v20191231preview

// OpenShiftClusterCredentials represents an OpenShift cluster's credentials
type OpenShiftClusterCredentials struct {
	// The password for the kubeadmin user
	KubeadminPassword string `json:"kubeadminPassword,omitempty"`
}
