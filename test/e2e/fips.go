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

// FIPS Mode is set differently and the test needs updating after 2022-04-01 API is active
var _ = XSpecify("Validate FIPS Mode", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	ctx := context.Background()
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
