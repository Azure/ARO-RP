package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
)

const (
	dnsTypeMetricsTopic = "cluster.dns.type"
)

// emitDNSType emits the DNS type (dnsmasq or clusterhosted) for the cluster.
func (mon *Monitor) emitDNSType(ctx context.Context) error {
	dnsType := pkgoperator.DNSTypeDnsmasq
	if mon.oc.Properties.OperatorFlags != nil {
		if v, ok := mon.oc.Properties.OperatorFlags[pkgoperator.DNSType]; ok && v != "" {
			dnsType = v
		}
	}

	mon.emitGauge(dnsTypeMetricsTopic, 1, map[string]string{
		"type": dnsType,
	})

	return nil
}
