package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

func (m *manager) validateResources(ctx context.Context) error {
	spp := m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile
	tokenClient := aad.NewTokenClient()
	token, err := tokenClient.GetToken(ctx, m.log, spp.ClientID, string(spp.ClientSecret), m.subscriptionDoc.Subscription.Properties.TenantID, m.env.Environment().ActiveDirectoryEndpoint, m.env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	spAuthorizer := refreshable.NewAuthorizer(token)

	ocDynamicValidator, err := validate.NewOpenShiftClusterDynamicValidator(
		m.log, m.env, m.doc.OpenShiftCluster, m.subscriptionDoc, m.fpAuthorizer, spAuthorizer, token,
	)
	if err != nil {
		return err
	}
	return ocDynamicValidator.Dynamic(ctx)
}
