package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/api/admin"
)

var _ = Describe("[Admin API] List clusters action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must return list of clusters with admin fields", func() {
		ctx := context.Background()
		resourceID := resourceIDFromEnv()

		testAdminClustersList(ctx, "/admin/providers/Microsoft.RedHatOpenShift/openShiftClusters", resourceID)
	})

	It("must return list of clusters with admin fields by subscription", func() {
		ctx := context.Background()
		resourceID := resourceIDFromEnv()

		path := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters", _env.SubscriptionID())
		testAdminClustersList(ctx, path, resourceID)
	})

	It("must return list of clusters with admin fields by resource group", func() {
		ctx := context.Background()
		resourceID := resourceIDFromEnv()

		path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters", _env.SubscriptionID(), vnetResourceGroup)
		testAdminClustersList(ctx, path, resourceID)
	})
})

func testAdminClustersList(ctx context.Context, path, wantResourceID string) {
	By("listing the cluster documents via RP admin API")
	ocs := adminListClusters(ctx, path)

	By("verifying that we received the expected cluster")
	var oc *admin.OpenShiftCluster
	for i := range ocs {
		if ocs[i].ID == wantResourceID {
			oc = ocs[i]
		}
	}
	Expect(oc).ToNot(BeNil())
	Expect(oc.ID).To(Equal(wantResourceID))

	By("checking that fields available only in Admin API have values")
	// Note: some fields will have empty values
	// on successfully provisioned cluster (oc.Properties.Install, for example)
	Expect(oc.Properties.StorageSuffix).ToNot(BeEmpty())
	Expect(oc.Properties.InfraID).ToNot(BeEmpty())
}
