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

	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	uptimeStrFmt = "2006-01-02 15:04:05" // https://go.dev/src/time/format.go
)

var _ = Describe("[Admin API] VM redeploy action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must trigger a selected VM to redeploy", func(ctx context.Context) {
		By("getting the resource group where the VM instances live in")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		clusterResourceGroup := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')

		By("picking the first VM to redeploy")
		vms, err := clients.VirtualMachines.List(ctx, clusterResourceGroup)
		Expect(err).NotTo(HaveOccurred())
		Expect(vms).NotTo(HaveLen(0))
		vm := vms[0]
		log.Infof("selected vm: %s", *vm.Name)

		By("saving the current uptime")
		oldUptime, err := getNodeUptime(Default, ctx, *vm.Name)
		Expect(err).NotTo(HaveOccurred())

		By("triggering VM redeployment via RP Admin API")
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/redeployvm", url.Values{"vmName": []string{*vm.Name}}, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("waiting for the redeployed VM to report Running power state in Azure")
		Eventually(func(g Gomega, ctx context.Context) {
			restartedVm, err := clients.VirtualMachines.Get(ctx, clusterResourceGroup, *vm.Name, mgmtcompute.InstanceView)
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(*restartedVm.InstanceView.Statuses).To(ContainElement(HaveField("Code", HaveValue(Equal("PowerState/running")))))
		}).WithContext(ctx).WithTimeout(10 * time.Minute).WithPolling(time.Minute).Should(Succeed())

		By("waiting for the redeployed node to eventually become Ready in OpenShift")
		// wait 1 minute - this will guarantee we pass the minimum (default) threshold of Node heartbeats (40 seconds)
		Eventually(func(g Gomega, ctx context.Context) {
			node, err := clients.Kubernetes.CoreV1().Nodes().Get(ctx, *vm.Name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(ready.NodeIsReady(node)).To(BeTrue())
		}).WithContext(ctx).WithTimeout(10 * time.Minute).WithPolling(time.Minute).Should(Succeed())

		By("getting system uptime again and making sure it is newer")
		newUptime, err := getNodeUptime(Default, ctx, *vm.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(newUptime).To(BeTemporally(">", oldUptime))
	})
})

func getNodeUptime(g Gomega, ctx context.Context, node string) (time.Time, error) {
	// container kernel = node kernel = `uptime` in a Pod reflects the Node as well
	namespace := "default"
	name := fmt.Sprintf("%s-uptime-%d", node, GinkgoParallelProcess())
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

	By("creating uptime pod")
	_, err := clients.Kubernetes.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return time.Time{}, err
	}

	// Defer Delete
	defer func() {
		By("deleting uptime pod")
		err := clients.Kubernetes.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			log.Error("Could not delete test Pod")
		}
	}()

	By("waiting for uptime pod to move into the Succeeded phase")
	g.Eventually(func(g Gomega, ctx context.Context) {
		p, err := clients.Kubernetes.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(p.Status.Phase).To(Equal(corev1.PodSucceeded))
	}).WithContext(ctx).Should(Succeed())

	By("getting logs")
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
