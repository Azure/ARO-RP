package metrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/prometheus/client_golang/prometheus"
	runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	metricsTag = "arooperator"

	endpointLabel = "endpoint_url"
	roleLabel     = "role"
)

var (
	metricServicePrincipalValid = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: metricsTag,
		Name:      "service_principal_valid",
		Help:      "ARO Service Principal is Valid",
	}, []string{})
	metricRequiredEndpointAccessible = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: metricsTag,
		Name:      "required_endpoint_accessible",
		Help:      "The required endpoint is accessible from nodes of the specified role",
	}, []string{endpointLabel, roleLabel})
	metricIngressCertificateValid = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: metricsTag,
		Name:      "ingress_certificate_valid",
		Help:      "ARO Ingress Certificate is Valid",
	}, []string{})
	metricDnsConfigurationValid = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: metricsTag,
		Name:      "dns_configuration_valid",
		Help:      "ARO DNS Configuration is Valid",
	}, []string{})
)

type Client interface {
	UpdateServicePrincipalValid(valid bool)
	UpdateRequiredEndpointAccessible(endpoint, role string, accessible bool)
	UpdateIngressCertificateValid(valid bool)
	UpdateDnsConfigurationValid(valid bool)
}

type client struct{}

func NewClient() Client {
	return &client{}
}

func (m *client) UpdateServicePrincipalValid(valid bool) {
	metricServicePrincipalValid.
		With(prometheus.Labels{}).
		Set(toFloat(valid))
}

func (m *client) UpdateRequiredEndpointAccessible(endpoint, role string, accessible bool) {
	metricRequiredEndpointAccessible.
		With(prometheus.Labels{
			endpointLabel: endpoint,
			roleLabel:     role,
		}).
		Set(toFloat(accessible))
}

func (m *client) UpdateIngressCertificateValid(valid bool) {
	metricIngressCertificateValid.
		With(prometheus.Labels{}).
		Set(toFloat(valid))
}

func (m *client) UpdateDnsConfigurationValid(valid bool) {
	metricDnsConfigurationValid.
		With(prometheus.Labels{}).
		Set(toFloat(valid))
}

func toFloat(b bool) float64 {
	return map[bool]float64{false: 0, true: 1}[b]
}

func RegisterMetrics() {
	runtimemetrics.Registry.MustRegister(metricServicePrincipalValid)
	runtimemetrics.Registry.MustRegister(metricRequiredEndpointAccessible)
	runtimemetrics.Registry.MustRegister(metricIngressCertificateValid)
	runtimemetrics.Registry.MustRegister(metricDnsConfigurationValid)
}
