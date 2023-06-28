package operator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

func GatewayEnabled(cluster *arov1alpha1.Cluster) bool {
	return len(cluster.Spec.GatewayDomains) > 0
}
