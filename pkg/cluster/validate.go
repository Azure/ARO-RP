package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api/validate"
)

func (m *manager) validateResourcesFromFP(ctx context.Context) error {
	pdpChecker, err := validate.CreatePDPClient(m.env, m.log, m.doc.OpenShiftCluster, m.subscriptionDoc.Subscription)
	if err != nil {
		return err
	}
	ocDynamicValidator := validate.NewFirstPartyOpenShiftClusterDynamicValidator(
		m.log, m.env, m.doc.OpenShiftCluster, m.subscriptionDoc, m.fpAuthorizer, pdpChecker,
	)
	return ocDynamicValidator.Dynamic(ctx)
}

func (m *manager) validateResourcesFromSP(ctx context.Context) error {
	pdpChecker, err := validate.CreatePDPClient(m.env, m.log, m.doc.OpenShiftCluster, m.subscriptionDoc.Subscription)
	if err != nil {
		return err
	}
	ocDynamicValidator := validate.NewClientOpenShiftClusterDynamicValidator(
		m.log, m.env, m.doc.OpenShiftCluster, m.subscriptionDoc, pdpChecker,
	)
	return ocDynamicValidator.Dynamic(ctx)
}
