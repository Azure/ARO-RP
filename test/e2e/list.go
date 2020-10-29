package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("List clusters", func() {
	Specify("the test cluster should be in the returned list", func() {
		ctx := context.Background()

		ocList, err := clients.OpenshiftClusters.List(ctx)
		Expect(err).NotTo(HaveOccurred())
		//Expect(len(ocList.Value)).To(Greater(1)))

		found := false
		for _, oc := range ocList {
			if *oc.Name == clusterName {
				found = true
				break
			}
		}

		Expect(found).To(Equal(true))
	})
	Specify("the test cluster should be in the returned listByResourceGroup", func() {
		ctx := context.Background()

		ocList, err := clients.OpenshiftClusters.ListByResourceGroup(ctx, im.ResourceGroup())
		Expect(err).NotTo(HaveOccurred())
		//Expect(len(ocList.Value)).To(Greater(1)))

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
