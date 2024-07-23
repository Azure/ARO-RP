package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeConditions struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	LastHeartbeatTime  string `json:"lastHeartbeatTime,omitempty"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
}

type Taint struct {
	Key       string `json:"key"`
	Value     string `json:"value,omitempty"`
	Effect    string `json:"effect"`
	TimeAdded string `json:"timeAdded,omitempty"`
}

type MachineResources struct {
	CPU           string `json:"cpu"`
	StorageVolume string `json:"storageVolume"`
	Memory        string `json:"memory"`
	Pods          string `json:"pods"`
}

type Volume struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type NodeInformation struct {
	Name        string            `json:"name"`
	CreatedTime string            `json:"createdTime"`
	Capacity    MachineResources  `json:"capacity"`
	Volumes     []Volume          `json:"volumes"`
	Allocatable MachineResources  `json:"allocatable"`
	Taints      []Taint           `json:"taints"`
	Conditions  []NodeConditions  `json:"conditions"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type NodeListInformation struct {
	Nodes []NodeInformation `json:"nodes"`
}

func NodesFromNodeList(nodes *corev1.NodeList) *NodeListInformation {
	final := &NodeListInformation{
		Nodes: make([]NodeInformation, len(nodes.Items)),
	}

	for i, node := range nodes.Items {
		nodeInformation := NodeInformation{
			Name:        node.Name,
			CreatedTime: node.CreationTimestamp.String(),
			Capacity: MachineResources{
				CPU:           node.Status.Capacity.Cpu().String(),
				StorageVolume: node.Status.Capacity.StorageEphemeral().String(),
				Memory:        node.Status.Capacity.Memory().String(),
				Pods:          node.Status.Capacity.Pods().String(),
			},
			Allocatable: MachineResources{
				CPU:           node.Status.Allocatable.Cpu().String(),
				StorageVolume: node.Status.Allocatable.StorageEphemeral().String(),
				Memory:        node.Status.Allocatable.Memory().String(),
				Pods:          node.Status.Allocatable.Pods().String(),
			},
			Taints:      getTaints(node),
			Conditions:  getNodeConditions(node),
			Volumes:     getVolumes(node),
			Labels:      node.Labels,
			Annotations: node.Annotations,
		}
		final.Nodes[i] = nodeInformation
	}

	return final
}

func (f *realFetcher) Nodes(ctx context.Context) (*NodeListInformation, error) {
	r, err := f.kubernetesCli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return NodesFromNodeList(r), nil
}

func (c *client) Nodes(ctx context.Context) (*NodeListInformation, error) {
	return c.fetcher.Nodes(ctx)
}

// Helping Functions
func getNodeConditions(node corev1.Node) []NodeConditions {
	conditions := []NodeConditions{}
	for _, condition := range node.Status.Conditions {
		conditions = append(conditions, NodeConditions{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastHeartbeatTime:  condition.LastHeartbeatTime.String(),
			LastTransitionTime: condition.LastTransitionTime.String(),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}
	return conditions
}

func getTaints(node corev1.Node) []Taint {
	taints := []Taint{}
	for _, taint := range node.Spec.Taints {
		timeAdded := ""
		if taint.TimeAdded != nil {
			timeAdded = taint.TimeAdded.String()
		}
		taints = append(taints, Taint{
			Key:       taint.Key,
			Value:     taint.Value,
			Effect:    string(taint.Effect),
			TimeAdded: timeAdded,
		})
	}
	return taints
}

func getVolumes(node corev1.Node) []Volume {
	volumes := []Volume{}
	for _, volume := range node.Status.VolumesAttached {
		volumes = append(volumes, Volume{
			Name: string(volume.Name),
			Path: volume.DevicePath,
		})
	}
	return volumes
}
