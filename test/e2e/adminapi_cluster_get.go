package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/api/admin"
)

var _ = Describe("[Admin API] Get cluster action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should be able to return single cluster with admin fields", func() {
		ctx := context.Background()
		resourceID := resourceIDFromEnv()

		By("requesting the cluster document via RP admin API")
		var oc admin.OpenShiftCluster
		resp, err := adminRequest(ctx, http.MethodGet, resourceID, nil, nil, &oc)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("checking that we received the expected cluster")
		Expect(oc.ID).To(Equal(resourceID))

		By("checking that fields available only in Admin API have values")
		// Note: some fields will have empty values
		// on successfully provisioned cluster (oc.Properties.Install, for example)
		Expect(oc.Properties.ServicePrincipalProfile.TenantID).ToNot(BeEmpty())
		Expect(oc.Properties.StorageSuffix).ToNot(BeEmpty())
	})
})
