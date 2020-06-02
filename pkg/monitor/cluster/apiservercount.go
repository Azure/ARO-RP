package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitAPIServerCount(ctx context.Context) error {
	ds, err := mon.cli.AppsV1().DaemonSets("openshift-apiserver").Get("apiserver", metav1.GetOptions{})
	if err != nil {
		mon.emitGauge("apiserver.openshift.count", 0, map[string]string{})
		return nil
	}
	if ds.Status.NumberAvailable != ds.Status.DesiredNumberScheduled {
		mon.emitGauge("apiserver.openshift.count", int64(ds.Status.NumberAvailable), map[string]string{})
	}

	pods, err := mon.cli.CoreV1().Pods("openshift-kube-apiserver").List(metav1.ListOptions{
		LabelSelector: "app=openshift-kube-apiserver",
	})
	if int64(len(pods.Items)) != 3 {
		mon.emitGauge("apiserver.kube.count", int64(len(pods.Items)), map[string]string{})
	}

	return nil
}
