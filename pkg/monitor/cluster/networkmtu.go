package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// emitNetworkMTU collects and emits metrics related to cluster network MTU configuration
func (mon *Monitor) emitNetworkMTU(ctx context.Context) error {
	networkConfig, err := mon.configcli.ConfigV1().Networks().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		// Convention: if the Network CR is missing, emit a metric instead of erroring
		// so dashboards remain populated. Use "mtu=default" and "network_type=unknown"
		// to explicitly signal the absence of cluster-reported MTU data.
		mon.emitGauge("network.mtu", 1, map[string]string{
			"mtu":          "default",
			"network_type": "unknown",
		})
		if mon.hourlyRun {
			mon.log.WithFields(logrus.Fields{
				"mtu":          "default",
				"network_type": "unknown",
			}).Info("network MTU configuration")
		}
		return nil
	}

	mtuString := strconv.Itoa(networkConfig.Status.ClusterNetworkMTU)

	mon.emitGauge("network.mtu", 1, map[string]string{
		"mtu":          mtuString,
		"network_type": networkConfig.Spec.NetworkType,
	})

	if mon.hourlyRun {
		mon.log.WithFields(logrus.Fields{
			"mtu":          mtuString,
			"network_type": networkConfig.Spec.NetworkType,
		}).Info("network MTU configuration")
	}
	return nil
}
