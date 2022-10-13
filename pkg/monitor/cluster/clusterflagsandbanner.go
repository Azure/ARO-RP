package cluster

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	operatorFlagMetricsTopic  = "cluster.nonstandard.aro.operator.featureflags"
	supportBannerMetricsTopic = "cluster.nonstandard.aro.contact.support.banner"
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
			for flag, status := range cluster.Spec.OperatorFlags {
				if defualtFlags[flag] != status {
					nonStandardOperatorFlags[flag] = status
				}
			}
			if len(nonStandardOperatorFlags) > 0 {
				mon.emitGauge(operatorFlagMetricsTopic, 1, nonStandardOperatorFlags)
			}
		}
		if cluster.Spec.Banner.Content == arov1alpha1.BannerContactSupport {
			mon.emitGauge(supportBannerMetricsTopic, 1, nil)
		}
	}

	return nil
}
