package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	operatorv1 "github.com/openshift/api/operator/v1"
)

const consoleConfigResourceName string = "cluster"

func (m *manager) updateConsoleBranding(ctx context.Context) error {
	m.log.Print("updating console-operator branding")
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		operatorConfig, err := m.operatorcli.OperatorV1().Consoles().Get(ctx, consoleConfigResourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		operatorConfig.Spec.Customization.Brand = operatorv1.BrandAzure

		_, err = m.operatorcli.OperatorV1().Consoles().Update(ctx, operatorConfig, metav1.UpdateOptions{})
		return err
	})
}
