package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Admin API] List billing documents action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must return list of billing documents", func(ctx context.Context) {
		By("listing the billing documents via RP admin API")
		docs := adminListBillingDocuments(Default, ctx, "/admin/providers/Microsoft.RedHatOpenShift/billingDocuments")

		By("checking that we received a list")
		Expect(docs).ToNot(BeNil())

		By("checking that billing documents have expected fields if any exist")
		if len(docs) > 0 {
			doc := docs[0]
			Expect(doc.ID).ToNot(BeEmpty())
			Expect(doc.Billing).ToNot(BeNil())
			Expect(doc.Billing.TenantID).ToNot(BeEmpty())
			Expect(doc.Billing.Location).ToNot(BeEmpty())
		}
	})
})
