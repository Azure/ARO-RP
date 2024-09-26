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
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	mock_remotepdp "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/authz/remotepdp"
	mock_azcore "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azcore"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
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
				name                string
				oc                  *api.OpenShiftCluster
				actionInfos         []remotepdp.ActionInfo
				platformIdentities  map[string]api.PlatformWorkloadIdentity
				platformIdentityMap map[string][]string
				mocks               func(*mock_compute.MockDiskEncryptionSetsClient, *mock_remotepdp.MockRemotePDPClient, *mock_azcore.MockTokenCredential, context.CancelFunc)
				wantErr             string
				wantFPSPErr         string
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
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						pdpClient.EXPECT().
							CheckAccess(gomock.Any(), gomock.Any()).
							Return(validDiskEncryptionAuthorizationDecision, nil)
						diskEncryptionSets.EXPECT().
							Get(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.ResourceName).
							Return(mgmtcompute.DiskEncryptionSet{Location: to.StringPtr("eastus")}, nil)
					},
				},
				{
					name: "pass - MIWI Cluster",
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
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						pdpClient.EXPECT().
							CheckAccess(gomock.Any(), gomock.Any()).
							Return(validDiskEncryptionAuthorizationDecision, nil)
						diskEncryptionSets.EXPECT().
							Get(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.ResourceName).
							Return(mgmtcompute.DiskEncryptionSet{Location: to.StringPtr("eastus")}, nil)
					},
					platformIdentities: platformIdentities,
					platformIdentityMap: map[string][]string{
						"Dummy": platformIdentity1SubnetActions,
					},
				},
				{
					name: "Success - MIWI Cluster - No intersecting Subnet Actions",
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
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						diskEncryptionSets.EXPECT().
							Get(gomock.Any(), fakeDesR1.ResourceGroup, fakeDesR1.ResourceName).
							Return(mgmtcompute.DiskEncryptionSet{Location: to.StringPtr("eastus")}, nil)
					},
					platformIdentities: platformIdentities,
					platformIdentityMap: map[string][]string{
						"Dummy": platformIdentity1SubnetActionsNoIntersect,
					},
				},
				{
					name: "valid disk encryption set by enriched worker profile",
					oc: &api.OpenShiftCluster{
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								DiskEncryptionSetID: fakeDesID1,
							},
							WorkerProfilesStatus: []api.WorkerProfile{{
								DiskEncryptionSetID: fakeDesID1,
							}},
						},
					},
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						pdpClient.EXPECT().
							CheckAccess(gomock.Any(), gomock.Any()).
							Return(validDiskEncryptionAuthorizationDecision, nil)
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
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						pdpClient.EXPECT().
							CheckAccess(gomock.Any(), gomock.Any()).
							DoAndReturn(func(_ context.Context, authReq remotepdp.AuthorizationRequest) (*remotepdp.AuthorizationDecisionResponse, error) {
								cancel() // wait.PollImmediateUntil will always be invoked at least once
								switch authReq.Resource.Id {
								case fakeDesR1.String():
									return validDiskEncryptionAuthorizationDecision, nil
								case fakeDesR2.String():
									return validDiskEncryptionAuthorizationDecision, nil
								}
								return invalidDiskEncryptionAuthorizationDecisionsReadNotAllowed, nil
							},
							).AnyTimes()
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
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						pdpClient.EXPECT().
							CheckAccess(gomock.Any(), gomock.Any()).
							Return(validDiskEncryptionAuthorizationDecision, nil)
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
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						pdpClient.EXPECT().
							CheckAccess(gomock.Any(), gomock.Any()).
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
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						pdpClient.EXPECT().
							CheckAccess(gomock.Any(), gomock.Any()).
							Return(validDiskEncryptionAuthorizationDecision, nil)
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
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						pdpClient.EXPECT().
							CheckAccess(gomock.Any(), gomock.Any()).
							Do(func(arg0, arg1 interface{}) {
								cancel()
							})
					},
					wantErr: fmt.Sprintf("400: %s: properties.masterProfile.diskEncryptionSetId: The %s service principal does not have Reader permission on disk encryption set '%s'.", wantErrCode, authorizerType, fakeDesID1),
				},
				{
					name: "Fail - MIWI Cluster - permissions don't exist on diskEncryptionSet",
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
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						pdpClient.EXPECT().
							CheckAccess(gomock.Any(), gomock.Any()).
							Do(func(arg0, arg1 interface{}) {
								cancel()
							})
					},
					platformIdentities: platformIdentities,
					platformIdentityMap: map[string][]string{
						"Dummy": platformIdentity1SubnetActions,
					},
					wantErr:     fmt.Sprintf("400: %s: properties.masterProfile.diskEncryptionSetId: The Dummy platform managed identity does not have required permissions on disk encryption set '%s'.", api.CloudErrorCodeInvalidWorkloadIdentityPermissions, fakeDesID1),
					wantFPSPErr: fmt.Sprintf("400: %s: properties.masterProfile.diskEncryptionSetId: The %s service principal does not have Reader permission on disk encryption set '%s'.", wantErrCode, authorizerType, fakeDesID1),
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
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						pdpClient.EXPECT().
							CheckAccess(gomock.Any(), gomock.Any()).
							DoAndReturn(func(_ context.Context, authReq remotepdp.AuthorizationRequest) (*remotepdp.AuthorizationDecisionResponse, error) {
								cancel() // wait.PollImmediateUntil will always be invoked at least once
								switch authReq.Resource.Id {
								case fakeDesR1.String():
									return validDiskEncryptionAuthorizationDecision, nil
								case fakeDesR2.String():
									return nil, autorest.DetailedError{StatusCode: http.StatusNotFound}
								}
								return invalidDiskEncryptionAuthorizationDecisionsReadNotAllowed, nil
							},
							).AnyTimes()
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
					mocks: func(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, cancel context.CancelFunc) {
						mockTokenCredential(tokenCred)
						pdpClient.EXPECT().
							CheckAccess(gomock.Any(), gomock.Any()).
							Return(validDiskEncryptionAuthorizationDecision, nil)
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

					diskEncryptionSetsClient := mock_compute.NewMockDiskEncryptionSetsClient(controller)
					tokenCred := mock_azcore.NewMockTokenCredential(controller)
					remotePDPClient := mock_remotepdp.NewMockRemotePDPClient(controller)

					if tt.mocks != nil {
						tt.mocks(diskEncryptionSetsClient, remotePDPClient, tokenCred, cancel)
					}

					dv := &dynamic{
						appID:                      to.StringPtr("fff51942-b1f9-4119-9453-aaa922259eb7"),
						azEnv:                      &azureclient.PublicCloud,
						authorizerType:             authorizerType,
						log:                        logrus.NewEntry(logrus.StandardLogger()),
						diskEncryptionSets:         diskEncryptionSetsClient,
						pdpClient:                  remotePDPClient,
						checkAccessSubjectInfoCred: tokenCred,
					}

					if tt.platformIdentities != nil {
						dv.platformIdentities = tt.platformIdentities
						dv.platformIdentitiesActionsMap = tt.platformIdentityMap
						if authorizerType == AuthorizerClusterServicePrincipal {
							dv.authorizerType = AuthorizerWorkloadIdentity
						} else {
							tt.wantErr = tt.wantFPSPErr
						}
					}

					err := dv.ValidateDiskEncryptionSets(ctx, tt.oc)
					utilerror.AssertErrorMessage(t, err, tt.wantErr)
				})
			}
		})
	}
}

var (
	invalidDiskEncryptionAuthorizationDecisionsReadNotAllowed = &remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Compute/diskEncryptionSets/read",
				AccessDecision: remotepdp.NotAllowed,
			},
		},
	}
	validDiskEncryptionAuthorizationDecision = &remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Compute/diskEncryptionSets/read",
				AccessDecision: remotepdp.Allowed,
			},
		},
	}
)
