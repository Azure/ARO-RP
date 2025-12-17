package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var _ = Describe("Load Balancer", Label(smoke, ocpupgrade), Ordered, func() {
	var objs []unstructured.Unstructured
	BeforeAll(func(ctx context.Context) {
		// Initialize objs to avoid objs being counted multiple times when retrying.
		objs = []unstructured.Unstructured{}
		By("creating a load balancer")
		lb, err := staticResources.Open("static_resources/loadbalancer.yaml")
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			_ = lb.Close()
		}()

		dec := yaml.NewYAMLOrJSONDecoder(lb, 4096)
		// It can't load multiple objects from a single file, so we need to loop through the file and load them one by one.
		for {
			var obj unstructured.Unstructured
			err := dec.Decode(&obj)
			if errors.Is(err, io.EOF) {
				break
			}
			Expect(err).NotTo(HaveOccurred())
			objs = append(objs, obj)
		}

		for _, obj := range objs {
			By(fmt.Sprintf("creating %s/%s", obj.GetNamespace(), obj.GetName()))
			cli, err := clients.Dynamic.GetClient(&obj)
			Expect(err).NotTo(HaveOccurred())
			CreateK8sObjectWithRetry(ctx, cli.Create, &obj, metav1.CreateOptions{})
		}
	})

	It("should create a service", func(ctx context.Context) {
		var ip string
		By("waiting for the service to be created and get an IP address")
		Eventually(func(g Gomega, ctx context.Context) {
			svc, err := clients.Kubernetes.CoreV1().Services("default").Get(ctx, "test-lb", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc.Status.LoadBalancer.Ingress).NotTo(BeEmpty())
			ip = svc.Status.LoadBalancer.Ingress[0].IP
		}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())

		By("checking the service is reachable and it returns a 200 OK")
		Eventually(func(g Gomega, ctx context.Context) {
			client := http.Client{Timeout: 10 * time.Second}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s", ip), nil)
			g.Expect(err).NotTo(HaveOccurred())
			resp, err := client.Do(req)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resp.StatusCode).To(Equal(http.StatusOK))
		}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())
	})

	AfterAll(func(ctx context.Context) {
		for _, obj := range objs {
			By(fmt.Sprintf("deleting %s/%s", obj.GetNamespace(), obj.GetName()))
			cli, err := clients.Dynamic.GetClient(&obj)
			Expect(err).NotTo(HaveOccurred())
			CleanupK8sResource[*unstructured.Unstructured](ctx, cli, obj.GetName())
		}
	})
})
