package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Alerts", Label(smoke, basichealth), Serial, func() {
	It("should not be firing", func(ctx context.Context) {
		var host string
		Eventually(func(g Gomega, ctx context.Context) {
			route, err := clients.Route.RouteV1().Routes("openshift-monitoring").Get(ctx, "prometheus-k8s", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(route.Spec.Host).NotTo(BeEmpty())
			host = route.Spec.Host
		}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())

		var token *authenticationv1.TokenRequest
		var err error
		Eventually(func(g Gomega, ctx context.Context) {
			token, err = clients.Kubernetes.CoreV1().ServiceAccounts("openshift-monitoring").
				CreateToken(ctx, "prometheus-k8s", &authenticationv1.TokenRequest{}, metav1.CreateOptions{})
			g.Expect(err).NotTo(HaveOccurred())
		}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())

		// Skip TLS verification in dev env
		var roundTripper http.RoundTripper
		if _env.IsLocalDevelopmentMode() {
			rt := api.DefaultRoundTripper.(*http.Transport).Clone()
			rt.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			roundTripper = rt
		} else {
			roundTripper = api.DefaultRoundTripper
		}

		client, err := api.NewClient(api.Config{
			Address: fmt.Sprintf("https://%s", host),
			Client: &http.Client{
				Transport: config.NewAuthorizationCredentialsRoundTripper(
					"Bearer",
					config.NewInlineSecret(token.Status.Token),
					roundTripper,
				),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		promAPI := prometheusv1.NewAPI(client)

		Eventually(func(g Gomega, ctx context.Context) {
			result, err := promAPI.Alerts(ctx)
			g.Expect(err).NotTo(HaveOccurred())
			for _, alert := range result.Alerts {
				if alert.State != prometheusv1.AlertStateFiring {
					continue
				}
				g.Expect(alert).To(Satisfy(isIgnorable))
			}
		}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())
	})
})

func isIgnorable(alert prometheusv1.Alert) bool {
	severity := []model.LabelValue{"critical", "error", "warning"}
	if !slices.Contains(severity, alert.Labels["severity"]) {
		return true
	}
	// In prod, all alerts shouldn't be firing
	if !_env.IsLocalDevelopmentMode() {
		return false
	}
	// In dev, ignore mdsd pods alerts
	switch alert.Labels["alertname"] {
	case "KubePodCrashLooping":
		return alert.Labels["namespace"] == "openshift-azure-logging" && alert.Labels["container"] == "mdsd"
	case "KubeDaemonSetRolloutStuck":
		return alert.Labels["namespace"] == "openshift-azure-logging" && alert.Labels["daemonset"] == "mdsd"
	}
	return false
}
