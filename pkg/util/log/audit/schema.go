package audit

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Payload is the IFxAudit log payload that will be sent to Geneva. It has
// all the required and optional fields defined in IFxAudit Part-A and Part-B
// schema.
//
// Fields that are marked as optional or "required when applicable" in the
// schema are marked with the omitempty tag. Fields that are marked as "unused"
// are not included.
type Payload struct {
	// Part-A
	EnvVer                 float64 `json:"env_ver"`
	EnvName                string  `json:"env_name"`
	EnvTime                string  `json:"env_time" deep:"-"`
	EnvEpoch               string  `json:"env_epoch,omitempty" deep:"-"`
	EnvSeqNum              uint64  `json:"env_seqNum,omitempty" deep:"-"`
	EnvIKey                string  `json:"env_iKey,omitempty"`
	EnvFlags               int     `json:"env_flags,omitempty"`
	EnvAppID               string  `json:"env_appId"`
	EnvAppVer              string  `json:"env_appVer,omitempty"`
	EnvCV                  string  `json:"env_cv,omitempty"`
	EnvCloudName           string  `json:"env_cloud_name"`
	EnvCloudRole           string  `json:"env_cloud_role"`
	EnvCloudRoleVer        string  `json:"env_cloud_roleVer,omitempty"`
	EnvCloudRoleInstance   string  `json:"env_cloud_roleInstance"`
	EnvCloudEnvironment    string  `json:"env_cloud_environment,omitempty"`
	EnvCloudLocation       string  `json:"env_cloud_location"`
	EnvCloudDeploymentUnit string  `json:"env_cloud_deploymentUnit,omitempty"`
	EnvCloudVer            float64 `json:"env_cloud_ver"`

	// Part-B
	CallerIdentities []CallerIdentity `json:"CallerIdentities"`
	Category         string           `json:"Category"`
	OperationName    string           `json:"OperationName"`
	Result           Result           `json:"Result"`
	RequestID        string           `json:"requestId" deep:"-"`
	TargetResources  []TargetResource `json:"TargetResources"`
}

// CallerIdentity has identity information on the entity that invoke the
// operation described in the audit log.
type CallerIdentity struct {
	CallerDisplayName   string `json:"CallerDisplayName,omitempty"`
	CallerIdentityType  string `json:"CallerIdentityType"`
	CallerIdentityValue string `json:"CallerIdentityValue"`
	CallerIPAddress     string `json:"CallerIpAddress,omitempty"`
}

// Result provides information on the result of the operation.
type Result struct {
	ResultType        string `json:"ResultType"`
	ResultDescription string `json:"ResultDescription,omitempty"`
}

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
	ResultTypeUnknown     = "Unknown"
)
