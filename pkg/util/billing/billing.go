package billing

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

type Manager interface {
	Ensure(context.Context, *api.OpenShiftClusterDocument, *api.SubscriptionDocument) error
	Delete(context.Context, *api.OpenShiftClusterDocument) error
}

type manager struct {
	billingDB database.Billing
	log       *logrus.Entry
}

func NewManager(env env.Interface, billing database.Billing, sub database.Subscriptions, log *logrus.Entry) (Manager, error) {
	return &manager{
		billingDB: billing,
		log:       log,
	}, nil
}

func (m *manager) logForDoc(doc *api.OpenShiftClusterDocument) *logrus.Entry {
	infraID := ""
	if doc.OpenShiftCluster != nil {
		infraID = doc.OpenShiftCluster.Properties.InfraID
	}

	return m.log.WithFields(logrus.Fields{
		"resource_id":               utillog.Sanitize(doc.Key),
		"billing_id":                doc.ID,
		"cluster_resource_group_id": utillog.Sanitize(doc.ClusterResourceGroupIDKey),
		"infra_id":                  infraID,
	})
}

func (m *manager) Ensure(ctx context.Context, doc *api.OpenShiftClusterDocument, sub *api.SubscriptionDocument) error {
	_, err := m.billingDB.Create(ctx, &api.BillingDocument{
		ID:                        doc.ID,
		Key:                       doc.Key,
		ClusterResourceGroupIDKey: doc.ClusterResourceGroupIDKey,
		InfraID:                   doc.OpenShiftCluster.Properties.InfraID,
		Billing: &api.Billing{
			TenantID: sub.Subscription.Properties.TenantID,
			Location: doc.OpenShiftCluster.Location,
		},
	})
	if err, ok := err.(*cosmosdb.Error); ok &&
		err.StatusCode == http.StatusConflict {
		m.logForDoc(doc).Print("billing record already present in DB")
		return nil
	}
	if err != nil {
		return err
	}

	m.logForDoc(doc).Print("billing record created in DB")
	return nil
}

func (m *manager) Delete(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	m.logForDoc(doc).Print("updating billing record with deletion time")
	_, err := m.billingDB.MarkForDeletion(ctx, doc.ID)
	if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		m.logForDoc(doc).Print("billing record not found, nothing to mark for deletion")
		return nil
	}
	if err != nil {
		m.logForDoc(doc).WithError(err).Error("failed to mark billing record for deletion")
		return err
	}

	m.logForDoc(doc).Print("billing record marked for deletion")
	return nil
}
