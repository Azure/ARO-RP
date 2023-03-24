package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Get cluster", func() {
	It("must be possible get a cluster and retrieve some (enriched) fields", func(ctx context.Context) {
		By("getting the cluster resource")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		By("checking we retrieved the default Ingress Profile (and only this one by default)")
		Expect(oc.IngressProfiles).NotTo(BeNil())
		Expect(*oc.IngressProfiles).To(HaveLen(1))
		ingressProfile := (*oc.IngressProfiles)[0]
		Expect(*ingressProfile.Name).To(Equal("default"))
		Expect(ingressProfile.IP).NotTo(BeNil())
		Expect(*ingressProfile.IP).NotTo(BeEmpty())

		By("checking we retrieved Cluster version")
		clusterProfile := oc.ClusterProfile
		Expect(clusterProfile).NotTo(BeNil())
		Expect(*clusterProfile.Version).NotTo(BeEmpty())

		By("checking we retrieved at least one Worker Profile")
		workerProfiles := oc.WorkerProfiles
		Expect(workerProfiles).NotTo(BeNil())
		Expect(*workerProfiles).NotTo(BeEmpty())

		By("checking we retrieved associated systemData")
		systemData := oc.SystemData
		Expect(systemData).NotTo(BeNil())
	})
})
