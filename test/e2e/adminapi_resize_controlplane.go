package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	masterMachineRoleLabelSelector = "machine.openshift.io/cluster-api-machine-role=master"
	machineLabelInstanceType       = "machine.openshift.io/instance-type"
	nodeLabelInstanceType          = "node.kubernetes.io/instance-type"
)

func getControlPlaneVMs(ctx context.Context) []compute.VirtualMachine {
	oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
	Expect(err).NotTo(HaveOccurred())
	clusterResourceGroup := stringutils.LastTokenByte(*oc.ClusterProfile.ResourceGroupID, '/')
	vms, err := clients.VirtualMachines.List(ctx, clusterResourceGroup)
	Expect(err).NotTo(HaveOccurred())
	return slices.DeleteFunc(vms, func(vm compute.VirtualMachine) bool {
		Expect(vm.Name).ToNot(BeNil())
		return !strings.Contains(*vm.Name, "master")
	})
}

// getControlPlaneVMSize retrieves the VM size of one of the control plane
// (master) VMs in the cluster by listing all VMs in the cluster resource group
// and returning the size of the first VM whose name contains "master".
func getControlPlaneVMSize(ctx context.Context) string {
	vms := getControlPlaneVMs(ctx)
	Expect(vms).NotTo(BeEmpty())
	Expect(vms[0].HardwareProfile).NotTo(BeNil())
	return string(vms[0].HardwareProfile.VMSize)
}

// nextLargerSupportedMasterVMSize returns the supported master VM size in the
// same family as currentVMSize that has the smallest core count strictly
// greater than currentVMSize's core count. It returns an error if currentVMSize
// is not in the supported master list, or if no larger size exists in the same
// family.
func nextLargerSupportedMasterVMSize(currentVMSize string) (string, error) {
	supportedMasterSizes := validate.SupportedVMSizesByRole(validate.VMRoleMaster)
	currentInfo, ok := supportedMasterSizes[api.VMSize(currentVMSize)]
	if !ok {
		return "", fmt.Errorf("current VM size %q is not in the supported master list", currentVMSize)
	}

	targetSku := ""
	targetCores := math.MaxInt
	for size, info := range supportedMasterSizes {
		if info.Family != currentInfo.Family {
			continue
		}
		if info.CoreCount <= currentInfo.CoreCount {
			continue
		}
		if info.CoreCount < targetCores {
			targetCores = info.CoreCount
			targetSku = string(size)
		}
	}

	if targetSku == "" {
		return "", fmt.Errorf("no supported master VM size larger than %q (family %s, %d cores) is available", currentVMSize, currentInfo.Family, currentInfo.CoreCount)
	}
	return targetSku, nil
}

// validateMasterVMSizeLabels makes sure that master machine and node Resources in the cluster have the correct vmsize labels. It verifies that the following are equal to the targetSku
// - metadata.labels."machine.openshift.io/instance-type" for machine
// - spec.ProviderSpec.value.vmSize for machine
// - metadata.labels."node.kubernetes.io/instance-type" for node
// for each of the master nodes
//
// There is no return value, as this is supposed to be called directly from ginkgo test cases. This function validates the labels via [github.com/onsi/gomega.Expect] statements
func validateMasterVMSizeLabels(ctx context.Context, targetSku string) {
	masterMachinesList, err := clients.MachineAPI.MachineV1beta1().Machines("openshift-machine-api").List(ctx, metav1.ListOptions{
		LabelSelector: masterMachineRoleLabelSelector,
	})
	Expect(err).ToNot(HaveOccurred())

	for _, ma := range masterMachinesList.Items {
		By(fmt.Sprintf("Checking machine and node labels for %s", ma.GetName()))
		sizeLabelVal, ok := ma.GetObjectMeta().GetLabels()[machineLabelInstanceType]
		Expect(ok).To(BeTrue())
		Expect(sizeLabelVal).To(Equal(targetSku))

		var machineProvSpec machinev1beta1.AzureMachineProviderSpec
		Expect(json.Unmarshal(ma.Spec.ProviderSpec.Value.Raw, &machineProvSpec)).ToNot(HaveOccurred())
		Expect(machineProvSpec.VMSize).To(Equal(targetSku))

		Expect(ma.Status.NodeRef).ToNot(BeNil())

		var curNode corev1.Node
		err = clients.KubeClient.Get(ctx, types.NamespacedName{Name: ma.Status.NodeRef.Name}, &curNode)
		Expect(err).ToNot(HaveOccurred())

		nodeSizeLabelVal, ok := curNode.GetLabels()[nodeLabelInstanceType]
		Expect(ok).To(BeTrue())
		Expect(nodeSizeLabelVal).To(Equal(targetSku))
	}
}

var _ = Describe("[Admin API] Resize control plane", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should reject an unsupported VM size", func(ctx context.Context) {
		params := url.Values{
			"vmSize":       []string{"Standard_Invalid_Fake"},
			"deallocateVM": []string{"true"},
		}

		resp, err := adminRequest(ctx, http.MethodPost,
			"/admin"+clusterResourceID+"/resizecontrolplane",
			params, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	})

	It("should reject a request with missing vmSize", func(ctx context.Context) {
		params := url.Values{
			"deallocateVM": []string{"true"},
		}

		resp, err := adminRequest(ctx, http.MethodPost,
			"/admin"+clusterResourceID+"/resizecontrolplane",
			params, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	})

	It("should not resize when size is already the same", func(ctx context.Context) {
		By("Getting the current machine size")
		preResizeVMSize := getControlPlaneVMSize(ctx)
		Expect(preResizeVMSize).ToNot(BeZero())

		By(fmt.Sprintf("Resizing to the current machine size: %s", preResizeVMSize))

		params := url.Values{
			"deallocateVM": []string{"false"},
			"vmSize":       []string{preResizeVMSize},
		}

		resp, err := adminRequest(ctx, http.MethodPost,
			"/admin"+clusterResourceID+"/resizecontrolplane",
			params, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		controlPlaneVms := getControlPlaneVMs(ctx)
		Expect(controlPlaneVms).ToNot(BeEmpty())
		for _, vm := range controlPlaneVms {
			Expect(vm.HardwareProfile).ToNot(BeNil())
			Expect(string(vm.HardwareProfile.VMSize)).To(Equal(preResizeVMSize))
		}
	})

	It("Should not attempt to resize if there is no quota", func(ctx context.Context) {
		By("Finding a supported Master VM Size without Quota")
		usageRes, err := clients.Usages.List(ctx, _env.Location())
		Expect(err).ToNot(HaveOccurred())
		supportedSizes := validate.SupportedVMSizesByRole("master")
		// looking for supported vms with 0 quota
		targetSku := ""
		for size, sizeInfo := range supportedSizes {
			for _, u := range usageRes {
				if u.Name == nil ||
					u.Name.Value == nil ||
					*u.Name.Value != sizeInfo.Family ||
					u.Limit == nil {
					continue
				}

				if *u.Limit == 0 {
					targetSku = size.String()
				}
			}
		}

		if targetSku == "" {
			Skip("Can't run test. No supported SKU without quota found")
		}

		By(fmt.Sprintf("Trying to resize controlplane vms to %s", targetSku))
		params := url.Values{
			"deallocateVM": []string{"false"},
			"vmSize":       []string{targetSku},
		}

		out := api.CloudError{}
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/resizecontrolplane", params, true, nil, &out)

		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		Expect(out.Message).To(Equal("Pre-flight validation failed."))
		Expect(out.Details).To(HaveLen(1))
		Expect(out.Details[0].Code).To(Equal("ResourceQuotaExceeded"))
	})

	It("should do the resize when target size is different", Label(slow), FlakeAttempts(1), Serial, func(ctx context.Context) {
		By("Getting the current machine size")
		preResizeVMSize := getControlPlaneVMSize(ctx)
		Expect(preResizeVMSize).ToNot(BeZero())

		// Pick the next-larger VM size within the same family from the
		// supported-master list. This keeps the resize on a well-tested size
		// while avoiding arbitrary family swaps.
		targetSku, err := nextLargerSupportedMasterVMSize(preResizeVMSize)
		if err != nil {
			Skip(err.Error())
		}

		By(fmt.Sprintf("Resizing from %s to %s", preResizeVMSize, targetSku))
		params := url.Values{
			"deallocateVM": []string{"false"},
			"vmSize":       []string{targetSku},
		}

		requestError := api.CloudError{}
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/resizecontrolplane", params, true, nil, &requestError)
		// err will be [io.EOF] when request is successful, as response body will
		// be empty. In case of error, response body will be parsed into an [api.CloudError]
		if err == io.EOF {
			err = nil
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(requestError.Code).To(BeEmpty(), "unexpected CloudError: %+v", requestError)
		Expect(requestError.Message).To(BeEmpty(), "unexpected CloudError: %+v", requestError)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("Validating vm size after resize")
		controlPlaneVms := getControlPlaneVMs(ctx)
		Expect(controlPlaneVms).ToNot(BeEmpty())
		for _, vm := range controlPlaneVms {
			Expect(vm.HardwareProfile).ToNot(BeNil())
			Expect(string(vm.HardwareProfile.VMSize)).To(Equal(targetSku))
			Expect(vm.ProvisioningState).ToNot(BeNil())
			Expect(*vm.ProvisioningState).To(Equal(string(compute.ProvisioningStateSucceeded)))
		}

		By("Validating machine and node labels")
		validateMasterVMSizeLabels(ctx, targetSku)
	}, NodeTimeout(30*time.Minute))
})
