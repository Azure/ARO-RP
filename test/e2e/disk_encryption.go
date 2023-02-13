package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// E2E cluster comes with encryption at host and encryption at rest enabled,
// and it is a part of E2E set up.
// However but it is not a standard configuration: by default encryption
// at host is disabled and encryption at rest is enabled, but with
// Azure managed encryption key.
//
// Ideally we should test permutations of cluster features. For example:
//   - Both encryption at host and encryption at rest enabled
//   - Both disabled
//   - Encryption at host enabled, but encryption at rest disabled
//   - Encryption at host disabled, but encryption at rest enabled, etc
// Same for other cluster configurations such as public/private
// API server visibility. But we are not there yet.

var _ = Describe("Encryption at host", func() {
	It("must be enabled on the test cluster and each VM must have encryption at host enabled", func(ctx context.Context) {
		By("getting the test cluster resource")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		By("checking that encryption at host is enabled for masters")
		Expect(oc.OpenShiftClusterProperties).To(Not(BeNil()))
		Expect(oc.OpenShiftClusterProperties.MasterProfile).To(Not(BeNil()))
		Expect((*oc.OpenShiftClusterProperties.MasterProfile).EncryptionAtHost).To(BeEquivalentTo("Enabled"))

		By("checking that encryption at host is enabled for workers")
		Expect(oc.OpenShiftClusterProperties).To(Not(BeNil()))
		Expect(oc.OpenShiftClusterProperties.WorkerProfiles).To(Not(BeNil()))
		Expect(*oc.OpenShiftClusterProperties.WorkerProfiles).NotTo(BeEmpty())
		for _, profile := range *oc.OpenShiftClusterProperties.WorkerProfiles {
			Expect(profile.EncryptionAtHost).To(BeEquivalentTo("Enabled"))
		}

		By("getting the resource group where the VM instances live in")
		clusterResourceGroup := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')

		By("listing all VMs for the test cluster")
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

var _ = Describe("Disk encryption at rest", func() {
	It("must be enabled with customer managed key for the cluster and each disk must have it enabled", func(ctx context.Context) {
		By("getting the test cluster resource")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		By("checking that disk encryption at rest is enabled for masters")
		Expect(oc.OpenShiftClusterProperties).To(Not(BeNil()))
		Expect(oc.OpenShiftClusterProperties.MasterProfile).To(Not(BeNil()))
		Expect(*(*oc.OpenShiftClusterProperties.MasterProfile).DiskEncryptionSetID).NotTo(BeEmpty())

		By("checking that disk encryption at rest is enabled for workers")
		Expect(oc.OpenShiftClusterProperties).To(Not(BeNil()))
		Expect(oc.OpenShiftClusterProperties.WorkerProfiles).To(Not(BeNil()))
		Expect(*oc.OpenShiftClusterProperties.WorkerProfiles).NotTo(BeEmpty())
		for _, profile := range *oc.OpenShiftClusterProperties.WorkerProfiles {
			Expect(*profile.DiskEncryptionSetID).NotTo(BeEmpty())
		}

		By("getting the resource group where the VM instances live in")
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

		By("making sure the encrypted storage class is default")
		sc, err := clients.Kubernetes.StorageV1().StorageClasses().Get(ctx, "managed-premium-encrypted-cmk", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(sc).NotTo(BeNil())
		Expect(sc.Annotations).NotTo(BeNil())
		Expect(sc.Annotations["storageclass.kubernetes.io/is-default-class"]).To(Equal("true"))

		By("making sure the encrypted storage class uses worker disk encryption set")
		expectedDiskEncryptionSetID := ((*oc.OpenShiftClusterProperties.WorkerProfiles)[0].DiskEncryptionSetID)
		Expect(sc.Parameters).NotTo(BeNil())
		Expect(sc.Parameters["diskEncryptionSetID"]).NotTo(Equal(expectedDiskEncryptionSetID))
	})
})
