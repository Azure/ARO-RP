package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Admin API] Get billing document action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must return a billing document if it exists", func(ctx context.Context) {
		By("listing billing documents to get an ID")
		// Note: Billing documents are created by the RP when clusters are provisioned.
		// This test assumes billing documents exist from actual cluster operations in the database.
		docs := adminListBillingDocuments(Default, ctx, "/admin/providers/Microsoft.RedHatOpenShift/billingDocuments")

		if len(docs) == 0 {
			Skip("No billing documents exist in the database")
		}

		billingDocID := docs[0].ID

		By("getting a specific billing document via RP admin API")
		doc := adminGetBillingDocument(Default, ctx, "/admin/providers/Microsoft.RedHatOpenShift/billingDocuments/"+billingDocID)

		By("checking the billing document has expected fields")
		Expect(doc).ToNot(BeNil())
		Expect(doc.ID).To(Equal(billingDocID))
		Expect(doc.Billing).ToNot(BeNil())
		Expect(doc.Billing.TenantID).ToNot(BeEmpty())
		Expect(doc.Billing.Location).ToNot(BeEmpty())
	})
})
