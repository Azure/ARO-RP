package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	subnetscontroller "github.com/Azure/ARO-RP/pkg/operator/controllers/subnets"
	"k8s.io/apimachinery/pkg/types"
)

func (mon *Monitor) emitNSGReconciliation(ctx context.Context) error {
	cluster := &arov1alpha1.Cluster{}
	err := mon.ch.GetOne(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster)
	if err != nil {
		return err
	}

	updated, err := annotationUpdated(cluster.Annotations)
	if err != nil {
		return err
	}

	if updated {
		mon.emitGauge("nsg.reconciliations", int64(1), nil)
	}

	return nil
}

func annotationUpdated(annotations map[string]string) (bool, error) {
	if annotations == nil {
		return false, nil
	}

	t := annotations[subnetscontroller.AnnotationTimestamp]
	timestamp, err := time.Parse(time.RFC1123, t)
	if err != nil {
		return false, err
	}

	if time.Since(timestamp) < time.Second*70 {
		return true, nil
	}

	return false, nil
}
