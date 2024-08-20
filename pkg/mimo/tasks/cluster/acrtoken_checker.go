package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func EnsureACRToken(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	env := th.Environment()
	localFpAuthorizer, err := th.LocalFpAuthorizer()
	if err != nil {
		return mimo.TerminalError(err)
	}

	token, err := acrtoken.NewManager(env, localFpAuthorizer)
	if err != nil {
		return err
	}

	rp := token.GetRegistryProfile(th.GetOpenshiftClusterDocument().OpenShiftCluster)
	var now = time.Now().UTC()
	expiry := rp.Expiry.Time

	if expiry.After(now) {
		return mimo.TerminalError(errors.New("ACR Token has expired"))
	}

	return nil
}
