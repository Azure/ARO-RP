package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
const APIVersionKey = "api-version"

type OpenShiftClusterConverter interface {
	ToExternal(*OpenShiftCluster) interface{}
	ToExternalList([]*OpenShiftCluster, string) interface{}
	ToInternal(interface{}, *OpenShiftCluster)
}

type OpenShiftClusterStaticValidator interface {
	Static(interface{}, *OpenShiftCluster, string, string, bool, string) error
}

type ClusterManagerStaticValidator interface {
	Static(string, map[string]string) error
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

type SyncSetConverter interface {
	ToExternal(*SyncSet) interface{}
	ToExternalList([]*SyncSet) interface{}
	ToInternal(interface{}, *SyncSet)
}

type MachinePoolConverter interface {
	ToExternal(*MachinePool) interface{}
	ToExternalList([]*MachinePool) interface{}
	ToInternal(interface{}, *MachinePool)
}

type SyncIdentityProviderConverter interface {
	ToExternal(*SyncIdentityProvider) interface{}
	ToExternalList([]*SyncIdentityProvider) interface{}
	ToInternal(interface{}, *SyncIdentityProvider)
}

type SecretConverter interface {
	ToExternal(*Secret) interface{}
	ToExternalList([]*Secret) interface{}
	ToInternal(interface{}, *Secret)
}

// Version is a set of endpoints implemented by each API version
type Version struct {
	OpenShiftClusterConverter                OpenShiftClusterConverter
	OpenShiftClusterStaticValidator          OpenShiftClusterStaticValidator
	OpenShiftClusterCredentialsConverter     OpenShiftClusterCredentialsConverter
	OpenShiftClusterAdminKubeconfigConverter OpenShiftClusterAdminKubeconfigConverter
	OpenShiftVersionConverter                OpenShiftVersionConverter
	OpenShiftVersionStaticValidator          OpenShiftVersionStaticValidator
	OperationList                            OperationList
	SyncSetConverter                         SyncSetConverter
	MachinePoolConverter                     MachinePoolConverter
	SyncIdentityProviderConverter            SyncIdentityProviderConverter
	SecretConverter                          SecretConverter
	ClusterManagerStaticValidator            ClusterManagerStaticValidator
}

// APIs is the map of registered API versions
var APIs = map[string]*Version{}
