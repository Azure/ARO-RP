package iswimi

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/Azure/ARO-RP/pkg/api"

// IsWimi checks whether a cluster is Workload Identity or classic
func IsWimi(cluster api.OpenShiftClusterProperties) bool {
	if cluster.PlatformWorkloadIdentityProfile == nil || cluster.ServicePrincipalProfile != nil {
		return false
	}
	return true
}
