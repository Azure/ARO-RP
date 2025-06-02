package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/api/admin"
)

var _ = Describe("[Admin API] Cluster admin update action", Serial, func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must run cluster update operation on a cluster", func(ctx context.Context) {
		var oc = &admin.OpenShiftCluster{}
		var resp *http.Response
		var err error

		// Wait for the cluster to be in a succeeded state before continuing
		Eventually(func(g Gomega, ctx context.Context) {
			oc = adminGetCluster(g, ctx, clusterResourceID)
			g.Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		// Trigger the update via RP admin API, retrying on 409 Conflict
		By("triggering the update via RP admin API")
		for i := 0; i < 3; i++ {
			// Always get a fresh copy of the cluster so we have the latest ETag
			g := NewWithT(GinkgoT())
			oc = adminGetCluster(g, ctx, clusterResourceID)

			resp, err = adminRequest(ctx, http.MethodPatch, clusterResourceID, nil, true, json.RawMessage("{}"), oc)
			Expect(err).NotTo(HaveOccurred())

			if resp.StatusCode == http.StatusConflict {
				// Another update is already in flight; wait then retry
				time.Sleep(5 * time.Second)
				continue
			}

			// Expect 200 OK for a successful update
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			break
		}

		// If we have still received 409 on all 3 tries, explicitly fail
		if resp != nil && resp.StatusCode == http.StatusConflict {
			Fail("adminRequest returned 409 Conflict on all 3 retries")
		}

		By("checking provisioning state")
		g := NewWithT(GinkgoT())
		oc = adminGetCluster(g, ctx, clusterResourceID)
		Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateAdminUpdating))
		Expect(oc.Properties.LastProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))

		By("waiting for the update to complete")
		Eventually(func(g Gomega, ctx context.Context) {
			oc = adminGetCluster(g, ctx, clusterResourceID)
			g.Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		// Ensure there was no admin update error
		Expect(oc.Properties.LastAdminUpdateError).To(Equal(""))
	})
})
