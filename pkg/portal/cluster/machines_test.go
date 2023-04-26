package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	machinefake "github.com/openshift/client-go/machine/clientset/versioned/fake"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
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

type MockClientFactory struct {
	mockResourcesClient       *mock_features.MockResourcesClient
	mockVirtualMachinesClient *mock_compute.MockVirtualMachinesClient
}

func newMockClientFactory(mockResourcesClient *mock_features.MockResourcesClient, mockVirtualMachinesClient *mock_compute.MockVirtualMachinesClient) MockClientFactory {
	return MockClientFactory{mockResourcesClient: mockResourcesClient, mockVirtualMachinesClient: mockVirtualMachinesClient}
}

func (mcf MockClientFactory) NewResourcesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) features.ResourcesClient {
	return mcf.mockResourcesClient
}

func (mcf MockClientFactory) NewVirtualMachinesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) compute.VirtualMachinesClient {
	return mcf.mockVirtualMachinesClient
}

func TestVMAllocationStatus(t *testing.T) {
	ctx := context.Background()
	mockSubscriptionDoc := &api.SubscriptionDocument{
		ID:          "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		ResourceID:  "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		Timestamp:   1668689726,
		Self:        "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		ETag:        "\"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz\"",
		Attachments: "attachments/",
		Subscription: &api.Subscription{
			State: "Registered",
			Properties: &api.SubscriptionProperties{
				TenantID: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
				RegisteredFeatures: []api.RegisteredFeatureProfile{
					{Name: "Microsoft.RedHatOpenShift/RedHatEngineering", State: "Registered"},
				},
			},
		},
	}

	controller := gomock.NewController(t)
	mockEnv := mock_env.NewMockInterface(controller)
	mockResourcesClient := mock_features.NewMockResourcesClient(controller)
	mockVirtualMachinesClient := mock_compute.NewMockVirtualMachinesClient(controller)
	mockAROEnvironment := &azureclient.AROEnvironment{
		Environment: azure.Environment{
			ResourceManagerEndpoint: "temp",
		},
	}
	type test struct {
		name                            string
		returnedVM                      mgmtcompute.VirtualMachine
		returnedGenericResourceExpanded []mgmtfeatures.GenericResourceExpanded
		returnedAROEnvironment          *azureclient.AROEnvironment
		wantOutput                      map[string]string
		wantErr                         error
	}

	for _, tt := range []*test{
		{
			name: "Successfully fetching VMs allocation status. Calling all the required methods.",
			returnedGenericResourceExpanded: []mgmtfeatures.GenericResourceExpanded{
				{
					Kind: func(v string) *string { return &v }("something"),
					Type: func(v string) *string { return &v }("Microsoft.Compute/virtualMachines"),
					Name: func(v string) *string { return &v }("master-x"),
				},
			},
			returnedVM: mgmtcompute.VirtualMachine{
				Name: func(v string) *string { return &v }("master-x"),
				VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
					InstanceView: &mgmtcompute.VirtualMachineInstanceView{
						Statuses: &[]mgmtcompute.InstanceViewStatus{
							{
								Code: func() *string {
									s := new(string)
									*s = "PowerState/running"
									return s
								}(),
							},
						},
					},
				},
			},
			returnedAROEnvironment: mockAROEnvironment,
			wantOutput:             map[string]string{"master-x": "PowerState/running"},
			wantErr:                nil,
		},
		{
			name:                            "No VM resource found",
			returnedGenericResourceExpanded: []mgmtfeatures.GenericResourceExpanded{},
			returnedAROEnvironment:          mockAROEnvironment,
			wantOutput:                      map[string]string{},
			wantErr:                         nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mockEnv.EXPECT().Environment().Return(tt.returnedAROEnvironment)
			mockResourcesClient.EXPECT().ListByResourceGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tt.returnedGenericResourceExpanded, tt.wantErr)
			if len(tt.returnedGenericResourceExpanded) > 0 {
				mockVirtualMachinesClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.returnedVM, tt.wantErr)
			}
			_, log := testlog.New()
			mcf := newMockClientFactory(mockResourcesClient, mockVirtualMachinesClient)
			azureSideFetcher := &azureSideFetcher{
				log:                          log,
				resourceGroupName:            "someResourceGroup",
				subscriptionDoc:              mockSubscriptionDoc,
				env:                          mockEnv,
				resourceClientFactory:        mcf,
				virtualMachinesClientFactory: mcf,
			}
			azureClient := &azureClient{fetcher: azureSideFetcher, log: log}
			vmAllocationStatuses, err := azureClient.VMAllocationStatus(ctx)
			if !reflect.DeepEqual(vmAllocationStatuses, tt.wantOutput) {
				t.Error("Expected output", tt.wantOutput, "Got", vmAllocationStatuses)
			}
			if err != nil && err != tt.wantErr || err == nil && tt.wantErr != nil {
				t.Error("Expected", tt.wantErr, "Got", err)
			}
		})
	}
}
