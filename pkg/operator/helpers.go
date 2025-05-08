package operator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

var clusterVersionForPodSecurityStandard = version.NewVersion(4, 11)

func GatewayEnabled(cluster *arov1alpha1.Cluster) bool {
	return len(cluster.Spec.GatewayDomains) > 0
}

// ShouldUsePodSecurityStandard is an admissions controller
// for pods which replaces pod security policies, enabled on
// OpenShift 4.11 and up
func ShouldUsePodSecurityStandard(ctx context.Context, client client.Reader) (bool, error) {
	cv := &configv1.ClusterVersion{}

	err := client.Get(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return false, err
	}

	vers, err := version.GetClusterVersion(cv)
	if err != nil {
		return false, err
	}

	if vers.Lt(clusterVersionForPodSecurityStandard) {
		return false, nil
	}

	return true, nil
}
