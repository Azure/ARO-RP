package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var _ = Describe("Encryption at host should be enabled", func() {
	Specify("each VM should have encryption at host enabled", func() {
		ctx := context.Background()

		By("getting the resource group where the VM instances live in")
		oc, err := clients.OpenshiftClustersv20200430.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		clusterResourceGroup := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')

		By("listing all VMs")
		vms, err := clients.VirtualMachines.List(ctx, clusterResourceGroup)
		Expect(err).NotTo(HaveOccurred())
		Expect(vms).NotTo(HaveLen(0))

		By("checking the encryption property on each VM")
		for _, vm := range vms {
			Expect(vm.SecurityProfile).To(Not(BeNil()))
			Expect(vm.SecurityProfile.EncryptionAtHost).To(Not(BeNil()))
			Expect(*vm.SecurityProfile.EncryptionAtHost).To(Equal(true))
		}

	})
})

var _ = Describe("Disk encryption at rest should be enabled with customer managed key", func() {
	Specify("each disk should have encryption at rest with customer managed key enabled", func() {
		ctx := context.Background()

		By("getting the resource group where the VM instances live in")
		oc, err := clients.OpenshiftClustersv20200430.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		clusterResourceGroup := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')

		By("listing all VMs")
		vms, err := clients.VirtualMachines.List(ctx, clusterResourceGroup)
		Expect(err).NotTo(HaveOccurred())
		Expect(vms).NotTo(HaveLen(0))

		// We have to get the disks by VM, because when getting all disks by resource group,
		// we do not get recently created disks, see https://github.com/Azure/azure-cli/issues/17123
		By("checking the encryption property on each OS disk of each VM")
		for _, vm := range vms {
			osDisk, err := clients.Disks.Get(ctx, clusterResourceGroup, *vm.StorageProfile.OsDisk.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(osDisk.Encryption.Type).To(Equal(mgmtcompute.EncryptionAtRestWithCustomerKey))
		}

		By("checking if the encrypted storage class is default")
		sc, err := clients.Kubernetes.StorageV1().StorageClasses().Get(ctx, "managed-premium-encrypted-cmk", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(sc).NotTo(BeNil())

		Expect(sc.Annotations).NotTo(BeNil())
		Expect(sc.Annotations["storageclass.kubernetes.io/is-default-class"]).To(Equal("true"))

		Expect(sc.Parameters).NotTo(BeNil())
		Expect(sc.Parameters["diskEncryptionSetID"]).NotTo(BeEmpty())
	})
})
