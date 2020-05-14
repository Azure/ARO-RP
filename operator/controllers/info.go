package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
)

func OperatorNamespace() (string, error) {
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		if errors.Is(err, k8sutil.ErrNoNamespace) {
			return LocalNamespace, nil
		}
		return "", err
	}
	return operatorNs, nil
}
