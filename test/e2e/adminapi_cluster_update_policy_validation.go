package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

var _ = Describe("[Admin API] Cluster admin update with policy validation", Serial, func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must fail admin update when storage account has policy violation tag, then succeed after tag removal", func(ctx context.Context) {
		oc := &admin.OpenShiftCluster{}

		By("waiting for cluster to be in succeeded state")
		Eventually(func(g Gomega, ctx context.Context) {
			oc = adminGetCluster(g, ctx, clusterResourceID)
			g.Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		By("tagging resource group to trigger policy violation")
		err := tagResource(ctx, oc.Properties.ClusterProfile.ResourceGroupID, armresources.TagsPatchOperationMerge, map[string]*string{
			"v4-e2e-V-test": pointerutils.ToPtr("trigger-policy"),
		})
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func(ctx context.Context) {
			if err := tagResource(ctx, oc.Properties.ClusterProfile.ResourceGroupID, armresources.TagsPatchOperationDelete, map[string]*string{"v4-e2e-V-test": pointerutils.ToPtr("trigger-policy")}); err != nil {
				log.Warnf("DeferCleanup: failed to remove trigger tag: %v", err)
			}
		})

		By("triggering admin update with policy violation")
		resp, err := adminRequest(ctx, http.MethodPatch, clusterResourceID, nil, true, json.RawMessage("{}"), oc)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("waiting for update to fail due to policy violation detection")
		Eventually(func(g Gomega, ctx context.Context) {
			oc = adminGetCluster(g, ctx, clusterResourceID)
			g.Expect(oc.Properties.FailedProvisioningState).To(Equal(admin.ProvisioningStateAdminUpdating))
			g.Expect(oc.Properties.LastAdminUpdateError).To(ContainSubstring("Unexpected property mutations detected"))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		By("removing policy violation tag from resource group")
		err = tagResource(ctx, oc.Properties.ClusterProfile.ResourceGroupID, armresources.TagsPatchOperationDelete, map[string]*string{"v4-e2e-V-test": pointerutils.ToPtr("trigger-policy")})
		Expect(err).NotTo(HaveOccurred())

		By("retrying admin update - should succeed now")
		Eventually(func(g Gomega, ctx context.Context) {
			oc = adminGetCluster(g, ctx, clusterResourceID)
			if oc.Properties.ProvisioningState != admin.ProvisioningStateAdminUpdating && oc.Properties.LastAdminUpdateError != "" {
				resp, err := adminRequest(ctx, http.MethodPatch, clusterResourceID, nil, true, json.RawMessage("{}"), oc)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			}
			oc = adminGetCluster(g, ctx, clusterResourceID)
			g.Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))
			g.Expect(oc.Properties.LastAdminUpdateError).To(BeEmpty())
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	})
})

func tagResource(ctx context.Context, resourceName string, operation armresources.TagsPatchOperation, tags map[string]*string) error {
	log.Infof("patch tags on %s (%s): %v", resourceName, operation, tags)

	_, err := clients.Tags.UpdateAtScope(ctx, resourceName, armresources.TagsPatchResource{
		Operation: pointerutils.ToPtr(operation),
		Properties: &armresources.Tags{
			Tags: tags,
		},
	}, nil)
	return err
}
