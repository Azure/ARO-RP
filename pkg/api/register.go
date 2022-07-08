package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type OpenShiftClusterConverter interface {
	ToExternal(*OpenShiftCluster) interface{}
	ToExternalList([]*OpenShiftCluster, string) interface{}
	ToInternal(interface{}, *OpenShiftCluster)
}

type OpenShiftClusterDocumentConverter interface {
	ToExternal(*OpenShiftClusterDocument) interface{}
	ToExternalList([]*OpenShiftClusterDocument, string) interface{}
}

type OpenShiftClusterStaticValidator interface {
	Static(interface{}, *OpenShiftCluster) error
}

type OpenShiftClusterCredentialsConverter interface {
	ToExternal(*OpenShiftCluster) interface{}
}

type OpenShiftClusterAdminKubeconfigConverter interface {
	ToExternal(*OpenShiftCluster) interface{}
}

// Version is a set of endpoints implemented by each API version
type Version struct {
	OpenShiftClusterConverter                func() OpenShiftClusterConverter
	OpenShiftClusterDocumentConverter        func() OpenShiftClusterDocumentConverter
	OpenShiftClusterStaticValidator          func(string, string, bool, string) OpenShiftClusterStaticValidator
	OpenShiftClusterCredentialsConverter     func() OpenShiftClusterCredentialsConverter
	OpenShiftClusterAdminKubeconfigConverter func() OpenShiftClusterAdminKubeconfigConverter
}

// APIs is the map of registered API versions
var APIs = map[string]*Version{}
