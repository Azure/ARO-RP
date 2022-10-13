package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	operatorFlagMetricsTopic  = "cluster.nonstandard.operator.featureflags"
	supportBannerMetricsTopic = "cluster.nonstandard.contact.support.banner"
)

func (mon *Monitor) emitOperatorFlagsAndSupportBanner(ctx context.Context) error {
	cs, err := mon.listAROClusters(ctx)
	if err != nil {
		return err
	}

	for _, cluster := range cs.Items {
		if cluster.Spec.OperatorFlags != nil {
			defualtFlags := api.DefaultOperatorFlags()
			nonStandardOperatorFlags := make(map[string]string)
			//check if the current set flags matches the default ones
			for flag, status := range cluster.Spec.OperatorFlags {
				if defualtFlags[flag] != status {
					nonStandardOperatorFlags[flag] = status
				}
			}
			if len(nonStandardOperatorFlags) > 0 {
				mon.emitGauge(operatorFlagMetricsTopic, 1, nonStandardOperatorFlags)
			}
		}

		//check if the contact support banner is activated
		if cluster.Spec.Banner.Content == arov1alpha1.BannerContactSupport {
			mon.emitGauge(supportBannerMetricsTopic, 1, nil)
		}
	}

	return nil
}
