package azure

// Metadata contains Azure metadata (e.g. for uninstalling the cluster).
type Metadata struct {
	ResourceGroup string `json:"resourceGroup"`
}
