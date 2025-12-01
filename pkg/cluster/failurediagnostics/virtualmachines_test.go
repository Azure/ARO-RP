package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
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
		expectedLogs   []testlog.ExpectedLogEntry
	}{
		{
			name: "failure to fetch VMs",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return(nil, errors.New("vm explod"))
			},
			expectedLogs: []testlog.ExpectedLogEntry{},
			expectedOutput: []interface{}{
				"vm listing error: vm explod",
			},
		},
		{
			name: "no VMs returned",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{}, nil)
			},
			expectedLogs: []testlog.ExpectedLogEntry{},
			expectedOutput: []interface{}{
				"no VMs found",
			},
		},
		{
			name: "failure to get VM serial console",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{
					{
						Name:     pointerutils.ToPtr("somename"),
						Location: pointerutils.ToPtr("eastus"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								BootDiagnostics: &mgmtcompute.BootDiagnosticsInstanceView{
									SerialConsoleLogBlobURI: pointerutils.ToPtr("bogusurl"),
								},
							},
						},
					},
				}, nil)

				vmClient.EXPECT().GetSerialConsoleForVM(
					gomock.Any(), "resourceGroupCluster", "somename", gomock.Any(),
				).Times(1).Return(errors.New("explod"))
			},
			expectedLogs: []testlog.ExpectedLogEntry{},
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
						Name:                     pointerutils.ToPtr("somename"),
						Location:                 pointerutils.ToPtr("eastus"),
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
			expectedLogs: []testlog.ExpectedLogEntry{
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
		{
			name: "success (pure duplicates)",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{
					{
						Name:                     pointerutils.ToPtr("somename"),
						Location:                 pointerutils.ToPtr("eastus"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{},
					},
				}, nil)

				iothing := bytes.NewBufferString("hello\nthere :)\nthere :)")
				vmClient.EXPECT().GetSerialConsoleForVM(
					gomock.Any(), "resourceGroupCluster", "somename", gomock.Any(),
				).Times(1).DoAndReturn(func(ctx context.Context,
					rg string, vmName string, target io.Writer) error {
					_, err := io.Copy(target, iothing)
					return err
				})
			},
			expectedLogs: []testlog.ExpectedLogEntry{
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
		{
			name: "success (empty blob)",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{
					{
						Name:                     pointerutils.ToPtr("somename"),
						Location:                 pointerutils.ToPtr("eastus"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{},
					},
				}, nil)

				iothing := bytes.NewBufferString("")
				vmClient.EXPECT().GetSerialConsoleForVM(
					gomock.Any(), "resourceGroupCluster", "somename", gomock.Any(),
				).Times(1).DoAndReturn(func(ctx context.Context,
					rg string, vmName string, target io.Writer) error {
					_, err := io.Copy(target, iothing)
					return err
				})
			},
			expectedLogs: []testlog.ExpectedLogEntry{},
			expectedOutput: []interface{}{
				`vm somename: {"location":"eastus","properties":{}}`,
			},
		},
		{
			name: "logs limited by kb",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{
					{
						Name:                     pointerutils.ToPtr("somename"),
						Location:                 pointerutils.ToPtr("eastus"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{},
					},
				}, nil)

				iothing := bytes.NewBufferString("")
				for i := 0; i < 11; i++ {
					fmt.Fprintf(iothing, "%d", i)
					for x := 0; x < 98; x++ {
						iothing.WriteByte('a')
					}
					iothing.WriteByte('\n')
				}
				vmClient.EXPECT().GetSerialConsoleForVM(
					gomock.Any(), "resourceGroupCluster", "somename", gomock.Any(),
				).Times(1).DoAndReturn(func(ctx context.Context,
					rg string, vmName string, target io.Writer) error {
					_, err := io.Copy(target, iothing)
					return err
				})
			},
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`),
					"failedRoleInstance": gomega.Equal("somename"),
				},
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`2aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`),
					"failedRoleInstance": gomega.Equal("somename"),
				},
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`3aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`),
					"failedRoleInstance": gomega.Equal("somename"),
				},
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`4aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`),
					"failedRoleInstance": gomega.Equal("somename"),
				},
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`5aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`),
					"failedRoleInstance": gomega.Equal("somename"),
				},
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`6aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`),
					"failedRoleInstance": gomega.Equal("somename"),
				},
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`7aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`),
					"failedRoleInstance": gomega.Equal("somename"),
				},
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`8aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`),
					"failedRoleInstance": gomega.Equal("somename"),
				},
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`9aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`),
					"failedRoleInstance": gomega.Equal("somename"),
				},
				{
					"level":              gomega.Equal(logrus.InfoLevel),
					"msg":                gomega.Equal(`10aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`),
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

			out, err := d.logVMSerialConsole(ctx, 1)
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
