package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/api/admin"
)

// Billing documents are created by the RP when clusters are provisioned.
// This test runs after cluster creation to ensure billing documents exist.
var _ = Describe("[Admin API] Billing documents", Serial, Ordered, func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must return a specific billing document", func(ctx context.Context) {
		var oc = &admin.OpenShiftCluster{}

		// Wait for the cluster to be in a succeeded state before continuing
		Eventually(func(g Gomega, ctx context.Context) {
			oc = adminGetCluster(g, ctx, clusterResourceID)
			g.Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		By("listing billing documents to get an ID")
		docs := adminListBillingDocuments(Default, ctx, "/admin/billingDocuments")

		By("ensuring billing documents exist")
		Expect(docs).ToNot(BeEmpty(), "expected billing documents to exist from previous test")

		billingDocID := docs[0].ID

		By("getting a specific billing document via RP admin API")
		doc := adminGetBillingDocument(Default, ctx, "/admin/billingDocuments/"+billingDocID)

		By("checking the billing document has expected fields")
		Expect(doc).ToNot(BeNil())
		Expect(doc.ID).To(Equal(billingDocID))
		Expect(doc.ClusterResourceGroupIDKey).ToNot(BeEmpty())
		Expect(doc.Billing).ToNot(BeNil())
		Expect(doc.Billing.TenantID).ToNot(BeEmpty())
		Expect(doc.Billing.Location).ToNot(BeEmpty())
	})
})
