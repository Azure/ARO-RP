package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	wFips = "99-worker-fips"
	mFips = "99-master-fips"
)

var _ = Describe("Validate FIPS Mode", func() {
	ctx := context.Background()
	It("should be possible to retrieve FipsValidatedModules from cluster document", func() {
		oc, err := clients.OpenshiftClustersv20210901preview.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		// Check we retrieve FipsValidatedModules
		clusterProfile := oc.ClusterProfile
		Expect(clusterProfile).NotTo(BeNil())
		Expect(string(clusterProfile.FipsValidatedModules)).To(Equal("Enabled"))

	})
	It("should be possible to validate fips master and worker machineconfigs exist", func() {
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
		if !masterFips {
			err = fmt.Errorf("FIPS machine configs not found on master")
		}
		Expect(err).NotTo(HaveOccurred())
		if !workerFips {
			err = fmt.Errorf("FIPS machine configs not found on worker")
		}
		Expect(err).NotTo(HaveOccurred())
	})
})
