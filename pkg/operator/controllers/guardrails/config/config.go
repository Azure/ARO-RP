package config

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type GuardRailsDeploymentConfig struct {
	Pullspec           string
	Namespace          string
	ManagerRequestsCPU string
	ManagerLimitCPU    string
	ManagerRequestsMem string
	ManagerLimitMem    string
	AuditRequestsCPU   string
	AuditLimitCPU      string
	AuditRequestsMem   string
	AuditLimitMem      string

	ValidatingWebhookTimeout       string
	ValidatingWebhookFailurePolicy string
	MutatingWebhookTimeout         string
	MutatingWebhookFailurePolicy   string
	RoleSCCResourceName            string
}

type GuardRailsPolicyConfig struct {
	Enforcement string
}
