package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func pickFirstWorkerVM(ctx context.Context, resourceGroup string) (*mgmtcompute.VirtualMachine, error) {
	vms, err := Clients.VirtualMachines.List(ctx, resourceGroup)
	if err != nil {
		return nil, err
	}

	// Iterate over and return first worker VM that is Succeeded provisioning
	for _, v := range vms {
		if strings.Contains(*v.Name, "-worker-") && *v.ProvisioningState == "Succeeded" {
			return &v, nil
		}
	}

	return nil, nil
}

func pollUntilProvisioningState(ctx context.Context, resourceGroup string, vmName string, desiredState string, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var err error

	err = wait.PollImmediateUntil(5*time.Second, func() (bool, error) {
		vm, err := Clients.VirtualMachines.Get(ctx, resourceGroup, vmName, mgmtcompute.InstanceView)
		if err != nil {
			return true, err
		}

		if *vm.ProvisioningState == desiredState {
			return true, nil
		}

		return false, nil
	}, timeoutCtx.Done())

	return err
}

var _ = Describe("Admin actions", func() {
	BeforeEach(runAdminTestsInDevOnly)

	Specify("Redeploy VM", func() {
		ctx := context.Background()

		// Get the resourcegroup that the VM instances live in
		oc, err := Clients.OpenshiftClusters.Get(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("CLUSTER"))
		Expect(err).NotTo(HaveOccurred())
		rg := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')

		// Pick the first worker VM to redeploy
		vm, err := pickFirstWorkerVM(ctx, rg)
		Expect(err).NotTo(HaveOccurred())
		Expect(vm).NotTo(Equal(nil))

		// Trigger a redeploy action
		_, err = adminRequest("POST", "redeployvm", "", nil, "vmName="+*vm.Name)
		Expect(err).NotTo(HaveOccurred())

		// Wait until the VM has been re-provisioned
		err = pollUntilProvisioningState(ctx, rg, *vm.Name, "Succeeded", 10*time.Minute)
		Expect(err).NotTo(HaveOccurred())
	})
})
