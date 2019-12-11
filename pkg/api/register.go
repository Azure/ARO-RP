package api

import (
	"context"

	"github.com/Azure/go-autorest/autorest"
)

// OpenShiftClusterToExternal is implemented by all APIs - it enables conversion
// of the internal OpenShiftCluster representation to the API-specific versioned
// external representation
type OpenShiftClusterToExternal interface {
	OpenShiftClusterToExternal(*OpenShiftCluster) interface{}
}

// OpenShiftClustersToExternal is implemented by APIs that can convert multiple
// internal OpenShiftCluster representations to the API-specific versioned
// external representation
type OpenShiftClustersToExternal interface {
	OpenShiftClustersToExternal([]*OpenShiftCluster) interface{}
}

// OpenShiftClusterToInternal is implemented by APIs that can convert their
// API-specific versioned external representation to the internal
// OpenShiftCluster representation.  It also includes validators
type OpenShiftClusterToInternal interface {
	OpenShiftClusterToInternal(interface{}, *OpenShiftCluster)
	ValidateOpenShiftCluster(string, string, interface{}, *OpenShiftCluster) error
	ValidateOpenShiftClusterDynamic(context.Context, func(string, string) (autorest.Authorizer, error), *OpenShiftCluster) error
}

// APIs is the map of registered external APIs
var APIs = map[string]map[string]interface{}{}
