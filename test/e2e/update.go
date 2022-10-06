package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"math/rand"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Update clusters", func() {
	It("should be possible to set the tag on a cluster via PUT", func() {
		// az resource tag --tags key=value --ids /subscriptions/xxx/resourceGroups/xxx/providers/Microsoft.RedHatOpenShift/openShiftClusters/xxx

		ctx := context.Background()
		value := strconv.Itoa(rand.Int())

		oc, err := clients.OpenshiftClustersv20200430.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		Expect(oc.Tags).NotTo(HaveKeyWithValue("key", &value))

		if oc.Tags == nil {
			oc.Tags = map[string]*string{}
		}
		oc.Tags["key"] = &value

		err = clients.OpenshiftClustersv20200430.CreateOrUpdateAndWait(ctx, vnetResourceGroup, clusterName, oc)
		Expect(err).NotTo(HaveOccurred())

		oc, err = clients.OpenshiftClustersv20200430.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		Expect(oc.Tags).To(HaveKeyWithValue("key", &value))
	})
})
