package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type OpenShiftClusterConverter interface {
	ToExternal(*OpenShiftCluster) interface{}
	ToExternalList([]*OpenShiftCluster) interface{}
	ToInternal(interface{}, *OpenShiftCluster)
}

type OpenShiftClusterStaticValidator interface {
	Static(interface{}, *OpenShiftCluster) error
}

type OpenShiftClusterCredentialsConverter interface {
	ToExternal(*OpenShiftCluster) interface{}
}

// Version is a set of endpoints implemented by each API version
type Version struct {
	OpenShiftClusterConverter            func() OpenShiftClusterConverter
	OpenShiftClusterStaticValidator      func(string, string) OpenShiftClusterStaticValidator
	OpenShiftClusterCredentialsConverter func() OpenShiftClusterCredentialsConverter
}

// APIs is the map of registered API versions
var APIs = map[string]*Version{}
