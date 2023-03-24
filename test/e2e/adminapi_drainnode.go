package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/drain"
)

var _ = Describe("[Admin API] Cordon and Drain node actions", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should be able to cordon, drain, and uncordon nodes", func(ctx context.Context) {
		By("selecting a worker node in the cluster")
		nodes, err := clients.Kubernetes.CoreV1().Nodes().List(ctx, metav1.ListOptions{
			LabelSelector: "node-role.kubernetes.io/worker",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(len(nodes.Items)).Should(BeNumerically(">", 0))
		node := nodes.Items[0]
		nodeName := node.Name

		drainer := &drain.Helper{
			Ctx:                 ctx,
			Client:              clients.Kubernetes,
			Force:               true,
			GracePeriodSeconds:  -1,
			IgnoreAllDaemonSets: true,
			Timeout:             60 * time.Second,
			DeleteEmptyDirData:  true,
			DisableEviction:     true,
			OnPodDeletedOrEvicted: func(pod *corev1.Pod, usingEviction bool) {
				log.Printf("deleted pod %s/%s", pod.Namespace, pod.Name)
			},
			Out:    log.Writer(),
			ErrOut: log.Writer(),
		}

		defer func() {
			By("uncordoning the node via Kubernetes API")
			err = drain.RunCordonOrUncordon(drainer, &node, false)
			Expect(err).NotTo(HaveOccurred())
		}()

		testCordonNodeOK(ctx, nodeName)
		testDrainNodeOK(ctx, nodeName, drainer)
		testUncordonNodeOK(ctx, nodeName)
	})
})

func testCordonNodeOK(ctx context.Context, nodeName string) {
	By("cordoning the node via RP admin API")
	params := url.Values{
		"shouldCordon": []string{"true"},
		"vmName":       []string{nodeName},
	}
	resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/cordonnode", params, true, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("checking that node was cordoned via Kubernetes API")
	node, err := clients.Kubernetes.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(node.Name).To(Equal(nodeName))
	Expect(node.Spec.Unschedulable).Should(BeTrue())
}

func testUncordonNodeOK(ctx context.Context, nodeName string) {
	By("uncordoning the node via RP admin API")
	params := url.Values{
		"shouldCordon": []string{"false"},
		"vmName":       []string{nodeName},
	}
	resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/cordonnode", params, true, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("checking that node was uncordoned via Kubernetes API")
	node, err := clients.Kubernetes.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(node.Name).To(Equal(nodeName))
	Expect(node.Spec.Unschedulable).Should(BeFalse())
}

func testDrainNodeOK(ctx context.Context, nodeName string, drainer *drain.Helper) {
	By("draining the node via RP admin API")
	params := url.Values{
		"vmName": []string{nodeName},
	}
	resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/drainnode", params, true, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}
