package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

// UpdateClusterOperatorFlags updates the OperatorFlags object in the ARO
// Cluster custom resource.
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

// Set an Operator flag in a cluster doc. Does not apply it to the cluster (see
// UpdateClusterOperatorFlags for that).
func SetOperatorFlagInClusterDoc(ctx context.Context, flagName string, flagValue string) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	_, err = th.PatchOpenShiftClusterDocument(ctx, func(oscd *api.OpenShiftClusterDocument) error {
		oscd.OpenShiftCluster.Properties.OperatorFlags[flagName] = flagValue
		return nil
	})
	return err
}
