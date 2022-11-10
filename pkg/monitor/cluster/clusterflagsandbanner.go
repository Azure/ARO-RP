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
	supportBannerMetricsTopic = "cluster.nonstandard.contact.support.banner"
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
				nonStandardOperatorFlags := make(map[string]string, len(defaultFlags))

				//check if the current set flags matches the default ones
				for name, value := range defaultFlags {
					flag, ok := cluster.Spec.OperatorFlags[name]
					if !ok {
						// if flag is not there, put "not exist" in the metrics
						nonStandardOperatorFlags[name] = "not exist"
					} else if flag != value {
						// if flag value does not match the defualt one, record the current set value in the metrics
						nonStandardOperatorFlags[name] = flag
					}
				}

				for flag, status := range cluster.Spec.OperatorFlags {
					if defaultFlags[flag] != status {
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

		cont = clusters.Continue
		if cont == "" {
			break
		}
	}
	return nil
}
