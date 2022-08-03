package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type ClusterManagerConverter interface {
	ToExternal(*ClusterManagerConfiguration) (interface{}, error)
	ToExternalList([]*ClusterManagerConfiguration, string) (interface{}, error)
	ToInternal(interface{}, *ClusterManagerConfiguration) error
}
type ClusterManagerStaticValidator interface {
	Static(interface{}, *ClusterManagerConfiguration) error
}
type OpenShiftClusterConverter interface {
	ToExternal(*OpenShiftCluster) interface{}
	ToExternalList([]*OpenShiftCluster, string) interface{}
	ToInternal(interface{}, *OpenShiftCluster)
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

type OpenShiftVersionConverter interface {
	ToExternal(*OpenShiftVersion) interface{}
	ToExternalList([]*OpenShiftVersion) interface{}
	ToInternal(interface{}, *OpenShiftVersion)
}

type InstallVersionsConverter interface {
	ToExternal(*InstallVersions) interface{}
}

type OpenShiftVersionStaticValidator interface {
	Static(interface{}, *OpenShiftVersion) error
}

// Version is a set of endpoints implemented by each API version
type Version struct {
	ClusterManagerConverter                  func() ClusterManagerConverter
	ClusterManagerStaticValidator            func() ClusterManagerStaticValidator
	OpenShiftClusterConverter                func() OpenShiftClusterConverter
	OpenShiftClusterStaticValidator          func(string, string, bool, string) OpenShiftClusterStaticValidator
	OpenShiftClusterCredentialsConverter     func() OpenShiftClusterCredentialsConverter
	OpenShiftClusterAdminKubeconfigConverter func() OpenShiftClusterAdminKubeconfigConverter
	OpenShiftVersionConverter                func() OpenShiftVersionConverter
	OpenShiftVersionStaticValidator          func() OpenShiftVersionStaticValidator
	InstallVersionsConverter                 func() InstallVersionsConverter
	OperationList                            OperationList
}

// APIs is the map of registered API versions
var APIs = map[string]*Version{}
