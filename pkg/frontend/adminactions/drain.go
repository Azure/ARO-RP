package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/drain"
)

func (k *kubeActions) CordonNode(ctx context.Context, nodeName string, shouldCordon bool) error {
	node, err := k.kubecli.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	drainLogWriter := k.log.Writer()
	defer func() {
		_ = drainLogWriter.Close()
	}()

	drainer := &drain.Helper{
		Ctx:                 ctx,
		Client:              k.kubecli,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		Timeout:             60 * time.Second,
		DeleteEmptyDirData:  true,
		DisableEviction:     true,
		OnPodDeletedOrEvicted: func(pod *corev1.Pod, usingEviction bool) {
			k.log.Infof("deleted pod %s/%s", pod.Namespace, pod.Name)
		},
		Out:    drainLogWriter,
		ErrOut: drainLogWriter,
	}

	return drain.RunCordonOrUncordon(drainer, node, shouldCordon)
}

func (k *kubeActions) DrainNode(ctx context.Context, nodeName string) error {
	drainLogWriter := k.log.Writer()
	defer func() {
		_ = drainLogWriter.Close()
	}()

	drainer := &drain.Helper{
		Ctx:                 ctx,
		Client:              k.kubecli,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		Timeout:             3 * time.Minute,
		DeleteEmptyDirData:  true,
		DisableEviction:     true,
		OnPodDeletedOrEvicted: func(pod *corev1.Pod, usingEviction bool) {
			k.log.Infof("deleted pod %s/%s", pod.Namespace, pod.Name)
		},
		Out:    drainLogWriter,
		ErrOut: drainLogWriter,
	}

	return drain.RunNodeDrain(drainer, nodeName)
}
