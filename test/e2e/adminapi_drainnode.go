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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/kubectl/pkg/drain"
)

var _ = Describe("[Admin API] Cordon and Drain node actions", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should be able to cordon, drain, and uncordon nodes", func() {
		By("selecting a worker node in the cluster")
		nodes, err := clients.Kubernetes.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
			LabelSelector: "node-role.kubernetes.io/worker",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(len(nodes.Items)).Should(BeNumerically(">", 0))
		node := nodes.Items[0]
		nodeName := node.Name

		drainer := &drain.Helper{
			Ctx:                 context.Background(),
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

		testCordonNodeOK(nodeName)
		testDrainNodeOK(nodeName, drainer)
		testUncordonNodeOK(nodeName)
	})
})

func testCordonNodeOK(nodeName string) {
	By("cordoning the node via RP admin API")
	params := url.Values{
		"shouldCordon": []string{"true"},
		"vmName":       []string{nodeName},
	}
	resp, err := adminRequest(context.Background(), http.MethodPost, "/admin"+resourceIDFromEnv()+"/cordonnode", params, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("checking that node was cordoned via Kubernetes API")
	node, err := clients.Kubernetes.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(node.Name).To(Equal(nodeName))
	Expect(node.Spec.Unschedulable).Should(BeTrue())
}

func testUncordonNodeOK(nodeName string) {
	By("uncordoning the node via RP admin API")
	params := url.Values{
		"shouldCordon": []string{"false"},
		"vmName":       []string{nodeName},
	}
	resp, err := adminRequest(context.Background(), http.MethodPost, "/admin"+resourceIDFromEnv()+"/cordonnode", params, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("checking that node was uncordoned via Kubernetes API")
	node, err := clients.Kubernetes.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(node.Name).To(Equal(nodeName))
	Expect(node.Spec.Unschedulable).Should(BeFalse())
}

func testDrainNodeOK(nodeName string, drainer *drain.Helper) {
	By("counting the number of pods on the node via Kubernetes API")
	podsListPreDrain, err := clients.Kubernetes.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String(),
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(len(podsListPreDrain.Items)).Should(BeNumerically(">", 0))

	By("counting the number of pods to be deleted/evicted via Kubernetes API")
	podsForDeletion, errs := drainer.GetPodsForDeletion(nodeName)
	Expect(errs).To(BeNil())
	Expect(len(podsForDeletion.Pods())).Should(BeNumerically(">", 0))

	By("draining the node via RP admin API")
	params := url.Values{
		"shouldCordon": []string{"true"},
		"vmName":       []string{nodeName},
	}
	resp, err := adminRequest(context.Background(), http.MethodPost, "/admin"+resourceIDFromEnv()+"/drainnode", params, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("counting the number of pods on the drained node via Kubernetes API")
	podsListPostDrain, err := clients.Kubernetes.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String(),
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(len(podsListPostDrain.Items)).Should(BeNumerically(">", 0))

	By("checking that the expected number of pods exist on the drained node")
	Expect(len(podsListPostDrain.Items)).Should(BeNumerically("<=", len(podsListPreDrain.Items)-len(podsForDeletion.Pods())))
}
