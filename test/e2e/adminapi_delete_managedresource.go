package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

var testStorageClass = storagev1.StorageClass{
	ObjectMeta: metav1.ObjectMeta{
		Name: "test-storageclass",
	},
	Provisioner: "disk.csi.azure.com",
	Parameters: map[string]string{
		"skuname": "Premium_LRS",
	},
	ReclaimPolicy: pointerutils.ToPtr(corev1.PersistentVolumeReclaimDelete),
	// Immediate binding so it creates it without us having to make a pod
	VolumeBindingMode: pointerutils.ToPtr(storagev1.VolumeBindingImmediate),
}

var diskPVC = corev1.PersistentVolumeClaim{
	ObjectMeta: metav1.ObjectMeta{
		Name: "testdisk-pvc",
	},
	Spec: corev1.PersistentVolumeClaimSpec{
		StorageClassName: pointerutils.ToPtr("test-storageclass"),
		AccessModes: []corev1.PersistentVolumeAccessMode{
			corev1.ReadWriteOnce,
		},
		// 1GB
		Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: *resource.NewScaledQuantity(1, resource.Giga)}},
	},
}

var _ = Describe("[Admin API] Delete managed resource action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should be possible to delete managed cluster resources", func(ctx context.Context) {
		var pvc *corev1.PersistentVolumeClaim
		const namespace = "default"

		By("creating a disk storage class")
		stc := clients.Kubernetes.StorageV1().StorageClasses()
		CreateK8sObjectWithRetry(ctx, stc.Create, &testStorageClass, metav1.CreateOptions{})

		By("creating a disk pvc")
		pvcs := clients.Kubernetes.CoreV1().PersistentVolumeClaims(namespace)
		CreateK8sObjectWithRetry(ctx, pvcs.Create, &diskPVC, metav1.CreateOptions{})

		defer func() {
			By("cleaning up the k8s pvc")
			CleanupK8sResource(ctx, pvcs, diskPVC.Name)
			By("cleaning up the storageclass")
			CleanupK8sResource(ctx, stc, testStorageClass.Name)
		}()

		// wait for disk to be created
		Eventually(func(g Gomega, ctx context.Context) {
			pvc = GetK8sObjectWithRetry(ctx, pvcs.Get, diskPVC.Name, metav1.GetOptions{})
			g.Expect(pvc.Status.Phase).To(Equal(corev1.ClaimBound))
			g.Expect(pvc.Spec.VolumeName).ToNot(BeEmpty())
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		By("getting the newly created pvc info")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		pvcID := *oc.ClusterProfile.ResourceGroupID + "/providers/Microsoft.Compute/disks/" + pvc.Spec.VolumeName

		By("deleting the underlying PVC")
		testDeleteManagedResourceOK(ctx, pvcID)
	})

	It("should NOT be possible to delete a resource not within the cluster's managed resource group", func(ctx context.Context) {
		By("trying to delete the master subnet")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/deletemanagedresource", url.Values{"managedResourceID": []string{*oc.MasterProfile.SubnetID}}, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	})

	It("should NOT be possible to delete the private link service in the cluster's managed resource group", func(ctx context.Context) {
		By("trying to delete the private link service")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		// Fake name prevents accidentally deleting the PLS but still validates guardrail logic works.
		plsResourceID := fmt.Sprintf("%s/providers/Microsoft.Network/PrivateLinkServices/%s", *oc.ClusterProfile.ResourceGroupID, "fake-pls")

		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/deletemanagedresource", url.Values{"managedResourceID": []string{plsResourceID}}, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	})
})

func testDeleteManagedResourceOK(ctx context.Context, resourceID string) {
	resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/deletemanagedresource", url.Values{"managedResourceID": []string{resourceID}}, true, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}
