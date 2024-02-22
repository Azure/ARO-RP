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
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_storage "github.com/Azure/ARO-RP/pkg/util/mocks/storage"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestVirtualMachines(t *testing.T) {
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
		mock           func(vmClient *mock_compute.MockVirtualMachinesClient, stor *mock_storage.MockManager, blob *mock_storage.MockBlobStorageClient)
		expectedLogs   []map[string]types.GomegaMatcher
	}{
		{
			name: "failure to fetch VMs",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient, stor *mock_storage.MockManager, blob *mock_storage.MockBlobStorageClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return(nil, errors.New("vm explod"))
			},
			expectedLogs: []map[string]types.GomegaMatcher{},
			expectedOutput: []interface{}{
				"vm listing error: vm explod",
			},
		},
		{
			name: "no VMs returned",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient, stor *mock_storage.MockManager, blob *mock_storage.MockBlobStorageClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{}, nil)
			},
			expectedLogs: []map[string]types.GomegaMatcher{},
			expectedOutput: []interface{}{
				"no vms found",
			},
		},
		{
			name: "no console URIs returned",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient, stor *mock_storage.MockManager, blob *mock_storage.MockBlobStorageClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{
					{
						Name:     to.StringPtr("somename"),
						Location: to.StringPtr("eastus"),
					},
				}, nil)
			},
			expectedLogs: []map[string]types.GomegaMatcher{},
			expectedOutput: []interface{}{
				`vm somename: {"location":"eastus"}`,
				"no usable console URIs found",
			},
		},
		{
			name: "failure to fetch blob client",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient, stor *mock_storage.MockManager, blob *mock_storage.MockBlobStorageClient) {
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

				stor.EXPECT().BlobService(gomock.Any(), "resourceGroupCluster", "clusterPrefixHere", mgmtstorage.R, mgmtstorage.SignedResourceTypesO).
					Times(1).Return(nil, errors.New("explod"))
			},
			expectedLogs: []map[string]types.GomegaMatcher{},
			expectedOutput: []interface{}{
				`vm somename: {"location":"eastus","properties":{}}`,
				"blob storage error: explod",
			},
		},
		{
			name: "failed blob client get",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient, stor *mock_storage.MockManager, blob *mock_storage.MockBlobStorageClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{
					{
						Name:     to.StringPtr("somename"),
						Location: to.StringPtr("eastus"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								BootDiagnostics: &mgmtcompute.BootDiagnosticsInstanceView{
									SerialConsoleLogBlobURI: to.StringPtr("bogusurl/boguscontainer/bogusblob"),
								},
							},
						},
					},
				}, nil)

				stor.EXPECT().BlobService(gomock.Any(), "resourceGroupCluster", "clusterPrefixHere", mgmtstorage.R, mgmtstorage.SignedResourceTypesO).
					Times(1).Return(blob, nil)

				blob.EXPECT().Get("bogusurl/boguscontainer/bogusblob").Times(1).Return(nil, errors.New("can't read"))
			},
			expectedLogs: []map[string]types.GomegaMatcher{},
			expectedOutput: []interface{}{
				`vm somename: {"location":"eastus","properties":{}}`,
				"blob storage get error on somename: can't read",
			},
		},
		{
			name: "failed blob decoding",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient, stor *mock_storage.MockManager, blob *mock_storage.MockBlobStorageClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{
					{
						Name:     to.StringPtr("somename"),
						Location: to.StringPtr("eastus"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								BootDiagnostics: &mgmtcompute.BootDiagnosticsInstanceView{
									SerialConsoleLogBlobURI: to.StringPtr("bogusurl/boguscontainer/bogusblob"),
								},
							},
						},
					},
				}, nil)

				stor.EXPECT().BlobService(gomock.Any(), "resourceGroupCluster", "clusterPrefixHere", mgmtstorage.R, mgmtstorage.SignedResourceTypesO).
					Times(1).Return(blob, nil)

				out := bytes.NewBufferString("aGVsbG8KdGhlcmUgOikKZ")

				blob.EXPECT().Get("bogusurl/boguscontainer/bogusblob").Times(1).Return(io.NopCloser(out), nil)
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
				`blob storage scan on somename: unexpected EOF`,
			},
		},
		{
			name: "success",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient, stor *mock_storage.MockManager, blob *mock_storage.MockBlobStorageClient) {
				vmClient.EXPECT().List(gomock.Any(), "resourceGroupCluster").Return([]mgmtcompute.VirtualMachine{
					{
						Name:     to.StringPtr("somename"),
						Location: to.StringPtr("eastus"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								BootDiagnostics: &mgmtcompute.BootDiagnosticsInstanceView{
									SerialConsoleLogBlobURI: to.StringPtr("bogusurl/boguscontainer/bogusblob"),
								},
							},
						},
					},
				}, nil)

				stor.EXPECT().BlobService(gomock.Any(), "resourceGroupCluster", "clusterPrefixHere", mgmtstorage.R, mgmtstorage.SignedResourceTypesO).
					Times(1).Return(blob, nil)

				out := bytes.NewBufferString("aGVsbG8KdGhlcmUgOikK")

				blob.EXPECT().Get("bogusurl/boguscontainer/bogusblob").Times(1).Return(io.NopCloser(out), nil)
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
			storageClient := mock_storage.NewMockManager(controller)
			blobClient := mock_storage.NewMockBlobStorageClient(controller)

			tt.mock(vmClient, storageClient, blobClient)

			d := &manager{
				log:             entry,
				doc:             oc,
				virtualMachines: vmClient,
				storage:         storageClient,
			}

			out, err := d.LogAzureInformation(ctx)
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
