package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	consoleapi "github.com/openshift/console-operator/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func (i *Installer) updateConsoleBranding(ctx context.Context, operatorClient operatorclient.Interface) error {
	i.log.Print("updating console-operator branding")
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		operatorConfig, err := operatorClient.OperatorV1().Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		operatorConfig.Spec.Customization.Brand = operatorv1.BrandAzure

		_, err = operatorClient.OperatorV1().Consoles().Update(operatorConfig)
		return err
	})
}
