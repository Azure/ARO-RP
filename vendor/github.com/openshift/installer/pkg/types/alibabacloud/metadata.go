package alibabacloud

// Metadata contains Alibaba Cloud metadata (e.g. for uninstalling the cluster).
type Metadata struct {
	Region        string `json:"region"`
	ClusterDomain string `json:"clusterDomain"`
	PrivateZoneID string `json:"privateZoneID"`
}
