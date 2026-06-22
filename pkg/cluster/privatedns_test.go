package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"

	mock_privatedns "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/privatedns"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var (
	resourceGroupName = "testGroup"
	subscriptionID    = "0000000-0000-0000-0000-000000000000"
	resourceGroupID   = "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroupName
	resourceID        = resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/" + vnetName
)

func TestDeletePrivateDNSVNetLinks(t *testing.T) {
	type testCase struct {
		name                string
		resourceID          string
		wantErr             string
		ensureMocksBehavior func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient)
	}
	testcases := []testCase{
		{
			name:       "propagates invalid resource id error",
			resourceID: "invalid_resourceId",
			wantErr:    "parsing failed for invalid_resourceId. Invalid resource Id format",
			ensureMocksBehavior: func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient) {
				vNetLinksClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("parsing failed for invalid_resourceId. Invalid resource Id format"))
			},
		},
		{
			name:    "propagates error from vNetLinksClient.List",
			wantErr: "some_error",
			ensureMocksBehavior: func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient) {
				vNetLinksClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("some_error"))
			},
		},
		{
			name:    "ppropagates error from vNetLinksClient.DeleteAndWait",
			wantErr: "some_error",
			ensureMocksBehavior: func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient) {
				name := "name"
				listResult := []mgmtprivatedns.VirtualNetworkLink{{Name: &name}}
				vNetLinksClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(listResult, nil)
				vNetLinksClient.EXPECT().DeleteAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("some_error"))
			},
		},
		{
			name: "returns nil when no errors found",
			ensureMocksBehavior: func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient) {
				name := "name"
				listResult := []mgmtprivatedns.VirtualNetworkLink{{Name: &name}}
				vNetLinksClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(listResult, nil)
				vNetLinksClient.EXPECT().DeleteAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			vNetLinksClient := mock_privatedns.NewMockVirtualNetworkLinksClient(controller)

			if tc.ensureMocksBehavior != nil {
				tc.ensureMocksBehavior(vNetLinksClient)
			}

			err := DeletePrivateDNSVNetLinks(context.Background(), vNetLinksClient, resourceID)
			utilerror.AssertErrorMessage(t, err, tc.wantErr)
		})
	}
}
