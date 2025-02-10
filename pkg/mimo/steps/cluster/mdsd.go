package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func EnsureMDSDCertificates(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	ch, err := th.ClientHelper()
	if err != nil {
		return mimo.TerminalError(err)
	}

	err = cluster.RenewMDSDCertificate(ctx, th.Log(), th.Environment(), ch)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// if the operator secret is not found then something has gone
			// seriously wrong, give up
			return mimo.TerminalError(err)
		} else {
			return mimo.TransientError(err)
		}
	}

	return nil
}
