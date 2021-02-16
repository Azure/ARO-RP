package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"time"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
)

func (m *manager) clusterSPObjectID(ctx context.Context) error {
	var clusterSPObjectID string
	spp := &m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile

	token, err := aad.GetToken(ctx, m.log, m.doc.OpenShiftCluster, m.subscriptionDoc, m.env.Environment().ActiveDirectoryEndpoint, m.env.Environment().GraphEndpoint)
	if err != nil {
		return err
	}

	spGraphAuthorizer := autorest.NewBearerAuthorizer(token)

	applications := graphrbac.NewApplicationsClient(m.env.Environment(), m.subscriptionDoc.Subscription.Properties.TenantID, spGraphAuthorizer)

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// NOTE: Do not override err with the error returned by
	// wait.PollImmediateUntil. Doing this will not propagate the latest error
	// to the user in case when wait exceeds the timeout
	_ = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		var res azgraphrbac.ServicePrincipalObjectResult
		res, err = applications.GetServicePrincipalsIDByAppID(ctx, spp.ClientID)
		if err != nil {
			if strings.Contains(err.Error(), "Authorization_IdentityNotFound") {
				m.log.Info(err)
				return false, nil
			}
			return false, err
		}

		clusterSPObjectID = *res.Value
		return true, nil
	}, timeoutCtx.Done())
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
