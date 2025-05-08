package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/Azure/ARO-RP/pkg/api"
	utilgraph "github.com/Azure/ARO-RP/pkg/util/graph"
)

// initializeClusterSPClients initialized clients, based on cluster service principal
func (m *manager) initializeClusterSPClients(ctx context.Context) error {
	spp := m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile
	options := m.env.Environment().ClientSecretCredentialOptions()
	spTokenCredential, err := azidentity.NewClientSecretCredential(
		m.subscriptionDoc.Subscription.Properties.TenantID,
		spp.ClientID, string(spp.ClientSecret), options)
	if err != nil {
		return err
	}

	m.spGraphClient, err = m.env.Environment().NewGraphServiceClient(spTokenCredential)

	return err
}

func (m *manager) clusterSPObjectID(ctx context.Context) error {
	var (
		err               error
		clusterSPObjectID *string
	)

	spp := m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile
	clusterSPObjectID, err = utilgraph.GetServicePrincipalIDByAppID(ctx, m.spGraphClient, spp.ClientID)
	if err != nil {
		return err
	}

	if clusterSPObjectID == nil {
		return fmt.Errorf("no service principal found for application ID '%s'", spp.ClientID)
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID = *clusterSPObjectID
		return nil
	})

	return err
}

func (m *manager) fixupClusterSPObjectID(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID != "" {
		return nil
	}

	err := m.initializeClusterSPClients(ctx)
	if err != nil {
		m.log.Print(err)
		return nil
	}

	err = m.clusterSPObjectID(ctx)
	if err != nil {
		m.log.Print(err)
	}

	return nil
}
