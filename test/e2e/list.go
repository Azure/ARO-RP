package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("List clusters", func() {
	It("must contain the test cluster in the returned list", func(ctx context.Context) {
		By("listing clusters")
		ocList, err := clients.OpenshiftClusters.List(ctx)
		Expect(err).NotTo(HaveOccurred())

		By("checking if the test cluster is in the list")
		found := false
		for _, oc := range ocList {
			if *oc.Name == clusterName {
				found = true
				break
			}
		}
		Expect(found).To(Equal(true))
	})

	// listByResourceGroup test marked Pending (X), don't reenable until ARM caching issue is fixed, see https://github.com/Azure/ARO-RP/pull/1995
	XIt("must contain the test cluster when listing clusters by a resource group", func(ctx context.Context) {
		By(fmt.Sprintf("listing clusters by a resource group %q", vnetResourceGroup))
		ocList, err := clients.OpenshiftClusters.ListByResourceGroup(ctx, vnetResourceGroup)
		Expect(err).NotTo(HaveOccurred())

		By("checking if the test cluster is in the list")
		found := false
		for _, oc := range ocList {
			if *oc.Name == clusterName {
				found = true
				break
			}
		}
		Expect(found).To(Equal(true))
	})
})
