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

type ClusterManagerStaticValidator interface {
	Static(string, string) error
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

type MaintenanceManifestConverter interface {
	ToExternal(doc *MaintenanceManifestDocument, clusterNamespaced bool) interface{}
	ToExternalList(docs []*MaintenanceManifestDocument, nextLink string, clusterNamespaced bool) interface{}
	ToInternal(interface{}, *MaintenanceManifestDocument)
}

type MaintenanceManifestStaticValidator interface {
	Static(interface{}, *MaintenanceManifestDocument) error
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
	SyncSetConverter                               SyncSetConverter
	MachinePoolConverter                           MachinePoolConverter
	SyncIdentityProviderConverter                  SyncIdentityProviderConverter
	SecretConverter                                SecretConverter
	ClusterManagerStaticValidator                  ClusterManagerStaticValidator
	MaintenanceManifestConverter                   MaintenanceManifestConverter
	MaintenanceManifestStaticValidator             MaintenanceManifestStaticValidator
}

// APIs is the map of registered API versions
var APIs = map[string]*Version{}
