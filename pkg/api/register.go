package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
const APIVersionKey = "api-version"

type OpenShiftClusterConverter interface {
	ToExternal(*OpenShiftCluster) interface{}
	ToExternalList([]*OpenShiftCluster, string) interface{}
	ToInternal(interface{}, *OpenShiftCluster)
	ExternalNoReadOnly(interface{})
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

type OpenShiftVersionStaticValidator interface {
	Static(interface{}, *OpenShiftVersion) error
}

type PlatformWorkloadIdentityRoleSetConverter interface {
	ToExternal(*PlatformWorkloadIdentityRoleSet) interface{}
	ToExternalList([]*PlatformWorkloadIdentityRoleSet) interface{}
	ToInternal(interface{}, *PlatformWorkloadIdentityRoleSet)
}

type PlatformWorkloadIdentityRoleSetStaticValidator interface {
	Static(interface{}, *PlatformWorkloadIdentityRoleSet) error
}

// Version is a set of endpoints implemented by each API version
type Version struct {
	OpenShiftClusterConverter                      OpenShiftClusterConverter
	OpenShiftClusterStaticValidator                OpenShiftClusterStaticValidator
	OpenShiftClusterCredentialsConverter           OpenShiftClusterCredentialsConverter
	OpenShiftClusterAdminKubeconfigConverter       OpenShiftClusterAdminKubeconfigConverter
	OpenShiftVersionConverter                      OpenShiftVersionConverter
	OpenShiftVersionStaticValidator                OpenShiftVersionStaticValidator
	PlatformWorkloadIdentityRoleSetConverter       PlatformWorkloadIdentityRoleSetConverter
	PlatformWorkloadIdentityRoleSetStaticValidator PlatformWorkloadIdentityRoleSetStaticValidator
	OperationList                                  OperationList
}

// APIs is the map of registered API versions
var APIs = map[string]*Version{}
