package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/api/admin"
)

var _ = Describe("[Admin API] Cluster admin update action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must run cluster update operation on a cluster", func(ctx context.Context) {
		var oc = &admin.OpenShiftCluster{}

		By("triggering the update via RP admin API")
		resp, err := adminRequest(ctx, http.MethodPatch, clusterResourceID, nil, true, json.RawMessage("{}"), oc)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("checking provisioning state")
		Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateAdminUpdating))
		Expect(oc.Properties.LastProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))

		By("waiting for the update to complete")
		Eventually(func(g Gomega, ctx context.Context) {
			oc = adminGetCluster(g, ctx, clusterResourceID)

			g.Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))
		}).WithContext(ctx).Should(Succeed())
	})
})
