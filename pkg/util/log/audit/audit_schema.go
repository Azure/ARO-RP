package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// AuditPayload is the log payload that will be persisted.
// It has all the fields defined in IFxAudit Part-A and Part-B schema.
// String fields are declared as pointers-to-string because IFxAudit wants
// nil, not empty string values.
type AuditPayload struct {
	// Part-A
	EnvVer                 float64 `json:"env_ver"`
	EnvName                *string `json:"env_name"`
	EnvTime                *string `json:"env_time"`
	EnvEpoch               *string `json:"env_epoch"`
	EnvSeqNum              uint64  `json:"env_seqNum"`
	EnvPopSample           float64 `json:"env_popSample"`
	EnvIKey                *string `json:"env_iKey"`
	EnvFlags               int     `json:"env_flags"`
	EnvCV                  *string `json:"env_cv"`
	EnvOS                  *string `json:"env_os"`
	EnvOSVer               *string `json:"env_osVer"`
	EnvAppId               *string `json:"env_appId"`
	EnvAppVer              *string `json:"env_appVer"`
	EnvCloudVer            float64 `json:"env_cloud_ver"`
	EnvCloudName           *string `json:"env_cloud_name"`
	EnvCloudRole           *string `json:"env_cloud_role"`
	EnvCloudRoleVer        *string `json:"env_cloud_roleVer"`
	EnvCloudRoleInstance   *string `json:"env_cloud_roleInstance"`
	EnvCloudEnvironment    *string `json:"env_cloud_environment"`
	EnvCloudLocation       *string `json:"env_cloud_location"`
	EnvCloudDeploymentUnit *string `json:"env_cloud_deploymentUnit"`

	// Part-B
	CallerIdentities []*CallerIdentity `json:"CallerIdentities"`
	Category         Category          `json:"Category"`
	NCloud           *string           `json:"nCloud"`
	OperationName    *string           `json:"OperationName"`
	Result           Result            `json:"Result"`
	RequestID        *string           `json:"requestId"`
	TargetResources  []*TargetResource `json:"TargetResources"`
}

// CallerIdentity has identity information on the entity that invoke the
// operation described in the audit log.
type CallerIdentity struct {
	CallerDisplayName   string             `json:"CallerDisplayName"`
	CallerIdentityType  CallerIdentityType `json:"CallerIdentityType"`
	CallerIdentityValue string             `json:"CallerIdentityValue"`
	CallerIPAddress     string             `json:"CallerIpAddress"`
}

// CallerIdentityType represents the type of identity used in an auditable event.
type CallerIdentityType string

// Category provides information for the category of the operation.
type Category string

// Result provides information on the result of the operation.
type Result struct {
	ResultType        string `json:"ResultType"`
	ResultDescription string `json:"ResultDescription"`
}

// ResultType indicates the outcome of the operation.
type ResultType string

// TargetResource has identity information on the entity affected by the
// operation described in the audit log.
type TargetResource struct {
	TargetResourceType string `json:"TargetResourceType"`
	TargetResourceName string `json:"TargetResourceName"`
}

const (
	CallerIdentityTypeUPN            = "UPN"
	CallerIdentityTypePUID           = "PUID"
	CallerIdentityTypeObjectID       = "ObjectID"
	CallerIdentityTypeCertificate    = "Certificate"
	CallerIdentityTypeClaim          = "Claim"
	CallerIdentityTypeUsername       = "Username"
	CallerIdentityTypeKeyName        = "KeyName"
	CallerIdentityTypeApplicationID  = "ApplicationID"
	CallerIdentityTypeSubscriptionID = "SubscriptionID"

	CategoryAuthentication        = "Authentication"
	CategoryAuthorization         = "Authorization"
	CategoryUserManagement        = "UserManagement"
	CategoryGroupManagement       = "GroupManagement"
	CategoryRoleManagement        = "RoleManagement"
	CategoryApplicationManagement = "ApplicationManagement"
	CategoryKeyManagement         = "KeyManagement"
	CategoryDirectoryManagement   = "DirectoryManagement"
	CategoryResourceManagement    = "ResourceManagement"
	CategoryPolicyManagement      = "PolicyManagement"
	CategoryDeviceManagement      = "DeviceManagement"
	CategoryEntitlementManagement = "EntitlementManagement"
	CategoryPasswordManagement    = "PasswordManagement"
	CategoryObjectManagement      = "ObjectManagement"
	CategoryIdentityProtection    = "IdentityProtection"
	CategoryOther                 = "Other"

	ResultTypeSuccess     = "Success"
	ResultTypeFail        = "Fail"
	ResultTypeTimeout     = "Timeout"
	ResultTypeClientError = "Client Error"
)
