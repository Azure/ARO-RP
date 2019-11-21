package v20191231preview

import (
	"github.com/jim-minter/rp/pkg/api"
)

const (
	resourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceType              = "openShiftClusters"
)

func init() {
	api.APIs[api.APIVersionType{APIVersion: "2019-12-31-preview", Type: "OpenShiftCluster"}] = func(oc *api.OpenShiftCluster) api.External {
		return OpenShiftClusterToExternal(oc)
	}

	api.APIs[api.APIVersionType{APIVersion: "2019-12-31-preview", Type: "OpenShiftClusterCredentials"}] = func(oc *api.OpenShiftCluster) api.External {
		return OpenShiftClusterCredentialsToExternal(oc)
	}
}
