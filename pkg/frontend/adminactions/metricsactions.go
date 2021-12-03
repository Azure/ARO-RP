package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// MetricActions are those that involve k8s objects, and thus depend upon k8s clients being createable
type MetricActions interface {
	TopPods(ctx context.Context, namespace string) ([]byte, error)
	TopNodes(ctx context.Context) ([]byte, error)
}

type metricActions struct {
	log *logrus.Entry
	oc  *api.OpenShiftCluster

	configcli metricsclient.Interface
}

// NewMetricActions returns a metricActions
func NewMetricActions(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster) (MetricActions, error) {
	restConfig, err := restconfig.RestConfig(env, oc)
	if err != nil {
		return nil, err
	}

	configcli, err := metricsclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &metricActions{
		log: log,
		oc:  oc,

		configcli: configcli,
	}, nil
}

func (m *metricActions) TopPods(ctx context.Context, namespace string) ([]byte, error) {

	un, err := m.configcli.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{Limit: 1000})
	if err != nil {
		return nil, err
	}

	return un.Marshal()
}

func (m *metricActions) TopNodes(ctx context.Context) ([]byte, error) {

	un, err := m.configcli.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{Limit: 1000})
	if err != nil {
		return nil, err
	}

	return un.Marshal()
}
