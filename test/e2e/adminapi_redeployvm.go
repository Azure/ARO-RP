package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
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

var _ = Describe("[Admin API] VM redeploy action", func() {
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
		log.Infof("selected vm: %s", *vm.Name)

		By("triggering the redeploy action")
		clockDrift := -1 * time.Minute
		startTime := time.Now().Add(clockDrift)
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+resourceID+"/redeployvm", url.Values{"vmName": []string{*vm.Name}}, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("verifying through cluster events that the redeployment happened")
		err = wait.PollImmediate(10*time.Second, 10*time.Minute, func() (bool, error) {
			events, err := clients.Kubernetes.EventsV1().Events("default").List(ctx, metav1.ListOptions{})

			if err != nil {
				return false, err
			}

			var nodeKillTime metav1.MicroTime

			for _, event := range events.Items {
				if nodeKillTime.IsZero() &&
					event.Reason == "TerminationStart" &&
					!event.CreationTimestamp.After(startTime) {
					nodeKillTime = metav1.MicroTime(event.CreationTimestamp)
					break
				}

			}

			var nodeNotReady, rebooted, nodeReady bool

			for _, event := range events.Items {
				if event.CreationTimestamp.After(nodeKillTime.Time) && event.Regarding.Name == *vm.Name {
					if !nodeNotReady && event.Reason == "NodeNotReady" {
						nodeNotReady = true
						log.Info("node entered not ready state")
					}
					if !rebooted && event.Reason == "Rebooted" {
						rebooted = true
						log.Info("node reboooted")
					}
					if !nodeReady && event.Reason == "NodeReady" {
						nodeReady = true
						log.Info("node returned to ready state")
						break
					}
				}
			}

			return nodeNotReady && rebooted && nodeReady, nil
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
