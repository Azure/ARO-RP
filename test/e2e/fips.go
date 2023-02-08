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

var _ = Describe("FIPS Mode", func() {
	It("must be set correctly", func(ctx context.Context) {
		expectedFIPSMachineConfigs := []string{"99-worker-fips", "99-master-fips"}

		By("getting the test cluster resource")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		By("listing machine configs")
		mcs, err := clients.MachineConfig.MachineconfigurationV1().MachineConfigs().List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		actualMachineConfigNames := []string{}
		for _, mc := range mcs.Items {
			actualMachineConfigNames = append(actualMachineConfigNames, mc.Name)
		}

		By("checking if FipsValidatedModules is enabled or disabled")
		if string(oc.ClusterProfile.FipsValidatedModules) == string(api.FipsValidatedModulesEnabled) {
			By("checking FIPS machine configs exist on master and worker")
			Expect(actualMachineConfigNames).To(ContainElements(expectedFIPSMachineConfigs))
		} else {
			By("checking FIPS machine configs do not exist on master and worker")
			Expect(actualMachineConfigNames).NotTo(ContainElements(expectedFIPSMachineConfigs))
		}
	})
})
