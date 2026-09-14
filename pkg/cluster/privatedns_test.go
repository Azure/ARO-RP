package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/util/wait"

	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	mock_privatedns "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/privatedns"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

// autorestTransientConflictError returns a retryable autorest.DetailedError matching the "Please retry later" transient conflict body.
func autorestTransientConflictError() autorest.DetailedError {
	return autorest.DetailedError{
		StatusCode: http.StatusConflict,
		Original:   errors.New("ConflictingConcurrentWriteNotAllowed: The operation was interrupted by a conflicting concurrent write on the same entity. Please retry later."),
	}
}

var (
	resourceGroupName = "testGroup"
	subscriptionID    = "0000000-0000-0000-0000-000000000000"
	resourceGroupID   = "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroupName
	resourceID        = resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/" + vnetName
)

func TestDeletePrivateDNSVNetLinks(t *testing.T) {
	// must not be called with t.Parallel(); mutates package-level arm.TransientBackoff
	log := logrus.NewEntry(logrus.StandardLogger())

	type testCase struct {
		name                string
		resourceID          string // required: every case must specify the input under test
		wantErr             string
		backoffSteps        int // 0 means use default (1 = no retries)
		ensureMocksBehavior func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient)
	}
	testcases := []testCase{
		{
			// azure.ParseResourceID fails before the client is used at all.
			name:                "propagates invalid resource id error",
			resourceID:          "invalid_resourceId",
			wantErr:             "parsing failed for invalid_resourceId. Invalid resource Id format",
			ensureMocksBehavior: nil,
		},
		{
			name:       "propagates error from vNetLinksClient.List",
			resourceID: resourceID,
			wantErr:    "some_error",
			ensureMocksBehavior: func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient) {
				vNetLinksClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("some_error"))
			},
		},
		{
			name:       "propagates non-retryable error from vNetLinksClient.DeleteAndWait",
			resourceID: resourceID,
			wantErr:    "some_error",
			ensureMocksBehavior: func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient) {
				name := "name"
				listResult := []mgmtprivatedns.VirtualNetworkLink{{Name: &name}}
				vNetLinksClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(listResult, nil)
				vNetLinksClient.EXPECT().DeleteAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("some_error"))
			},
		},
		{
			name:         "retries transient DeleteAndWait error and succeeds",
			resourceID:   resourceID,
			backoffSteps: 2, // allow one retry
			ensureMocksBehavior: func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient) {
				name := "name"
				listResult := []mgmtprivatedns.VirtualNetworkLink{{Name: &name}}
				vNetLinksClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(listResult, nil)
				gomock.InOrder(
					vNetLinksClient.EXPECT().DeleteAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(autorestTransientConflictError()),
					vNetLinksClient.EXPECT().DeleteAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil),
				)
			},
		},
		{
			name:       "returns nil when no errors found",
			resourceID: resourceID,
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
			origBackoff := arm.TransientBackoff
			steps := tc.backoffSteps
			if steps == 0 {
				steps = 1
			}
			arm.TransientBackoff = wait.Backoff{Steps: steps, Duration: time.Millisecond, Factor: 2.0}
			defer func() { arm.TransientBackoff = origBackoff }()

			controller := gomock.NewController(t)
			vNetLinksClient := mock_privatedns.NewMockVirtualNetworkLinksClient(controller)

			if tc.ensureMocksBehavior != nil {
				tc.ensureMocksBehavior(vNetLinksClient)
			}

			err := DeletePrivateDNSVNetLinks(context.Background(), log, vNetLinksClient, tc.resourceID)
			utilerror.AssertErrorMessage(t, err, tc.wantErr)
		})
	}
}
