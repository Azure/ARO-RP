package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	operatorFlagMetricsTopic  = "cluster.nonstandard.operator.featureflags"
	supportBannerMetricsTopic = "cluster.nonstandard.banner"
)

func (mon *Monitor) emitOperatorFlagsAndSupportBanner(ctx context.Context) error {
	var cont string
	for {
		clusters, err := mon.arocli.AroV1alpha1().Clusters().List(ctx, metav1.ListOptions{Limit: 20, Continue: cont})
		if err != nil {
			return err
		}

		for _, cluster := range clusters.Items {
			if cluster.Spec.OperatorFlags != nil {
				defaultFlags := api.DefaultOperatorFlags()
				nonStandardOperatorFlagDims := make(map[string]string, len(defaultFlags))

				//check if the current set flags matches the default ones
				for defaulFlagName, defaultFlagValue := range defaultFlags {
					currentFlagValue, ok := cluster.Spec.OperatorFlags[defaulFlagName]
					if !ok {
						// if flag is not there, put "DNE - do not exist" in the metrics
						nonStandardOperatorFlagDims[defaulFlagName] = "DNE"
					} else if currentFlagValue != defaultFlagValue {
						// if flag value does not match the defualt one, record the current set value in the metrics
						nonStandardOperatorFlagDims[defaulFlagName] = currentFlagValue
					}
				}
				if len(nonStandardOperatorFlagDims) > 0 {
					mon.emitGauge(operatorFlagMetricsTopic, 1, nonStandardOperatorFlagDims)
				}
			}

			//check if the contact support banner is activated
			if cluster.Spec.Banner.Content == arov1alpha1.BannerContactSupport {
				mon.emitGauge(supportBannerMetricsTopic, 1, map[string]string{"msg": "contact support"})
			}
		}

		cont = clusters.Continue
		if cont == "" {
			break
		}
	}
	return nil
}
