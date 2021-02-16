package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/aad"
)

func (m *manager) clusterSPObjectID(ctx context.Context) error {
	spp := &m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile

	am := aad.NewManager(m.log, m.env.Environment(), m.subscriptionDoc.Subscription.Properties.TenantID, spp.ClientID, string(spp.ClientSecret))

	clusterSPObjectID, err := am.GetServicePrincipalID(ctx)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID = clusterSPObjectID
		return nil
	})
	return err
}

func (m *manager) fixupClusterSPObjectID(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID != "" {
		return nil
	}

	err := m.clusterSPObjectID(ctx)
	if err != nil {
		m.log.Print(err)
	}

	return nil
}
