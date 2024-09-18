package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

// UpdateClusterOperatorFlags updates the OperatorFlags object in the ARO
// Cluster document.
func UpdateClusterOperatorFlags(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	props := th.GetOpenShiftClusterProperties()

	ch, err := th.ClientHelper()
	if err != nil {
		return mimo.TerminalError(err)
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		clusterObj := &arov1alpha1.Cluster{}

		err = ch.GetOne(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, clusterObj)
		if err != nil {
			if kerrors.IsNotFound(err) {
				// cluster doc being gone is unrecoverable
				return mimo.TerminalError(err)
			}
			return mimo.TransientError(err)
		}

		clusterObj.Spec.OperatorFlags = arov1alpha1.OperatorFlags(props.OperatorFlags)

		err = ch.Update(ctx, clusterObj)
		if err != nil {
			if kerrors.IsConflict(err) {
				return err
			} else {
				return mimo.TransientError(err)
			}
		}
		return nil
	})
}
