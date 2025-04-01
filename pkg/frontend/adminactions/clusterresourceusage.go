package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

// ContainerMetrics is used to store metrics for containers
type ContainerMetrics struct {
	Name   string `json:"name"`
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// PodMetrics represents the metrics for a pod
type PodMetrics struct {
	Namespace  string             `json:"namespace"`
	PodName    string             `json:"podName"`
	Containers []ContainerMetrics `json:"containers"`
}

// NodeMetrics represents the metrics for a node
type NodeMetrics struct {
	NodeName         string  `json:"nodeName"`
	CPUUsage         string  `json:"cpuUsage"`
	MemoryUsage      string  `json:"memoryUsage"`
	CPUPercentage    float64 `json:"cpuPercentage"`
	MemoryPercentage float64 `json:"memoryPercentage"`
}

// calculatePercentage calculates the percentage of resource usage.
func calculatePercentage(usage string, total int64) float64 {
	usageInt, _ := resource.ParseQuantity(usage)
	usageInt64 := usageInt.Value()
	return float64(usageInt64) / float64(total) * 100
}

// TopPods fetches the resource usage for all pods (or across namespaces) and calculates their usage.
func (k *kubeActions) TopPods(ctx context.Context, restConfig *restclient.Config, allNamespaces bool) ([]PodMetrics, error) {
	var ns string
	if allNamespaces {
		ns = ""
	} else {
		// prevent unsafe defaulting to "default"
		return nil, fmt.Errorf("explicit namespace must be provided when allNamespaces is false")
	}

	client, err := metricsv1beta1.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	podMetrics, err := client.MetricsV1beta1().PodMetricses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []PodMetrics
	for _, item := range podMetrics.Items {
		var containers []ContainerMetrics
		for _, c := range item.Containers {
			var cpu, memory string
			if c.Usage.Cpu() != nil {
				cpu = c.Usage.Cpu().String()
			}
			if c.Usage.Memory() != nil {
				memory = c.Usage.Memory().String()
			}

			containers = append(containers, ContainerMetrics{
				Name:   c.Name,
				CPU:    cpu,
				Memory: memory,
			})
		}

		result = append(result, PodMetrics{
			Namespace:  item.Namespace,
			PodName:    item.Name,
			Containers: containers,
		})
	}

	// Filter out pods with 0 CPU and 0 Memory
	var filtered []PodMetrics
	for _, pod := range result {
		hasUsage := false
		for _, c := range pod.Containers {
			if c.CPU != "0" && c.Memory != "0" {
				hasUsage = true
				break
			}
		}
		if hasUsage {
			filtered = append(filtered, pod)
		}
	}

	// Sort by total CPU usage (descending)
	sort.Slice(filtered, func(i, j int) bool {
		sumCPU := func(containers []ContainerMetrics) int64 {
			var total int64
			for _, c := range containers {
				q, err := resource.ParseQuantity(c.CPU)
				if err == nil {
					total += q.MilliValue()
				}
			}
			return total
		}
		return sumCPU(filtered[i].Containers) > sumCPU(filtered[j].Containers)
	})

	return filtered, nil
}

// TopNodes fetches the resource usage for all nodes in the cluster and calculates their usage percentage.
func (k *kubeActions) TopNodes(ctx context.Context, restConfig *restclient.Config) ([]NodeMetrics, error) {
	client, err := metricsv1beta1.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	nodeMetrics, err := client.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []NodeMetrics
	for _, item := range nodeMetrics.Items {
		node, err := k.kubecli.CoreV1().Nodes().Get(ctx, item.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		totalCPUQuantity := node.Status.Capacity["cpu"]
		totalMemoryQuantity := node.Status.Capacity["memory"]

		totalCPU, _ := totalCPUQuantity.AsInt64()
		totalMemory, _ := totalMemoryQuantity.AsInt64()

		var usageCPU, usageMemory string
		if item.Usage.Cpu() != nil {
			usageCPU = item.Usage.Cpu().String()
		}
		if item.Usage.Memory() != nil {
			usageMemory = item.Usage.Memory().String()
		}

		percentageCPU := calculatePercentage(usageCPU, totalCPU)
		percentageMemory := calculatePercentage(usageMemory, totalMemory)

		result = append(result, NodeMetrics{
			NodeName:         item.Name,
			CPUUsage:         usageCPU,
			MemoryUsage:      usageMemory,
			CPUPercentage:    percentageCPU,
			MemoryPercentage: percentageMemory,
		})
	}

	// Sort by CPU usage percentage descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].CPUPercentage > result[j].CPUPercentage
	})

	return result, nil
}
