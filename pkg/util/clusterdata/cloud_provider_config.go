package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// azureAuthConfig holds auth related part of cloud config
type azureAuthConfig struct {
	// The AAD Tenant ID for the Subscription that the cluster is deployed in
	TenantID string `json:"tenantId,omitempty" yaml:"tenantId,omitempty"`
	// The ClientID for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientID string `json:"aadClientId,omitempty" yaml:"aadClientId,omitempty"`
}

// cloudProviderConfig is a simplified version of https://github.com/openshift/kubernetes-legacy-cloud-providers/blob/9b98dc6790542766fc413261aedcce250fff10d3/azure/azure.go#L83-L176
type cloudProviderConfig struct {
	azureAuthConfig
}
