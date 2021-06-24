package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Clean the remaining PVCs in openshift-monitoring namespace.
// These PVCs with labels: app=prometheus,prometheus=k8s are left
// behind after switching back to use emptydir as persistent storage
// for prometheus by disabling featureflag in monitoing controller.
// This workaround is in effect for all clusters set to
// have non-persistent prometheus.
// The cleanup loop removes only up to 2 PVCs as this is the
// production configuration at the time of the workaround release.

import (
	"context"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/monitoring"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	prometheusLabels    = "app=prometheus,prometheus=k8s"
	monitoringName      = "cluster-monitoring-config"
	monitoringNamespace = "openshift-monitoring"
)

type cleanPromPVC struct {
	log *logrus.Entry
	cli kubernetes.Interface
}

func NewCleanFromPVCWorkaround(log *logrus.Entry, cli kubernetes.Interface) Workaround {
	return &cleanPromPVC{
		log: log,
		cli: cli,
	}
}

func (*cleanPromPVC) Name() string {
	return "Clean prometheus PVC after disabling persistency"
}

func (c *cleanPromPVC) IsRequired(clusterVersion *version.Version) bool {
	cm, err := c.cli.CoreV1().ConfigMaps(monitoringNamespace).Get(context.Background(), monitoringName, metav1.GetOptions{})
	if err != nil {
		return false
	}

	configDataJSON, err := yaml.YAMLToJSON([]byte(cm.Data["config.yaml"]))
	if err != nil {
		return false
	}
	var configData monitoring.Config
	handle := new(codec.JsonHandle)
	err = codec.NewDecoderBytes(configDataJSON, handle).Decode(&configData)
	if err != nil {
		return false
	}

	if configData.PrometheusK8s.Retention == nil && reflect.DeepEqual(configData.PrometheusK8s.VolumeClaimTemplate, struct{ api.MissingFields }{}) {
		return true
	}

	return false
}

func (c *cleanPromPVC) Ensure(ctx context.Context) error {

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pvcList, err := c.cli.CoreV1().PersistentVolumeClaims(monitoringNamespace).List(ctx, metav1.ListOptions{LabelSelector: prometheusLabels})
		if err != nil {
			return err
		}

		for _, pvc := range pvcList.Items {
			err = c.cli.CoreV1().PersistentVolumeClaims(monitoringNamespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (c *cleanPromPVC) Remove(ctx context.Context) error {
	return nil
}
