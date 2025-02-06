package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/util/clusteroperators"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func EnsureAPIServerIsUp(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return err
	}

	ch, err := th.ClientHelper()
	if err != nil {
		return mimo.TerminalError(err)
	}

	co := &configv1.ClusterOperator{}

	err = ch.GetOne(ctx, types.NamespacedName{Name: "kube-apiserver"}, co)
	if err != nil {
		// 404 on kube-apiserver is likely terminal
		if kerrors.IsNotFound(err) {
			return mimo.TerminalError(err)
		}

		return mimo.TransientError(err)
	}

	available := clusteroperators.IsOperatorAvailable(co)
	if !available {
		return mimo.TransientError(errors.New(clusteroperators.OperatorStatusText(co)))
	}
	return nil
}
