package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/drain"
)

const (
	drainMaxRetries = 3
	drainRetryDelay = 2 * time.Second
)

func (k *kubeActions) CordonNode(ctx context.Context, nodeName string, shouldCordon bool) error {
	node, err := k.kubecli.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

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
			log.Printf("deleted pod %s/%s", pod.Namespace, pod.Name)
		},
		Out:    log.Writer(),
		ErrOut: log.Writer(),
	}

	return drain.RunCordonOrUncordon(drainer, node, shouldCordon)
}

func (k *kubeActions) DrainNode(ctx context.Context, nodeName string) error {
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
			log.Printf("deleted pod %s/%s", pod.Namespace, pod.Name)
		},
		Out:    log.Writer(),
		ErrOut: log.Writer(),
	}

	return drain.RunNodeDrain(drainer, nodeName)
}

func (k *kubeActions) DrainNodeWithRetries(ctx context.Context, nodeName string) error {
	var lastErr error
	for attempt := 0; attempt <= drainMaxRetries; attempt++ {
		err := k.DrainNode(ctx, nodeName)
		if err == nil {
			return nil
		}
		lastErr = err
		remaining := drainMaxRetries - attempt
		if remaining > 0 {
			k.log.Infof("Drain attempt %d failed for %s: %v. Retrying %d more times.", attempt+1, nodeName, err, remaining)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(drainRetryDelay):
			}
		}
	}
	return fmt.Errorf("could not drain node after %d retries: %w", drainMaxRetries, lastErr)
}
