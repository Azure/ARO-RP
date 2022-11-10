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
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	machinefake "github.com/openshift/client-go/machine/clientset/versioned/fake"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestMachines(t *testing.T) {
	ctx := context.Background()

	txt, _ := machinesJsonBytes()

	var machines machinev1beta1.MachineList
	err := json.Unmarshal(txt, &machines)
	if err != nil {
		t.Error(err)
	}

	converted := make([]kruntime.Object, len(machines.Items))
	for i := range machines.Items {
		converted[i] = &machines.Items[i]
	}

	machineClient := machinefake.NewSimpleClientset(converted...)

	_, log := testlog.New()

	rf := &realFetcher{
		machineClient: machineClient,
		log:           log,
	}

	c := &client{fetcher: rf, log: log}

	info, err := c.Machines(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	expected := &MachineListInformation{
		Machines: []MachinesInformation{
			{
				Name:          "aro-v4-shared-gxqb4-master-0",
				LastOperation: "Update",
				Status:        "Running",
				ErrorReason:   "None",
				ErrorMessage:  "None",
			},
		},
	}

	sort.SliceStable(info.Machines, func(i, j int) bool { return info.Machines[i].Name < info.Machines[j].Name })
	sort.SliceStable(expected.Machines, func(i, j int) bool { return expected.Machines[i].Name < expected.Machines[j].Name })

	for i, machine := range info.Machines {
		if machine.CreatedTime == "" {
			t.Fatal("Node field CreatedTime was null, expected not null")
		}
		info.Machines[i].CreatedTime = ""

		if machine.LastUpdated == "" {
			t.Fatal("Machine field LastUpdated was null, expected not null")
		}
		info.Machines[i].LastUpdated = ""

		if machine.LastOperationDate == "" {
			t.Fatal("Node field LastOperationDate was null, expected not null")
		}

		dateRegex := regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} [\+-]\d{4} \w+`)

		if !dateRegex.Match([]byte(machine.LastOperationDate)) {
			expDateFormat := "2021-08-10T12:21:47 +1000 AEST"

			t.Fatalf("Node field LastOperationDate was in incorrect format %v, expected format of %v",
				machine.LastOperationDate, expDateFormat)
		}
		info.Machines[i].LastOperationDate = ""
	}

	// No need to check every single machine
	for _, r := range deep.Equal(expected.Machines[0], info.Machines[0]) {
		t.Fatal(r)
	}
}
