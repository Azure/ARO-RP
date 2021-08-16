package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var _ = Describe("Encryption at host should be enabled", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

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
			log.Println(vm.Name)
			Expect(vm.SecurityProfile).To(Not(BeNil()))
			Expect(vm.SecurityProfile.EncryptionAtHost).To(Not(BeNil()))
			Expect(*vm.SecurityProfile.EncryptionAtHost).To(Equal(true))
		}

	})
})

var _ = Describe("Disk encryption at rest should be enabled with customer managed key", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	// We have to get the disks by VM, because when getting all disks by resource group, we do not get recently created disks, see https://github.com/Azure/azure-cli/issues/17123
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

		By("attaching arbitrary data disks to the VMs")
		diskEncryptionSetId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/diskEncryptionSets/%s", _env.SubscriptionID(), vnetResourceGroup, clusterName+"-des")
		for _, vm := range vms {
			vm.StorageProfile.DataDisks = &[]mgmtcompute.DataDisk{{
				Lun:          to.Int32Ptr(0),
				CreateOption: mgmtcompute.DiskCreateOptionTypesEmpty,
				DiskSizeGB:   to.Int32Ptr(1),
				ManagedDisk: &mgmtcompute.ManagedDiskParameters{
					DiskEncryptionSet: &mgmtcompute.DiskEncryptionSetParameters{ID: to.StringPtr(diskEncryptionSetId)},
				},
			}}
			err = clients.VirtualMachines.CreateOrUpdateAndWait(ctx, clusterResourceGroup, *vm.Name, vm)
			Expect(err).NotTo(HaveOccurred())
		}

		// get the VMs again because now the disks have been created and they received generated names
		vms, err = clients.VirtualMachines.List(ctx, clusterResourceGroup)
		Expect(err).NotTo(HaveOccurred())
		Expect(vms).NotTo(HaveLen(0))

		By("checking the encryption property on each OS disk of each VM")
		for _, vm := range vms {
			osDisk, err := clients.Disks.Get(ctx, clusterResourceGroup, *vm.StorageProfile.OsDisk.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(osDisk.Encryption.Type).To(Equal(mgmtcompute.EncryptionAtRestWithCustomerKey))
		}

		By("checking the encryption property on each data disk of each VM")
		for _, vm := range vms {
			Expect(vm.StorageProfile.DataDisks).NotTo(BeNil())
			Expect(*vm.StorageProfile.DataDisks).NotTo(BeEmpty())
			for _, dataDisk := range *vm.StorageProfile.DataDisks {
				disk, err := clients.Disks.Get(ctx, clusterResourceGroup, *dataDisk.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(disk.Encryption.Type).To(Equal(mgmtcompute.EncryptionAtRestWithCustomerKey))
			}
		}
	})
})
