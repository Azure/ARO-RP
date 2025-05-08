package example

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func ReportClusterVersion(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return err
	}

	ch, err := th.ClientHelper()
	if err != nil {
		return err
	}

	cv := &configv1.ClusterVersion{}

	err = ch.GetOne(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return fmt.Errorf("unable to get ClusterVersion: %w", err)
	}

	th.SetResultMessage(fmt.Sprintf("cluster version is: %s", cv.Status.History[0].Version))

	return nil
}
