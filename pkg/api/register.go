package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type ClusterManagerConfigurationConverter interface {
	ToExternal(*ClusterManagerConfiguration) (interface{}, error)
	ToExternalList([]*ClusterManagerConfiguration, string) (interface{}, error)
	ToInternal(interface{}, *ClusterManagerConfiguration) error
}
type ClusterManagerConfigurationStaticValidator interface {
	Static(interface{}, *ClusterManagerConfiguration) error
}
type OpenShiftClusterConverter interface {
	ToExternal(*OpenShiftCluster) interface{}
	ToExternalList([]*OpenShiftCluster, string) interface{}
	ToInternal(interface{}, *OpenShiftCluster)
}

type OpenShiftClusterStaticValidator interface {
	Static(interface{}, *OpenShiftCluster, string, string, bool, string) error
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
	OpenShiftClusterConverter                OpenShiftClusterConverter
	OpenShiftClusterStaticValidator          OpenShiftClusterStaticValidator
	OpenShiftClusterCredentialsConverter     OpenShiftClusterCredentialsConverter
	OpenShiftClusterAdminKubeconfigConverter OpenShiftClusterAdminKubeconfigConverter
	OpenShiftVersionConverter                OpenShiftVersionConverter
	OpenShiftVersionStaticValidator          OpenShiftVersionStaticValidator
	InstallVersionsConverter                 InstallVersionsConverter
	OperationList                            OperationList
}

// APIs is the map of registered API versions
var APIs = map[string]*Version{}
