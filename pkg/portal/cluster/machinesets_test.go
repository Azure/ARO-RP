package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/go-test/deep"

	kruntime "k8s.io/apimachinery/pkg/runtime"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	machinefake "github.com/openshift/client-go/machine/clientset/versioned/fake"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestMachineSets(t *testing.T) {
	ctx := context.Background()

	txt, _ := machinesetsJsonBytes()

	var machineSets machinev1beta1.MachineSetList
	err := json.Unmarshal(txt, &machineSets)
	if err != nil {
		t.Error(err)
	}

	converted := make([]kruntime.Object, len(machineSets.Items))
	for i := range machineSets.Items {
		converted[i] = &machineSets.Items[i]
	}

	machineClient := machinefake.NewSimpleClientset(converted...)

	_, log := testlog.New()

	rf := &realFetcher{
		machineClient: machineClient,
		log:           log,
	}

	c := &client{fetcher: rf, log: log}

	info, err := c.MachineSets(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	for i, machineSet := range machineSets.Items {
		if i >= len(info.MachineSets) {
			t.Fatal(err)
		}
		info.MachineSets[i].CreatedAt = machineSet.CreationTimestamp.In(time.UTC).String()
	}

	expected := &MachineSetListInformation{
		MachineSets: []MachineSetsInformation{
			{
				Name:                     "aro-v4-shared-gxqb4-infra-eastus1",
				Type:                     "infra",
				CreatedAt:                "2021-03-09 13:48:16 +0000 UTC",
				DesiredReplicas:          0,
				Replicas:                 0,
				ErrorReason:              "None",
				ErrorMessage:             "None",
				PublicLoadBalancerName:   "aro-v4-shared-gxqb4",
				OSDiskAccountStorageType: "Premium_LRS",
				VNet:                     "vnet",
				Subnet:                   "worker-subnet",
				VMSize:                   "Standard_D4s_v3",
			},
		},
	}

	sort.SliceStable(info.MachineSets, func(i, j int) bool { return info.MachineSets[i].Replicas < info.MachineSets[j].Replicas })
	sort.SliceStable(expected.MachineSets, func(i, j int) bool { return expected.MachineSets[i].Replicas < expected.MachineSets[j].Replicas })

	// No need to check every single machine
	for _, r := range deep.Equal(expected.MachineSets[0], info.MachineSets[0]) {
		t.Error(r)
	}
}
