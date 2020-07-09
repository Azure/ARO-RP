package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var _ = Describe("[Admin API] VM redeploy action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should trigger a selected VM to redeploy", func() {
		ctx := context.Background()
		resourceID := resourceIDFromEnv()
		resourceGroup := os.Getenv("RESOURCEGROUP")
		resourceName := os.Getenv("CLUSTER")

		By("getting the resource group where the VM instances live in")
		oc, err := clients.OpenshiftClusters.Get(ctx, resourceGroup, resourceName)
		Expect(err).NotTo(HaveOccurred())
		clusterResourceGroup := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')

		By("picking the first VM to redeploy")
		vms, err := clients.VirtualMachines.List(ctx, clusterResourceGroup)
		Expect(err).NotTo(HaveOccurred())
		Expect(vms).NotTo(HaveLen(0))
		vm := vms[0]

		By("triggering the redeploy action")
		startTime := time.Now()
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+resourceID+"/redeployvm", url.Values{"vmName": []string{*vm.Name}}, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("verifying through Azure activity logs that the redeployment happened")
		err = wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
			filter := fmt.Sprintf(
				"eventTimestamp ge '%s' and resourceId eq '%s'",
				startTime.Format(time.RFC3339),
				*vm.ID,
			)

			activityLogs, err := clients.ActivityLogs.List(ctx, filter, "status,operationName")
			if err != nil {
				return false, err
			}

			var count int
			for _, activityLog := range activityLogs {
				if *activityLog.OperationName.Value == "Microsoft.Compute/virtualMachines/redeploy/action" &&
					*activityLog.Status.Value == "Succeeded" {
					count++
				}
			}

			return count == 1, nil
		})
		Expect(err).NotTo(HaveOccurred())
	})
})
