package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/go-test/deep"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	kruntime "k8s.io/apimachinery/pkg/runtime"

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

	// lol golang
	converted := make([]kruntime.Object, len(machineSets.Items))
	for i := range machineSets.Items {
		converted[i] = &machineSets.Items[i]
	}

	maoclient := maofake.NewSimpleClientset(converted...)

	_, log := testlog.New()

	rf := &realFetcher{
		maoclient: maoclient,
		log:       log,
	}

	c := &client{fetcher: rf, log: log}

	info, err := c.MachineSets(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	expected := &MachineSetListInformation{
		MachineSets: []MachineSetsInformation{
			{
				Name:            "aro-v4-shared-gxqb4-infra-eastus1",
				Type:            "infra",
				CreatedAt:       "2021-03-09T13:48:16Z",
				DesiredReplicas: 0,
				Replicas:        0,
				ErrorReason:     "None",
				ErrorMessage:    "None",
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
