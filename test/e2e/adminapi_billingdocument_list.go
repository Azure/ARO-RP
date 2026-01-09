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

	It("must return list of billing documents", func(ctx context.Context) {
		var oc = &admin.OpenShiftCluster{}

		// Wait for the cluster to be in a succeeded state before continuing
		Eventually(func(g Gomega, ctx context.Context) {
			oc = adminGetCluster(g, ctx, clusterResourceID)
			g.Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		By("listing the billing documents via RP admin API")
		docs := adminListBillingDocuments(Default, ctx, "/admin/providers/Microsoft.RedHatOpenShift/billingDocuments")

		By("checking that we received a list")
		Expect(docs).ToNot(BeNil())

		By("checking that at least one billing document exists from cluster creation")
		Expect(docs).ToNot(BeEmpty(), "expected at least one billing document to exist from cluster creation")

		By("checking that billing documents have expected fields")
		doc := docs[0]
		Expect(doc.ID).ToNot(BeEmpty())
		Expect(doc.ClusterResourceGroupIDKey).ToNot(BeEmpty())
		Expect(doc.Billing).ToNot(BeNil())
		Expect(doc.Billing.TenantID).ToNot(BeEmpty())
		Expect(doc.Billing.Location).ToNot(BeEmpty())
	})
})
