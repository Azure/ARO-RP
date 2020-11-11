package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
)

func (m *manager) Update(ctx context.Context) error {
	var err error
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		err = m.ocDynamicValidator.Dynamic(ctx)
		if azureerrors.HasAuthorizationFailedError(err) ||
			azureerrors.HasLinkedAuthorizationFailedError(err) {
			m.log.Print(err)
			return false, nil
		}
		return err == nil, err
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	return nil
}
