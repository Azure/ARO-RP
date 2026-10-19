package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/Azure/ARO-RP/pkg/api/util/immutable"

var openShiftClusterUpdatePolicy = immutable.Policy{
	Mutable: []string{
		"tags",
		"properties.servicePrincipalProfile.clientId",
		"properties.servicePrincipalProfile.clientSecret",
		"properties.platformWorkloadIdentityProfile.upgradeableTo",
		"properties.platformWorkloadIdentityProfile.platformWorkloadIdentities",
		"properties.networkProfile.loadBalancerProfile.managedOutboundIps",
		"identity.principalId",
		"identity.tenantId",
		"identity.userAssignedIdentities",
	},
	ReadOnly: []string{
		"systemData",
		"properties.workerProfilesStatus",
		"properties.clusterProfile.oidcIssuer",
		"properties.consoleProfile.url",
		"properties.networkProfile.loadBalancerProfile.effectiveOutboundIps",
		"properties.apiserverProfile.url",
		"properties.apiserverProfile.ip",
		"properties.ingressProfiles*.ip",
	},
	ReadOnlyValue: []string{
		"properties.consoleProfile.url",
		"properties.apiserverProfile.url",
		"properties.apiserverProfile.ip",
		"properties.ingressProfiles*.ip",
	},
	CaseInsensitive: []string{
		"id",
		"name",
		"type",
	},
	NormalizeNil: []string{
		"properties",
		"properties.clusterProfile",
		"properties.consoleProfile",
		"properties.networkProfile",
		"properties.masterProfile",
		"properties.apiserverProfile",
	},
}
