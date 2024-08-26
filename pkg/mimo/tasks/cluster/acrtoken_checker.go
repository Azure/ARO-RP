package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"time"

	"k8s.io/apimachinery/pkg/types"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func EnsureACRTokenIsValid(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	ch, err := th.ClientHelper()
	if err != nil {
		return err
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

	cluster := &arov1alpha1.Cluster{}
	err = ch.GetOne(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster)
	if err != nil {
		return mimo.NewMIMOError(err, mimo.MIMOErrorTypeTerminalError)
	}

	rp := token.GetRegistryProfile(th.GetOpenshiftClusterDocument().OpenShiftCluster)
	var now = time.Now().UTC()
	expiry := rp.Expiry.Time

	if expiry.After(now) {
		return mimo.TerminalError(errors.New("ACR token has expired"))
	}

	th.SetResultMessage("ACR token is valid")
	return nil
}
