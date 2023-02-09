package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	utilgraph "github.com/Azure/ARO-RP/pkg/util/graph"
)

// initializeClusterSPClients initialized clients, based on cluster service principal
func (m *manager) initializeClusterSPClients(ctx context.Context) error {
	spp := m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile
	options := m.env.Environment().ClientSecretCredentialOptions()
	tokenCredential, err := azidentity.NewClientSecretCredential(
		m.subscriptionDoc.Subscription.Properties.TenantID,
		spp.ClientID, string(spp.ClientSecret), options)
	if err != nil {
		return err
	}

	m.spGraphClient, err = m.env.Environment().NewGraphServiceClient(tokenCredential)

	return err
}

func (m *manager) clusterSPObjectID(ctx context.Context) error {
	var clusterSPObjectID *string
	var err error

	spp := m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// NOTE: Do not override err with the error returned by
	// wait.PollImmediateUntil. Doing this will not propagate the latest error
	// to the user in case when wait exceeds the timeout
	_ = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		clusterSPObjectID, err = utilgraph.GetServicePrincipalIDByAppID(ctx, m.spGraphClient, spp.ClientID)
		if err != nil {
			return false, err
		}
		if clusterSPObjectID == nil {
			return false, fmt.Errorf("no service principal found for application ID '%s'", spp.ClientID)
		}
		return true, nil
	}, timeoutCtx.Done())
	if err != nil {
		return err
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
