package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const APIVersionKey = "api-version"

type OpenShiftClusterConverter interface {
	ToExternal(*OpenShiftCluster) any
	ToExternalList([]*OpenShiftCluster, string) any
	ToInternal(any, *OpenShiftCluster)
	ExternalNoReadOnly(any)
}

type OpenShiftClusterStaticValidator interface {
	Static(any, *OpenShiftCluster, string, string, bool, ArchitectureVersion, string) error
}

type OpenShiftClusterCredentialsConverter interface {
	ToExternal(*OpenShiftCluster) any
}

type OpenShiftClusterAdminKubeconfigConverter interface {
	ToExternal(*OpenShiftCluster) any
}

type OpenShiftVersionConverter interface {
	ToExternal(*OpenShiftVersion) any
	ToExternalList([]*OpenShiftVersion) any
	ToInternal(any, *OpenShiftVersion)
}

type OpenShiftVersionStaticValidator interface {
	Static(any, *OpenShiftVersion) error
}

type PlatformWorkloadIdentityRoleSetConverter interface {
	ToExternal(*PlatformWorkloadIdentityRoleSet) any
	ToExternalList([]*PlatformWorkloadIdentityRoleSet) any
	ToInternal(any, *PlatformWorkloadIdentityRoleSet)
}

type PlatformWorkloadIdentityRoleSetStaticValidator interface {
	Static(any, *PlatformWorkloadIdentityRoleSet) error
}

type MaintenanceManifestConverter interface {
	ToExternal(doc *MaintenanceManifestDocument, clusterNamespaced bool) any
	ToExternalList(docs []*MaintenanceManifestDocument, nextLink string, clusterNamespaced bool) any
	ToInternal(any, *MaintenanceManifestDocument)
}

type MaintenanceManifestStaticValidator interface {
	Static(any, *MaintenanceManifestDocument) error
}

type MaintenanceScheduleConverter interface {
	ToExternal(doc *MaintenanceScheduleDocument) any
	ToExternalList(docs []*MaintenanceScheduleDocument, nextLink string) any
	ToInternal(any, *MaintenanceScheduleDocument)
}

type MaintenanceScheduleStaticValidator interface {
	Static(any, *MaintenanceScheduleDocument) error
}

type BillingDocumentConverter interface {
	ToExternal(doc *BillingDocument) any
	ToExternalList(docs []*BillingDocument, nextLink string) any
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
	MaintenanceManifestConverter                   MaintenanceManifestConverter
	MaintenanceManifestStaticValidator             MaintenanceManifestStaticValidator
	MaintenanceScheduleConverter                   MaintenanceScheduleConverter
	MaintenanceScheduleStaticValidator             MaintenanceScheduleStaticValidator
	BillingDocumentConverter                       BillingDocumentConverter
}

// APIs is the map of registered API versions
var APIs = map[string]*Version{}
