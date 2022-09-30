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

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api/admin"
)

var _ = Describe("[Admin API] Cluster admin update action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should be able to run cluster update operation on a cluster", func() {
		var oc = &admin.OpenShiftCluster{}
		ctx := context.Background()
		resourceID := resourceIDFromEnv()

		By("triggering the update via RP admin API")
		resp, err := adminRequest(ctx, http.MethodPatch, resourceID, nil, json.RawMessage("{}"), oc)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("checking provisioning state")
		Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateAdminUpdating))
		Expect(oc.Properties.LastProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))

		By("waiting for the update to complete")
		err = wait.PollImmediate(10*time.Second, 30*time.Minute, func() (bool, error) {
			oc = getCluster(ctx, resourceID)
			return oc.Properties.ProvisioningState == admin.ProvisioningStateSucceeded, nil
		})
		Expect(err).NotTo(HaveOccurred())
	})
})
