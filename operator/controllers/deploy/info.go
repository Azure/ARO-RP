package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/Azure/ARO-RP/operator/controllers/consts"
)

func OperatorNamespace() (string, error) {
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		if errors.Is(err, k8sutil.ErrNoNamespace) {
			return consts.LocalNamespace, nil
		}
		return "", err
	}
	return operatorNs, nil
}
