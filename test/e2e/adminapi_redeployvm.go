package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	uptimeStrFmt = "2006-01-02 15:04:05" // https://go.dev/src/time/format.go
)

var _ = Describe("[Admin API] VM redeploy action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must trigger a selected VM to redeploy", func() {
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

		By("saving the current uptime")
		oldUptime, err := getNodeUptime(*vm.Name)
		Expect(err).NotTo(HaveOccurred())

		By("verifying redeploy action completes without error")
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+resourceID+"/redeployvm", url.Values{"vmName": []string{*vm.Name}}, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("waiting for the redeployed VM to report Running power state in Azure")
		// we can pollimmediate without fear of false positive because we have
		// already waited on the redeploy future
		err = wait.PollImmediate(1*time.Minute, 10*time.Minute, func() (bool, error) {
			restartedVm, err := clients.VirtualMachines.Get(ctx, clusterResourceGroup, *vm.Name, mgmtcompute.InstanceView)
			if err != nil {
				log.Info(fmt.Sprintf("Failed to get restarted vm: %v", err))
				return false, nil // swallow err, retry
			}
			for _, status := range *restartedVm.InstanceView.Statuses {
				if *status.Code == "PowerState/running" {
					return true, nil
				}
			}
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the redeployed node to eventually become Ready in OpenShift")
		// wait 1 minute - this will guarantee we pass the minimum (default) threshold of Node heartbeats (40 seconds)
		err = wait.Poll(1*time.Minute, 10*time.Minute, func() (bool, error) {
			node, err := clients.Kubernetes.CoreV1().Nodes().Get(ctx, *vm.Name, metav1.GetOptions{})
			if err != nil {
				log.Warn(err)
				return false, nil // swallow error, retry
			}
			if ready.NodeIsReady(node) {
				return true, nil
			}
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred())

		By("getting system uptime again and making sure it is newer")
		newUptime, err := getNodeUptime(*vm.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(oldUptime.Before(newUptime)).To(BeTrue())
	})
})

func getNodeUptime(node string) (time.Time, error) {
	// container kernel = node kernel = `uptime` in a Pod reflects the Node as well
	ctx := context.Background()
	namespace := "default"
	name := node
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "cli",
					Image: "image-registry.openshift-image-registry.svc:5000/openshift/cli",
					Command: []string{
						"/bin/sh",
						"-c",
						"uptime -s",
					},
				},
			},
			RestartPolicy: "Never",
			NodeName:      node,
		},
	}

	// Create
	_, err := clients.Kubernetes.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return time.Time{}, err
	}

	// Defer Delete
	defer func() {
		err := clients.Kubernetes.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			log.Error("Could not delete test Pod")
		}
	}()

	// Wait for Completion
	err = wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
		p, err := clients.Kubernetes.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if p.Status.Phase == "Succeeded" {
			return true, nil
		}
		return false, nil // retry
	})
	if err != nil {
		return time.Time{}, err
	}

	// Logs (uptime)
	req := clients.Kubernetes.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{})
	stream, err := req.Stream(ctx)
	if err != nil {
		return time.Time{}, err
	}
	defer stream.Close()
	message := ""
	reader := bufio.NewScanner(stream)
	for reader.Scan() {
		select {
		case <-ctx.Done():
		default:
			line := reader.Text()
			message += line
		}
	}
	return time.Parse(uptimeStrFmt, strings.TrimSpace(message))
}
