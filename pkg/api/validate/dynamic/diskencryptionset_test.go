package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_authorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/test/util/matcher"
)

func TestValidateDiskEncryptionSets(t *testing.T) {
	fakeDesID1 := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/fakeRG/providers/Microsoft.Compute/diskEncryptionSets/fakeDES1"
	fakeDesR1, err := azure.ParseResourceID(fakeDesID1)
	if err != nil {
		t.Fatal(err)
	}
	fakeDesID2 := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/fakeRG/providers/Microsoft.Compute/diskEncryptionSets/fakeDES2"
	fakeDesR2, err := azure.ParseResourceID(fakeDesID2)
	if err != nil {
		t.Fatal(err)
	}

	for _, authorizerType := range []AuthorizerType{AuthorizerClusterServicePrincipal, AuthorizerFirstParty} {
		wantErrCode := api.CloudErrorCodeInvalidResourceProviderPermissions
		if authorizerType == AuthorizerClusterServicePrincipal {
			wantErrCode = api.CloudErrorCodeInvalidServicePrincipalPermissions
		}

		t.Run(string(authorizerType), func(t *testing.T) {
			for _, tt := range []struct {
				name    string
				oc      *api.OpenShiftCluster
				mocks   func(permissions *mock_authorization.MockPermissionsClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, cancel context.CancelFunc)
				wantErr string
			}{
				{
					name: "no disk encryption set provided",
					oc:   &api.OpenShiftCluster{},
				},
				{
					name: "valid disk encryption set",
					oc: &api.OpenShiftCluster{
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								DiskEncryptionSetID: fakeDesID1,
							},
							WorkerProfiles: []api.WorkerProfile{{
								DiskEncryptionSetID: fakeDesID1,
							}},
						},
					},
					mocks: func(permissions *mock_authorization.MockPermissionsClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, cancel context.CancelFunc) {
						permissions.EXPECT().
							ListForResource(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.Provider, "", fakeDesR1.ResourceType, fakeDesR1.ResourceName).
							Return([]mgmtauthorization.Permission{{
								Actions:    &[]string{"Microsoft.Compute/diskEncryptionSets/read"},
								NotActions: &[]string{},
							}}, nil)
						diskEncryptionSets.EXPECT().
							Get(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.ResourceName).
							Return(mgmtcompute.DiskEncryptionSet{Location: to.StringPtr("eastus")}, nil)
					},
				},
				{
					name: "valid permissions multiple disk encryption sets",
					oc: &api.OpenShiftCluster{
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								DiskEncryptionSetID: fakeDesID1,
							},
							WorkerProfiles: []api.WorkerProfile{{
								DiskEncryptionSetID: fakeDesID2,
							}},
						},
					},
					mocks: func(permissions *mock_authorization.MockPermissionsClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, cancel context.CancelFunc) {
						permissions.EXPECT().
							ListForResource(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.Provider, "", fakeDesR1.ResourceType, fakeDesR1.ResourceName).
							Return([]mgmtauthorization.Permission{{
								Actions:    &[]string{"Microsoft.Compute/diskEncryptionSets/read"},
								NotActions: &[]string{},
							}}, nil)
						permissions.EXPECT().
							ListForResource(gomock.Any(), fakeDesR2.ResourceGroup, fakeDesR2.Provider, "", fakeDesR2.ResourceType, fakeDesR2.ResourceName).
							Return([]mgmtauthorization.Permission{{
								Actions:    &[]string{"Microsoft.Compute/diskEncryptionSets/read"},
								NotActions: &[]string{},
							}}, nil)
						diskEncryptionSets.EXPECT().
							Get(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.ResourceName).
							Return(mgmtcompute.DiskEncryptionSet{Location: to.StringPtr("eastus")}, nil)
						diskEncryptionSets.EXPECT().
							Get(gomock.Any(), fakeDesR2.ResourceGroup, fakeDesR2.ResourceName).
							Return(mgmtcompute.DiskEncryptionSet{Location: to.StringPtr("eastus")}, nil)
					},
				},
				{
					name: "disk encryption set not found",
					oc: &api.OpenShiftCluster{
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								DiskEncryptionSetID: fakeDesID1,
							},
							WorkerProfiles: []api.WorkerProfile{{
								DiskEncryptionSetID: fakeDesID1,
							}},
						},
					},
					mocks: func(permissions *mock_authorization.MockPermissionsClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, cancel context.CancelFunc) {
						permissions.EXPECT().
							ListForResource(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.Provider, "", fakeDesR1.ResourceType, fakeDesR1.ResourceName).
							Return([]mgmtauthorization.Permission{{
								Actions:    &[]string{"Microsoft.Compute/diskEncryptionSets/read"},
								NotActions: &[]string{},
							}}, nil)
						diskEncryptionSets.EXPECT().
							Get(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.ResourceName).
							Return(mgmtcompute.DiskEncryptionSet{}, autorest.DetailedError{StatusCode: http.StatusNotFound})
					},
					wantErr: fmt.Sprintf("400: InvalidLinkedDiskEncryptionSet: properties.masterProfile.diskEncryptionSetId: The disk encryption set '%s' could not be found.", fakeDesID1),
				},
				{
					name: "disk encryption set unhandled permissions error",
					oc: &api.OpenShiftCluster{
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								DiskEncryptionSetID: fakeDesID1,
							},
							WorkerProfiles: []api.WorkerProfile{{
								DiskEncryptionSetID: fakeDesID1,
							}},
						},
					},
					mocks: func(permissions *mock_authorization.MockPermissionsClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, cancel context.CancelFunc) {
						permissions.EXPECT().
							ListForResource(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.Provider, "", fakeDesR1.ResourceType, fakeDesR1.ResourceName).
							Return(nil, errors.New("fakeerr"))
					},
					wantErr: "fakeerr",
				},
				{
					name: "disk encryption set unhandled get error",
					oc: &api.OpenShiftCluster{
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								DiskEncryptionSetID: fakeDesID1,
							},
							WorkerProfiles: []api.WorkerProfile{{
								DiskEncryptionSetID: fakeDesID1,
							}},
						},
					},
					mocks: func(permissions *mock_authorization.MockPermissionsClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, cancel context.CancelFunc) {
						permissions.EXPECT().
							ListForResource(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.Provider, "", fakeDesR1.ResourceType, fakeDesR1.ResourceName).
							Return([]mgmtauthorization.Permission{{
								Actions:    &[]string{"Microsoft.Compute/diskEncryptionSets/read"},
								NotActions: &[]string{},
							}}, nil)
						diskEncryptionSets.EXPECT().
							Get(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.ResourceName).
							Return(mgmtcompute.DiskEncryptionSet{}, errors.New("fakeerr"))
					},
					wantErr: "fakeerr",
				},
				{
					name: "invalid permissions",
					oc: &api.OpenShiftCluster{
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								DiskEncryptionSetID: fakeDesID1,
							},
							WorkerProfiles: []api.WorkerProfile{{
								DiskEncryptionSetID: fakeDesID2,
							}},
						},
					},
					mocks: func(permissions *mock_authorization.MockPermissionsClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, cancel context.CancelFunc) {
						permissions.EXPECT().
							ListForResource(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.Provider, "", fakeDesR1.ResourceType, fakeDesR1.ResourceName).
							Do(func(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) {
								cancel()
							})
					},
					wantErr: fmt.Sprintf("400: %s: properties.masterProfile.diskEncryptionSetId: The %s service principal does not have Reader permission on disk encryption set '%s'.", wantErrCode, authorizerType, fakeDesID1),
				},
				{
					name: "one of the disk encryption set permissions not found",
					oc: &api.OpenShiftCluster{
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								DiskEncryptionSetID: fakeDesID1,
							},
							WorkerProfiles: []api.WorkerProfile{{
								DiskEncryptionSetID: fakeDesID2,
							}},
						},
					},
					mocks: func(permissions *mock_authorization.MockPermissionsClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, cancel context.CancelFunc) {
						permissions.EXPECT().
							ListForResource(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.Provider, "", fakeDesR1.ResourceType, fakeDesR1.ResourceName).
							Return([]mgmtauthorization.Permission{{
								Actions:    &[]string{"Microsoft.Compute/diskEncryptionSets/read"},
								NotActions: &[]string{},
							}}, nil)
						permissions.EXPECT().
							ListForResource(gomock.Any(), fakeDesR2.ResourceGroup, fakeDesR2.Provider, "", fakeDesR2.ResourceType, fakeDesR2.ResourceName).
							Return(nil, autorest.DetailedError{StatusCode: http.StatusNotFound})
						diskEncryptionSets.EXPECT().
							Get(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.ResourceName).
							Return(mgmtcompute.DiskEncryptionSet{Location: to.StringPtr("eastus")}, nil)
					},
					wantErr: fmt.Sprintf("400: InvalidLinkedDiskEncryptionSet: properties.workerProfiles[0].diskEncryptionSetId: The disk encryption set '%s' could not be found.", fakeDesID2),
				},
				{
					name: "disk encryption set invalid location",
					oc: &api.OpenShiftCluster{
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								DiskEncryptionSetID: fakeDesID1,
							},
							WorkerProfiles: []api.WorkerProfile{{
								DiskEncryptionSetID: fakeDesID1,
							}},
						},
					},
					mocks: func(permissions *mock_authorization.MockPermissionsClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, cancel context.CancelFunc) {
						permissions.EXPECT().
							ListForResource(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.Provider, "", fakeDesR1.ResourceType, fakeDesR1.ResourceName).
							Return([]mgmtauthorization.Permission{{
								Actions:    &[]string{"Microsoft.Compute/diskEncryptionSets/read"},
								NotActions: &[]string{},
							}}, nil)
						diskEncryptionSets.EXPECT().
							Get(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.ResourceName).
							Return(mgmtcompute.DiskEncryptionSet{Location: to.StringPtr("westeurope")}, nil)
					},
					wantErr: "400: InvalidLinkedDiskEncryptionSet: : The disk encryption set location 'westeurope' must match the cluster location 'eastus'.",
				},
			} {
				t.Run(tt.name, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					controller := gomock.NewController(t)
					defer controller.Finish()

					permissionsClient := mock_authorization.NewMockPermissionsClient(controller)
					diskEncryptionSetsClient := mock_compute.NewMockDiskEncryptionSetsClient(controller)

					if tt.mocks != nil {
						tt.mocks(permissionsClient, diskEncryptionSetsClient, cancel)
					}

					dv := &dynamic{
						authorizerType:     authorizerType,
						log:                logrus.NewEntry(logrus.StandardLogger()),
						permissions:        permissionsClient,
						diskEncryptionSets: diskEncryptionSetsClient,
					}

					err := dv.ValidateDiskEncryptionSets(ctx, tt.oc)
					matcher.AssertErrHasWantMsg(t, err, tt.wantErr)
				})
			}
		})
	}
}
