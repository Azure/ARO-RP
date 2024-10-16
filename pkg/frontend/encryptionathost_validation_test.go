package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armfeatures"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armfeatures "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armfeatures"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestValidateEncryptionAtHost(t *testing.T) {
	getClusterWithNodeProfiles := func(MasterProfile api.MasterProfile, WorkerProfiles []api.WorkerProfile) *api.OpenShiftCluster {
		return &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				MasterProfile:  MasterProfile,
				WorkerProfiles: WorkerProfiles,
			},
		}
	}

	getSubscriptionWithFeatureState := func(state armfeatures.SubscriptionFeatureRegistrationState) *armfeatures.SubscriptionFeatureRegistrationsClientGetResponse {
		return &armfeatures.SubscriptionFeatureRegistrationsClientGetResponse{
			SubscriptionFeatureRegistration: armfeatures.SubscriptionFeatureRegistration{
				Properties: &armfeatures.SubscriptionFeatureRegistrationProperties{
					State: &state,
				},
			},
		}
	}

	for _, tt := range []struct {
		name            string
		oc              *api.OpenShiftCluster
		mockResponse    *armfeatures.SubscriptionFeatureRegistrationsClientGetResponse
		mockErr         error
		wantErr         string
		wantNumArmCalls int
	}{
		{
			name:            "valid: cluster encryption at host disabled and subscription feature isn't registered",
			oc:              getClusterWithNodeProfiles(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostDisabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostDisabled}}),
			mockResponse:    getSubscriptionWithFeatureState(armfeatures.SubscriptionFeatureRegistrationStateNotRegistered),
			wantNumArmCalls: 0,
		},
		{
			name:            "valid: cluster encryption at host disabled and subscription feature is registered",
			oc:              getClusterWithNodeProfiles(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostDisabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostDisabled}}),
			mockResponse:    getSubscriptionWithFeatureState(armfeatures.SubscriptionFeatureRegistrationStateRegistered),
			wantNumArmCalls: 0,
		},
		{
			name:            "valid: cluster encryption at host enabled and subscription feature is registered",
			oc:              getClusterWithNodeProfiles(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostEnabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostEnabled}}),
			mockResponse:    getSubscriptionWithFeatureState(armfeatures.SubscriptionFeatureRegistrationStateRegistered),
			wantNumArmCalls: 1,
		},
		{
			name:            "invalid: cluster master and worker encryption at host enabled and subscription feature isn't registered",
			oc:              getClusterWithNodeProfiles(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostEnabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostEnabled}}),
			mockResponse:    getSubscriptionWithFeatureState(armfeatures.SubscriptionFeatureRegistrationStateNotRegistered),
			wantErr:         "400: InvalidParameter: armfeatures.SubscriptionFeatureRegistrationProperties: Microsoft.Compute/EncryptionAtHost feature is not enabled for this subscription. Register the feature using 'az feature register --namespace Microsoft.Compute --name EncryptionAtHost'",
			wantNumArmCalls: 1,
		},
		{
			name:            "invalid: cluster master encryption at host enabled and subscription feature isn't registered",
			oc:              getClusterWithNodeProfiles(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostEnabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostDisabled}}),
			mockResponse:    getSubscriptionWithFeatureState(armfeatures.SubscriptionFeatureRegistrationStateNotRegistered),
			wantErr:         "400: InvalidParameter: armfeatures.SubscriptionFeatureRegistrationProperties: Microsoft.Compute/EncryptionAtHost feature is not enabled for this subscription. Register the feature using 'az feature register --namespace Microsoft.Compute --name EncryptionAtHost'",
			wantNumArmCalls: 1,
		},
		{
			name:            "invalid: cluster worker encryption at host enabled and subscription feature isn't registered",
			oc:              getClusterWithNodeProfiles(api.MasterProfile{EncryptionAtHost: api.EncryptionAtHostDisabled}, []api.WorkerProfile{{EncryptionAtHost: api.EncryptionAtHostEnabled}}),
			mockResponse:    getSubscriptionWithFeatureState(armfeatures.SubscriptionFeatureRegistrationStateNotRegistered),
			wantErr:         "400: InvalidParameter: armfeatures.SubscriptionFeatureRegistrationProperties: Microsoft.Compute/EncryptionAtHost feature is not enabled for this subscription. Register the feature using 'az feature register --namespace Microsoft.Compute --name EncryptionAtHost'",
			wantNumArmCalls: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			subFeatureRegistrationsClient := mock_armfeatures.NewMockSubscriptionFeatureRegistrationsClient(controller)
			subFeatureRegistrationsClient.EXPECT().Get(gomock.Any(), "Microsoft.Compute", "EncryptionAtHost", gomock.Any()).Return(*tt.mockResponse, tt.mockErr).Times(tt.wantNumArmCalls)

			err := validateEncryptionAtHostGivenClient(context.Background(), subFeatureRegistrationsClient, tt.oc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
