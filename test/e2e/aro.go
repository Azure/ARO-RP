//go:build e2e
// +build e2e

package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// The high level idea of these tests is to definitively say that
// a cluster is an ARO cluster. This should help us catch regressions
// when migrating between installers (e.g. vendored one vs CLI one run by Hive)
var _ = Describe("ARO Cluster", func() {
	It("must have ARO-specific machine configs", func(ctx context.Context) {
		expectedMachineConfigs := []string{
			"90-aro-worker-registries",
			"99-master-aro-dns",
			"99-master-ssh",
			"99-worker-aro-dns",
			"99-worker-ssh",
		}

		By("listing machine configs")
		mcs, err := clients.MachineConfig.MachineconfigurationV1().MachineConfigs().List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		actualMachineConfigNames := []string{}
		for _, mc := range mcs.Items {
			actualMachineConfigNames = append(actualMachineConfigNames, mc.Name)
		}

		By("verifying that ARO-specific machine configs exist")
		Expect(actualMachineConfigNames).To(ContainElements(expectedMachineConfigs))
	})

	It("must have ARO-specific custom resource", func(ctx context.Context) {
		// acrDomainList should contain acrDomain verifier
		acrDomainList := []string{"arointsvc.azurecr.io", "arointsvc.azurecr.us", "arosvc.azurecr.io", "arosvc.azurecr.us"}
		azEnvironmentList := []string{azureclient.PublicCloud.Environment.Name, azureclient.USGovernmentCloud.Environment.Name}

		By("getting an ARO operator cluster resource")
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("verifying AcrDomain exists and is a value we expect")
		Expect(co.Spec.ACRDomain).NotTo(BeNil())
		Expect(acrDomainList).To(ContainElement(co.Spec.ACRDomain))

		By("verifying AZEnvironment exists and is a value we expect")
		Expect(co.Spec.AZEnvironment).NotTo(BeNil())
		Expect(azEnvironmentList).To(ContainElement(co.Spec.AZEnvironment))

		By("verifying GenevaLogging exists")
		Expect(co.Spec.GenevaLogging).NotTo(BeNil())

		By("verifying OperatorFlags are set and equivalent to latest defaults")
		Expect(co.Spec.OperatorFlags).To(BeEquivalentTo(operator.DefaultOperatorFlags()))

		By("verifying InternetChecker exists")
		Expect(co.Spec.InternetChecker).NotTo(BeNil())
	})
})
