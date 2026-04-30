package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type billingDocumentConverter struct{}

func (b billingDocumentConverter) ToExternal(doc *api.BillingDocument) interface{} {
	if doc == nil || doc.Billing == nil {
		return nil
	}

	return &BillingDocument{
		ID: doc.ID,

		Key:                       doc.Key,
		ClusterResourceGroupIDKey: doc.ClusterResourceGroupIDKey,
		InfraID:                   doc.InfraID,

		Billing: &Billing{
			CreationTime:    doc.Billing.CreationTime,
			DeletionTime:    doc.Billing.DeletionTime,
			LastBillingTime: doc.Billing.LastBillingTime,
			Location:        doc.Billing.Location,
			TenantID:        doc.Billing.TenantID,
		},
	}
}

func (b billingDocumentConverter) ToExternalList(docs []*api.BillingDocument, nextLink string) interface{} {
	l := &BillingDocumentList{
		BillingDocuments: make([]*BillingDocument, 0, len(docs)),
		NextLink:         nextLink,
	}

	for _, doc := range docs {
		if converted := b.ToExternal(doc); converted != nil {
			l.BillingDocuments = append(l.BillingDocuments, converted.(*BillingDocument))
		}
	}

	return l
}
