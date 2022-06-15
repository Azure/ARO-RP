package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
)

const (
	wFips = "99-worker-fips"
	mFips = "99-master-fips"
)

var _ = Describe("Validate FIPS Mode", func() {
	ctx := context.Background()
	It("should be possible to validate fips mode is set correctly", func() {
		oc, err := clients.OpenshiftClustersv20220401.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		mcp, err := clients.MachineConfig.MachineconfigurationV1().MachineConfigPools().List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		masterFips, workerFips := false, false
		for _, m := range mcp.Items {
			for _, mc := range m.Spec.Configuration.Source {
				if mc.Name == wFips {
					workerFips = true
				}
				if mc.Name == mFips {
					masterFips = true
				}
			}
		}
		By("checking if FipsValidatedModules is enabled or disabled")
		if string(oc.ClusterProfile.FipsValidatedModules) == string(api.FipsValidatedModulesEnabled) {
			By("checking FIPs machine configs exist on master and worker")
			Expect(masterFips).To(BeTrue())
			Expect(workerFips).To(BeTrue())
		} else {
			By("checking FIPs machine configs do not exist on master and worker")
			Expect(masterFips).To(BeFalse())
			Expect(workerFips).To(BeFalse())
		}
	})
})
