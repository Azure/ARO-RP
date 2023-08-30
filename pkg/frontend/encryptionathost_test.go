package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armfeatures"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestValidateEncryptionAtHost(t *testing.T) {
	EncryptionAtHostEnabledOrDisabled := func(MasterProfile api.MasterProfile, WorkerProfiles []api.WorkerProfile) *api.OpenShiftCluster {
		return &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				MasterProfile:  MasterProfile,
				WorkerProfiles: WorkerProfiles,
			},
		}
	}

	EncryptionAtHostFeatureEnabledOrDisabled := func(state armfeatures.SubscriptionFeatureRegistrationState) *armfeatures.SubscriptionFeatureRegistrationsClientGetResponse {
		return &armfeatures.SubscriptionFeatureRegistrationsClientGetResponse{
			SubscriptionFeatureRegistration: armfeatures.SubscriptionFeatureRegistration{
				Properties: &armfeatures.SubscriptionFeatureRegistrationProperties{
					State: &state,
				},
			},
		}
	}

	for _, tt := range []struct {
		name         string
		oc           *api.OpenShiftCluster
		mockResponse *armfeatures.SubscriptionFeatureRegistrationsClientGetResponse
		mockErr      error
		wantErr      string
	}{
		{
			name:         "encryption at host disabled - feature isn't registered",
			oc:           EncryptionAtHostEnabledOrDisabled(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostDisabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostDisabled}}),
			mockResponse: EncryptionAtHostFeatureEnabledOrDisabled(armfeatures.SubscriptionFeatureRegistrationStateNotRegistered),
		},
		{
			name:         "encryption at host disabled - feature is registered",
			oc:           EncryptionAtHostEnabledOrDisabled(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostDisabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostDisabled}}),
			mockResponse: EncryptionAtHostFeatureEnabledOrDisabled(armfeatures.SubscriptionFeatureRegistrationStateRegistered),
		},
		{
			name:         "encryption at host enabled - feature is registered",
			oc:           EncryptionAtHostEnabledOrDisabled(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostEnabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostEnabled}}),
			mockResponse: EncryptionAtHostFeatureEnabledOrDisabled(armfeatures.SubscriptionFeatureRegistrationStateRegistered),
		},
		{
			name:         "encryption at host enabled - feature isn't registered",
			oc:           EncryptionAtHostEnabledOrDisabled(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostEnabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostEnabled}}),
			mockResponse: EncryptionAtHostFeatureEnabledOrDisabled(armfeatures.SubscriptionFeatureRegistrationStateNotRegistered),
			wantErr:      "400: InvalidParameter: armfeatures.SubscriptionFeatureRegistrationProperties: Microsoft.Compute/EncryptionAtHost feature is not enabled for this subscription. Register the feature using 'az feature register --namespace Microsoft.Compute --name EncryptionAtHost'",
		},
		{
			name:         "MasterProfile encryption at host enabled - feature isn't registered",
			oc:           EncryptionAtHostEnabledOrDisabled(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostEnabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostDisabled}}),
			mockResponse: EncryptionAtHostFeatureEnabledOrDisabled(armfeatures.SubscriptionFeatureRegistrationStateNotRegistered),
			wantErr:      "400: InvalidParameter: armfeatures.SubscriptionFeatureRegistrationProperties: Microsoft.Compute/EncryptionAtHost feature is not enabled for this subscription. Register the feature using 'az feature register --namespace Microsoft.Compute --name EncryptionAtHost'",
		},
		{
			name:         "WorkerProfile encryption at host enabled - feature isn't registered",
			oc:           EncryptionAtHostEnabledOrDisabled(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostDisabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostEnabled}}),
			mockResponse: EncryptionAtHostFeatureEnabledOrDisabled(armfeatures.SubscriptionFeatureRegistrationStateNotRegistered),
			wantErr:      "400: InvalidParameter: armfeatures.SubscriptionFeatureRegistrationProperties: Microsoft.Compute/EncryptionAtHost feature is not enabled for this subscription. Register the feature using 'az feature register --namespace Microsoft.Compute --name EncryptionAtHost'",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			subFeatureRegistrationsClient := mock_features.NewMockSubscriptionFeatureRegistrationsClient(controller)
			subFeatureRegistrationsClient.EXPECT().Get(gomock.Any(), "Microsoft.Compute", "EncryptionAtHost", gomock.Any()).Return(*tt.mockResponse, tt.mockErr).AnyTimes()

			err := validateEncryptionAtHost(context.Background(), subFeatureRegistrationsClient, tt.oc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
