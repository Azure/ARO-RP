package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Router", Label(smoke, ocpupgrade), Ordered, func() {
	BeforeAll(func(ctx context.Context) {
		By("creating a load balancer")
		f, err := staticResources.Open("static_resources/route.yaml")
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err = f.Close()
			Expect(err).NotTo(HaveOccurred())
		}()
		objs, err := loadResourcesFromYaml(f)
		Expect(err).NotTo(HaveOccurred())
		createResources(ctx, objs...)

		DeferCleanup(func(ctx context.Context) {
			cleanupResources(ctx, objs...)
		})
	})

	It("should create a route", func(ctx context.Context) {
		const namespace = "route-test"
		var host string
		By("waiting for the service to be created and get an endpoint")
		Eventually(func(g Gomega, ctx context.Context) {
			route, err := clients.Route.RouteV1().Routes(namespace).Get(ctx, "test-route", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(route.Spec.Host).NotTo(BeEmpty())
			host = route.Spec.Host
		}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())

		By("checking the service is reachable and it returns a 200 OK")
		Eventually(func(g Gomega, ctx context.Context) {
			client := http.Client{Timeout: 10 * time.Second}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s", host), nil)
			g.Expect(err).NotTo(HaveOccurred())
			resp, err := client.Do(req)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resp.StatusCode).To(Equal(http.StatusOK))
		}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())
	})
})
