package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

// ContainerMetrics is used to store individual container resource usage (not directly used in current code).
type ContainerMetrics struct {
	Name   string `json:"name"`
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// PodMetrics contains resource usage and percentage for a pod.
type PodMetrics struct {
	Namespace        string  `json:"namespace"`
	PodName          string  `json:"podName"`
	NodeName         string  `json:"nodeName"`
	CPUUsage         string  `json:"cpuUsage"`
	MemoryUsage      string  `json:"memoryUsage"`
	CPUPercentage    float64 `json:"cpuPercentage"`
	MemoryPercentage float64 `json:"memoryPercentage"`
}

// NodeMetrics contains resource usage and percentage for a node.
type NodeMetrics struct {
	NodeName         string  `json:"nodeName"`
	CPUUsage         string  `json:"cpuUsage"`
	MemoryUsage      string  `json:"memoryUsage"`
	CPUPercentage    float64 `json:"cpuPercentage"`
	MemoryPercentage float64 `json:"memoryPercentage"`
}

// calculatePercentage returns usage as a percentage of total resources.
func calculatePercentage(usage string, total int64) float64 {
	usageInt, _ := resource.ParseQuantity(usage)
	var usageInt64 int64
	if strings.HasSuffix(usage, "m") && !strings.HasSuffix(usage, "Mi") {
		usageInt64 = usageInt.MilliValue()
	} else {
		usageInt64 = usageInt.Value()
	}
	return float64(usageInt64) / float64(total) * 100
}

// TopPods fetches and returns pod-level metrics (CPU/memory usage + percent) across all namespaces.
// If `allNamespaces` is false, an error is returned.
func (k *kubeActions) TopPods(ctx context.Context, restConfig *restclient.Config, allNamespaces bool) ([]PodMetrics, error) {
	var ns string
	if allNamespaces {
		ns = "" // Empty string fetches pods from all namespaces
	} else {
		return nil, fmt.Errorf("explicit namespace must be provided when allNamespaces is false")
	}

	// Create metrics client
	client, err := metricsv1beta1.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	// Get metrics for all pods
	podMetricsList, err := client.MetricsV1beta1().PodMetricses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Map pods to their assigned nodes
	podsList, err := k.kubecli.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	podToNode := make(map[string]string)
	for _, pod := range podsList.Items {
		podToNode[pod.Name] = pod.Spec.NodeName
	}

	// Fetch node capacities
	nodeList, err := k.kubecli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nodeCPUMap := make(map[string]int64)
	nodeMemMap := make(map[string]int64)
	for _, node := range nodeList.Items {
		if cpuQty, ok := node.Status.Capacity["cpu"]; ok {
			if cpuVal, ok := cpuQty.AsInt64(); ok {
				nodeCPUMap[node.Name] = cpuVal
			}
		}
		if memQty, ok := node.Status.Capacity["memory"]; ok {
			if memVal, ok := memQty.AsInt64(); ok {
				nodeMemMap[node.Name] = memVal
			}
		}
	}

	var result []PodMetrics

	// Aggregate metrics per pod
	for _, item := range podMetricsList.Items {
		var totalCPUMilli int64
		var totalMemBytes int64

		for _, c := range item.Containers {
			if c.Usage.Cpu() != nil {
				totalCPUMilli += c.Usage.Cpu().MilliValue() // CPU in millicores
			}
			if c.Usage.Memory() != nil {
				totalMemBytes += c.Usage.Memory().Value() // Memory in bytes
			}
		}

		// Skip pods with no usage
		if totalCPUMilli == 0 && totalMemBytes == 0 {
			continue
		}

		// 🔐 Filter to only include "openshift-" namespaces
		if !strings.HasPrefix(item.Namespace, "openshift-") {
			continue
		}

		nodeName := podToNode[item.Name]
		nodeCPU := nodeCPUMap[nodeName]
		nodeMem := nodeMemMap[nodeName]

		// Compute CPU and memory usage percent relative to node capacity
		cpuPct := 0.0
		memPct := 0.0
		if nodeCPU > 0 {
			cpuPct = (float64(totalCPUMilli) / float64(nodeCPU*1000)) * 100
		}
		if nodeMem > 0 {
			memPct = (float64(totalMemBytes) / float64(nodeMem)) * 100
		}

		// Append formatted pod metrics
		result = append(result, PodMetrics{
			Namespace:        item.Namespace,
			PodName:          item.Name,
			NodeName:         nodeName,
			CPUUsage:         fmt.Sprintf("%dm", totalCPUMilli),
			MemoryUsage:      fmt.Sprintf("%dKi", totalMemBytes/1024),
			CPUPercentage:    roundPercentage(cpuPct),
			MemoryPercentage: roundPercentage(memPct),
		})
	}

	// Sort pods by memory percentage (highest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].MemoryPercentage > result[j].MemoryPercentage
	})

	return result, nil
}

// roundPercentage returns a float rounded to 2 decimal places
func roundPercentage(val float64) float64 {
	if val == 0 {
		return 0
	}
	return math.Round(val*100) / 100
}

// TopNodes fetches node-level metrics (CPU/memory usage + percent)
func (k *kubeActions) TopNodes(ctx context.Context, restConfig *restclient.Config) ([]NodeMetrics, error) {
	client, err := metricsv1beta1.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	// Get metrics for all nodes
	nodeMetrics, err := client.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []NodeMetrics
	for _, item := range nodeMetrics.Items {
		// Get full node object to retrieve capacity
		node, err := k.kubecli.CoreV1().Nodes().Get(ctx, item.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		totalCPUQuantity := node.Status.Capacity["cpu"]
		totalMemoryQuantity := node.Status.Capacity["memory"]

		totalCPU, _ := totalCPUQuantity.AsInt64()       // in cores
		totalMemory, _ := totalMemoryQuantity.AsInt64() // in bytes

		var usageCPU, usageMemory string
		if item.Usage.Cpu() != nil {
			usageCPU = item.Usage.Cpu().String()
		}
		if item.Usage.Memory() != nil {
			usageMemory = item.Usage.Memory().String()
		}

		// Compute usage percentages; convert node CPU capacity to millicores
		percentageCPU := calculatePercentage(usageCPU, totalCPU*1000)
		percentageMemory := calculatePercentage(usageMemory, totalMemory)

		// Append node metrics
		result = append(result, NodeMetrics{
			NodeName:         item.Name,
			CPUUsage:         usageCPU,
			MemoryUsage:      usageMemory,
			CPUPercentage:    roundPercentage(percentageCPU),
			MemoryPercentage: roundPercentage(percentageMemory),
		})
	}

	// Sort nodes by CPU usage percentage
	sort.Slice(result, func(i, j int) bool {
		return result[i].CPUPercentage > result[j].CPUPercentage
	})

	return result, nil
}
