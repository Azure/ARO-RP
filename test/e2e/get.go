package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Get cluster", func() {
	It("should be possible get a cluster and retrieve some (enriched) fields", func() {
		ctx := context.Background()
		oc, err := clients.OpenshiftClustersv20200430.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		// Check we retrieve default Ingress Profile (and only this one by default)
		Expect(oc.IngressProfiles).NotTo(BeNil())
		Expect(*oc.IngressProfiles).To(HaveLen(1))
		ingressProfile := (*oc.IngressProfiles)[0]
		Expect(*ingressProfile.Name).To(Equal("default"))
		Expect(ingressProfile.IP).NotTo(BeNil())

		// Check we retrieve Cluster version
		clusterProfile := oc.ClusterProfile
		Expect(clusterProfile).NotTo(BeNil())
		Expect(*clusterProfile.Version).NotTo(BeEmpty())

		// Check we managed to retrieve at least one Worker Profile
		workerProfiles := oc.WorkerProfiles
		Expect(workerProfiles).NotTo(BeNil())
		Expect(*workerProfiles).NotTo(BeEmpty())
	})
})
