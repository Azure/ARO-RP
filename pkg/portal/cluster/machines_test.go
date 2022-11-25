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
	"github.com/golang/mock/gomock"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	machinefake "github.com/openshift/client-go/machine/clientset/versioned/fake"

	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	testlog "github.com/Azure/ARO-RP/test/util/log"

	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"

	//mgmtcompute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_refreshable "github.com/Azure/ARO-RP/pkg/util/mocks/refreshable"
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

func TestVMAllocationStatus(t *testing.T) {
	//ctx := context.Background()

	type test struct {
		name    string
		mocks   func(*test, *mock_compute.MockVirtualMachinesClient, *mock_env.MockInterface, *mock_refreshable.MockAuthorizer)
		wantErr string
	}
	for _, tt := range []*test{
		{
			name: "allow when there's enough resources - limits set to exact requirements, offset by 100 of current value",
			mocks: func(tt *test, cuc *mock_compute.MockVirtualMachinesClient, env *mock_env.MockInterface, authorizer *mock_refreshable.MockAuthorizer) {
				// cuc.EXPECT().List(ctx, "someResourceGroup").Return([]mgmtcompute.VirtualMachine{{Name: func() *string {
				// 	s := new(string)
				// 	*s = "vm1"
				// 	return s
				// }(), VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{InstanceView: &mgmtcompute.VirtualMachineInstanceView{Statuses: &[]mgmtcompute.InstanceViewStatus{{Code: func() *string {
				// 	s := new(string)
				// 	*s = "PowerState/running"
				// 	return s
				// }()}}}}}}, nil)
				env.EXPECT().FPAuthorizer("someString", "someEndpoint").Return(authorizer, nil)
				env.EXPECT().Environment()
			},
			wantErr: "",
		},
		// {
		// 	name:    "not enough cores",
		// 	wantErr: "400: ResourceQuotaExceeded: : Resource quota of cores exceeded. Maximum allowed: 204, Current in use: 101, Additional requested: 104.",
		// 	mocks: func(tt *test, cuc *mock_compute.MockVirtualMachinesClient) {
		// 		cuc.EXPECT().
		// 			List(ctx, "ocLocation").
		// 			Return([]mgmtcompute.Usage{
		// 				{
		// 					Name: &mgmtcompute.UsageName{
		// 						Value: to.StringPtr("cores"),
		// 					},
		// 					CurrentValue: to.Int32Ptr(101),
		// 					Limit:        to.Int64Ptr(204),
		// 				},
		// 			}, nil)
		// 	},
		// },
		// {
		// 	name:    "not enough premium disks",
		// 	wantErr: "400: ResourceQuotaExceeded: : Resource quota of PremiumDiskCount exceeded. Maximum allowed: 113, Current in use: 101, Additional requested: 13.",
		// 	mocks: func(tt *test, cuc *mock_compute.MockVirtualMachinesClient) {
		// 		cuc.EXPECT().
		// 			List(ctx, "ocLocation").
		// 			Return([]mgmtcompute.Usage{
		// 				{
		// 					Name: &mgmtcompute.UsageName{
		// 						Value: to.StringPtr("PremiumDiskCount"),
		// 					},
		// 					CurrentValue: to.Int32Ptr(101),
		// 					Limit:        to.Int64Ptr(113),
		// 				},
		// 			}, nil)
		// 	},
		// },
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			ctx := context.Background()

			computeVirtualMachineClient := mock_compute.NewMockVirtualMachinesClient(controller)
			mockEnv := mock_env.NewMockInterface(controller)
			mockRefreshable := mock_refreshable.NewMockAuthorizer(controller)

			subscriptionDoc := &api.SubscriptionDocument{
				ID:          "fe16a035-e540-4ab7-80d9-373fa9a3d6ae",
				ResourceID:  "MrEvAJyKauIBAAAAAAAAAA==",
				Timestamp:   1668689726,
				Self:        "dbs/MrEvAA==/colls/MrEvAJyKauI=/docs/MrEvAJyKauIBAAAAAAAAAA==/",
				ETag:        "\"c4006fe2-0000-0100-0000-63762f3e0000\"",
				Attachments: "attachments/",
				Subscription: &api.Subscription{
					State: "Registered",
					Properties: &api.SubscriptionProperties{
						TenantID: "64dc69e4-d083-49fc-9569-ebece1dd1408",
						RegisteredFeatures: []api.RegisteredFeatureProfile{
							{Name: "Microsoft.RedHatOpenShift/RedHatEngineering", State: "Registered"},
						},
					},
				},
			}

			if tt.mocks != nil {
				tt.mocks(tt, computeVirtualMachineClient, mockEnv, mockRefreshable)
			}

			// oc := &api.OpenShiftCluster{
			// 	Location: "ocLocation",
			// 	Properties: api.OpenShiftClusterProperties{
			// 		Install: &api.Install{
			// 			Phase: api.InstallPhaseBootstrap,
			// 		},
			// 		MasterProfile: api.MasterProfile{
			// 			VMSize: "Standard_D8s_v3",
			// 		},
			// 		WorkerProfiles: []api.WorkerProfile{
			// 			{
			// 				VMSize: "Standard_D8s_v3",
			// 				Count:  10,
			// 			},
			// 		},
			// 	},
			// }

			_, log := testlog.New()

			azureSideFetcher := azureSideFetcher{
				resourceGroupName: "someResourceGroup",
				env:               mockEnv,
				subscriptionDoc:   subscriptionDoc,
			}

			realFetcher := &realFetcher{
				log:              log,
				azureSideFetcher: azureSideFetcher,
			}

			client := &client{fetcher: realFetcher, log: log}

			vmAllocationStatus, err := client.VMAllocationStatus(ctx)
			t.Log(vmAllocationStatus)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

		})
	}
}
