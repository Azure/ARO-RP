package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"testing"

	"github.com/go-test/deep"

	corev1 "k8s.io/api/core/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestNodes(t *testing.T) {
	ctx := context.Background()

	txt, _ := nodesJsonBytes()

	var nodes corev1.NodeList
	err := json.Unmarshal(txt, &nodes)
	if err != nil {
		t.Error(err)
	}

	converted := make([]kruntime.Object, len(nodes.Items))
	for i := range nodes.Items {
		converted[i] = &nodes.Items[i]
	}

	kubernetes := fake.NewSimpleClientset(converted...)

	_, log := testlog.New()

	rf := &realFetcher{
		kubernetesCli: kubernetes,
		log:           log,
	}

	c := &client{fetcher: rf, log: log}

	info, err := c.Nodes(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	expected := &NodeListInformation{
		Nodes: []NodeInformation{
			{
				Name: "aro-master-0",
				Capacity: MachineResources{
					CPU:           "8",
					StorageVolume: "1073189868Ki",
					Memory:        "32933416Ki",
					Pods:          "250",
				},
				Allocatable: MachineResources{
					CPU:           "7500m",
					StorageVolume: "987978038888",
					Memory:        "31782440Ki",
					Pods:          "250",
				},
				Taints: []Taint{
					{
						Key:       "node-role.kubernetes.io/master",
						Value:     "",
						Effect:    "NoSchedule",
						TimeAdded: "",
					},
				},
				Conditions: []NodeConditions{
					{
						Type:    "MemoryPressure",
						Status:  "False",
						Reason:  "KubeletHasSufficientMemory",
						Message: "kubelet has sufficient memory available",
					},
					{
						Type:    "DiskPressure",
						Status:  "False",
						Reason:  "KubeletHasNoDiskPressure",
						Message: "kubelet has no disk pressure",
					},
					{
						Type:    "PIDPressure",
						Status:  "False",
						Reason:  "KubeletHasSufficientPID",
						Message: "kubelet has sufficient PID available",
					},
					{
						Type:    "Ready",
						Status:  "True",
						Reason:  "KubeletReady",
						Message: "kubelet is posting ready status",
					},
				},
				Volumes: make([]Volume, 0),
				Labels: map[string]string{
					"beta.kubernetes.io/arch":                  "amd64",
					"beta.kubernetes.io/instance-type":         "Standard_D8s_v3",
					"beta.kubernetes.io/os":                    "linux",
					"failure-domain.beta.kubernetes.io/region": "eastus",
					"failure-domain.beta.kubernetes.io/zone":   "eastus-1",
					"kubernetes.io/arch":                       "amd64",
					"kubernetes.io/hostname":                   "aro-master-0",
					"kubernetes.io/os":                         "linux",
					"node-role.kubernetes.io/master":           "",
					"node.kubernetes.io/instance-type":         "Standard_D8s_v3",
					"node.openshift.io/os_id":                  "rhcos",
					"topology.kubernetes.io/region":            "eastus",
					"topology.kubernetes.io/zone":              "eastus-1",
				},
				Annotations: map[string]string{
					"machine.openshift.io/machine":                           "openshift-machine-api/aro-master-0",
					"machineconfiguration.openshift.io/currentConfig":        "rendered-master-ebd6f663e22984bdce9081039a6f01c0",
					"machineconfiguration.openshift.io/desiredConfig":        "rendered-master-ebd6f663e22984bdce9081039a6f01c0",
					"machineconfiguration.openshift.io/reason":               "",
					"machineconfiguration.openshift.io/state":                "Done",
					"volumes.kubernetes.io/controller-managed-attach-detach": "true",
				},
			},
		},
	}

	sort.SliceStable(info.Nodes, func(i, j int) bool { return info.Nodes[i].Name < info.Nodes[j].Name })
	sort.SliceStable(expected.Nodes, func(i, j int) bool { return expected.Nodes[i].Name < expected.Nodes[j].Name })

	for i, node := range info.Nodes {
		dateRegex := regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} [\+-]\d{4} \w+`)
		if node.CreatedTime == "" {
			t.Fatal("Node field CreatedTime was null, expected not null")
		}

		if !dateRegex.Match([]byte(node.CreatedTime)) {
			expDateFormat := "2021-08-10 12:21:47 +1000 AEST"
			t.Fatalf("Node field CreatedTime was in incorrect format %v, expected format of %v",
				node.CreatedTime, expDateFormat)
		}
		info.Nodes[i].CreatedTime = ""

		for j, condition := range node.Conditions {
			if condition.LastHeartbeatTime == "" {
				t.Error("Node field LastHeartbeatTime was null, expected not null")
			}

			if !dateRegex.Match([]byte(condition.LastHeartbeatTime)) {
				expDateFormat := "2021-08-10 12:21:47 +1000 AEST"
				t.Fatalf("Node field LastHeartbeatTime was in incorrect format %v, expected format of %v",
					condition.LastHeartbeatTime, expDateFormat)
			}
			info.Nodes[i].Conditions[j].LastHeartbeatTime = ""

			if condition.LastTransitionTime == "" {
				t.Fatal("Node field LastTransitionTime was null, expected not null")
			}

			if !dateRegex.Match([]byte(condition.LastTransitionTime)) {
				expDateFormat := "2021-08-10 12:21:47 +1000 AEST"
				t.Fatalf("Node field LastTransitionTime was in incorrect format %v, expected format of %v",
					condition.LastTransitionTime, expDateFormat)
			}
			info.Nodes[i].Conditions[j].LastTransitionTime = ""
		}
	}

	// No need to check every single node
	for _, r := range deep.Equal(expected.Nodes[0], info.Nodes[0]) {
		t.Error(r)
	}
}
