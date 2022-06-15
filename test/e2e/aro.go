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

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

const (
	wSsh = "99-worker-ssh"
	mSsh = "99-master-ssh"
)

// The high level idea of these tests is to definitively say that a cluster is an ARO cluster.
var _ = Describe("Verify Attributes of ARO Cluster", func() {
	ctx := context.Background()
	// acrDomainList should contain acrDomain verifier
	acrDomainList := []string{"arointsvc.azurecr.io", "arointsvc.azurecr.us", "arosvc.azurecr.io", "arosvc.azurecr.us"}
	azEnvironmentList := []string{azureclient.PublicCloud.Environment.Name, azureclient.USGovernmentCloud.Environment.Name}
	It("should be possible to definitively confirm cluster is ARO", func() {
		// Get cluster object
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		mcp, err := clients.MachineConfig.MachineconfigurationV1().MachineConfigPools().List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Expect 99-worker-ssh and 99-master-ssh to exist
		masterSsh, workerSsh := false, false
		for _, m := range mcp.Items {
			for _, mc := range m.Spec.Configuration.Source {
				if mc.Name == wSsh {
					workerSsh = true
				}
				if mc.Name == mSsh {
					masterSsh = true
				}
			}
		}
		By("Checking if workerSsh and masterSsh is enabled or disabled")
		Expect(workerSsh).To(BeTrue())
		Expect(masterSsh).To(BeTrue())

		// Expect ACR Domain
		By("Verifying AcrDomain exists and is a value we expect")
		Expect(co.Spec.ACRDomain).NotTo(BeNil())
		Expect(acrDomainList).To(ContainElement(co.Spec.ACRDomain))

		// Verify AzEnvironment exists and is equal to cloud
		By("Verifying AZEnvironment exists and is a value we expect")
		Expect(co.Spec.AZEnvironment).NotTo(BeNil())
		Expect(azEnvironmentList).To(ContainElement(co.Spec.AZEnvironment))

		// Expect Geneva Logging to be present
		By("Verifying GenevaLogging Exists")
		Expect(co.Spec.GenevaLogging).NotTo(BeNil())

		// Expect Clusters operator flags to be equivalent to default operator flags
		By("Verifying OperatorFlags are set and equivalent to Latest defaults")
		Expect(co.Spec.OperatorFlags).To(BeEquivalentTo(api.DefaultOperatorFlags()))

		// Expect ARO InternetChecker to Exist
		By("Verifying InternetChecker exists")
		Expect(co.Spec.InternetChecker).NotTo(BeNil())

	})
})
