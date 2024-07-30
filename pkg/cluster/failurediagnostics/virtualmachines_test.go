package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestVirtualMachinesSerialConsole(t *testing.T) {
	const (
		key            = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
		clusterProfile = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupCluster"
	)

	oc := &api.OpenShiftClusterDocument{
		Key: strings.ToLower(key),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: key,
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: clusterProfile,
				},
				StorageSuffix: "PrefixHere",
			},
		},
	}

	for _, tt := range []struct {
		name           string
		expectedOutput interface{}
		mock           func(vmClient *mock_compute.MockVirtualMachinesClient)
		expectedLogs   []map[string]types.GomegaMatcher
	}{
		{
			name: "failure to fetch VMs",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return(nil, errors.New("vm explod"))
			},
			expectedLogs: []map[string]types.GomegaMatcher{},
			expectedOutput: []interface{}{
				"vm listing error: vm explod",
			},
		},
		{
			name: "no VMs returned",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{}, nil)
			},
			expectedLogs: []map[string]types.GomegaMatcher{},
			expectedOutput: []interface{}{
				"no VMs found",
			},
		},
		{
			name: "failure to get VM serial console",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{
					{
						Name:     to.StringPtr("somename"),
						Location: to.StringPtr("eastus"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								BootDiagnostics: &mgmtcompute.BootDiagnosticsInstanceView{
									SerialConsoleLogBlobURI: to.StringPtr("bogusurl"),
								},
							},
						},
					},
				}, nil)

				vmClient.EXPECT().GetSerialConsoleForVM(
					gomock.Any(), "resourceGroupCluster", "somename", gomock.Any(),
				).Times(1).Return(errors.New("explod"))
			},
			expectedLogs: []map[string]types.GomegaMatcher{},
			expectedOutput: []interface{}{
				`vm somename: {"location":"eastus","properties":{}}`,
				"vm boot diagnostics retrieval error for somename: explod",
			},
		},
		{
			name: "success",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{
					{
						Name:                     to.StringPtr("somename"),
						Location:                 to.StringPtr("eastus"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{},
					},
				}, nil)

				iothing := bytes.NewBufferString("hello\nthere :)")
				vmClient.EXPECT().GetSerialConsoleForVM(
					gomock.Any(), "resourceGroupCluster", "somename", gomock.Any(),
				).Times(1).DoAndReturn(func(ctx context.Context,
					rg string, vmName string, target io.Writer) error {
					_, err := io.Copy(target, iothing)
					return err
				})
			},
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`hello`),
					"failedRoleInstance": gomega.Equal("somename"),
				},
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`there :)`),
					"failedRoleInstance": gomega.Equal("somename"),
				},
			},
			expectedOutput: []interface{}{
				`vm somename: {"location":"eastus","properties":{}}`,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			hook, entry := testlog.New()

			controller := gomock.NewController(t)
			defer controller.Finish()

			vmClient := mock_compute.NewMockVirtualMachinesClient(controller)

			tt.mock(vmClient)

			d := &manager{
				log:             entry,
				doc:             oc,
				virtualMachines: vmClient,
			}

			out, err := d.LogVMSerialConsole(ctx)
			if err != nil {
				t.Errorf("returned %s, should never return an error", err)
			}

			err = testlog.AssertLoggingOutput(hook, tt.expectedLogs)
			if err != nil {
				t.Error(err)
			}

			for _, e := range deep.Equal(out, tt.expectedOutput) {
				t.Error(e)
			}
		})
	}
}
