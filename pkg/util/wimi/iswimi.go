package wimi

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/Azure/ARO-RP/pkg/api"

// IsWimi checks whether a cluster is a Workload Identity cluster or Service Principal cluster
func IsWimi(oc api.OpenShiftCluster) bool {
	if oc.Properties.PlatformWorkloadIdentityProfile == nil || oc.Properties.ServicePrincipalProfile != nil {
		return false
	}
	return true
}
