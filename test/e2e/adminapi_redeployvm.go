package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var _ = XDescribe("[Admin API] VM redeploy action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should trigger a selected VM to redeploy", func() {
		ctx := context.Background()
		resourceID := resourceIDFromEnv()

		By("getting the resource group where the VM instances live in")
		oc, err := clients.OpenshiftClustersv20200430.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		clusterResourceGroup := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')

		By("picking the first VM to redeploy")
		vms, err := clients.VirtualMachines.List(ctx, clusterResourceGroup)
		Expect(err).NotTo(HaveOccurred())
		Expect(vms).NotTo(HaveLen(0))
		vm := vms[0]

		By("triggering the redeploy action")
		clockDrift := -1 * time.Minute
		startTime := time.Now().Add(clockDrift)
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+resourceID+"/redeployvm", url.Values{"vmName": []string{*vm.Name}}, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("verifying through Azure activity logs that the redeployment happened")
		// Can be material delays in shipping activity logs, hence healthy 15 min wait.
		// https://docs.microsoft.com/en-us/azure/azure-monitor/logs/data-ingestion-time#azure-activity-logs-resource-logs-and-metrics
		err = wait.PollImmediate(10*time.Second, 20*time.Minute, func() (bool, error) {
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

		By("waiting for all nodes to be ready")
		err = wait.PollImmediate(10*time.Second, 10*time.Minute, func() (bool, error) {
			nodes, err := clients.Kubernetes.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
			if err != nil {
				log.Warn(err)
				return false, nil // swallow error
			}

			for _, node := range nodes.Items {
				if !ready.NodeIsReady(&node) {
					return false, nil
				}
			}

			return true, nil
		})
		Expect(err).NotTo(HaveOccurred())
	})
})
