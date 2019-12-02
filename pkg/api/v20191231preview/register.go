package v20191231preview

import (
	"context"

	"github.com/Azure/go-autorest/autorest"

	"github.com/jim-minter/rp/pkg/api"
)

const (
	resourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceType              = "openShiftClusters"
)

type openShiftCluster struct{}

var _ api.OpenShiftClusterToExternal = (*openShiftCluster)(nil)
var _ api.OpenShiftClustersToExternal = (*openShiftCluster)(nil)
var _ api.OpenShiftClusterToInternal = (*openShiftCluster)(nil)

func (openShiftCluster) OpenShiftClusterToExternal(oc *api.OpenShiftCluster) interface{} {
	return openShiftClusterToExternal(oc)
}

func (openShiftCluster) OpenShiftClustersToExternal(ocs []*api.OpenShiftCluster) interface{} {
	return openShiftClustersToExternal(ocs)
}

func (openShiftCluster) OpenShiftClusterToInternal(oc interface{}, out *api.OpenShiftCluster) {
	openShiftClusterToInternal(oc.(*OpenShiftCluster), out)
}

func (openShiftCluster) ValidateOpenShiftCluster(location, resourceID string, oc interface{}, current *api.OpenShiftCluster) error {
	return validateOpenShiftCluster(location, resourceID, oc.(*OpenShiftCluster), openShiftClusterToExternal(current))
}

func (openShiftCluster) ValidateOpenShiftClusterDynamic(ctx context.Context, fpAuthorizer autorest.Authorizer, oc *api.OpenShiftCluster) error {
	return validateOpenShiftClusterDynamic(ctx, fpAuthorizer, oc)
}

type openShiftClusterCredentials struct{}

var _ api.OpenShiftClusterToExternal = (*openShiftClusterCredentials)(nil)

func (openShiftClusterCredentials) OpenShiftClusterToExternal(oc *api.OpenShiftCluster) interface{} {
	return openShiftClusterCredentialsToExternal(oc)
}

func init() {
	api.APIs["2019-12-31-preview"] = map[string]interface{}{
		"OpenShiftCluster":            &openShiftCluster{},
		"OpenShiftClusterCredentials": &openShiftClusterCredentials{},
	}
}
