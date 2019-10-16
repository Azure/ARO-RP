package v20191231preview

// OpenShiftClusterCredentials represents an OpenShift cluster's credentials
type OpenShiftClusterCredentials struct {
	KubeadminPassword string `json:"kubeadminPassword,omitempty"`
}
