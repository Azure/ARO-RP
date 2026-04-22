package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_azsecrets "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azsecrets"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_msidataplane "github.com/Azure/ARO-RP/pkg/util/mocks/msidataplane"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEnsureClusterMsiCertificate(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time {
		return time.Date(2025, time.September, 29, 16, 0, 0, 0, time.UTC)
	}

	mockGuid := "00000000-0000-0000-0000-000000000000"

	for _, tt := range []struct {
		name                 string
		usesWorkloadIdentity bool
		clusterResourceID    string
		mockSetup            func(*mock_env.MockInterface, *mock_azsecrets.MockClient, *mock_msidataplane.MockClientFactory)
		wantSkip             bool
		wantErr              string
	}{
		{
			name:                 "skip when cluster doesn't use workload identity",
			usesWorkloadIdentity: false,
			wantSkip:             true,
		},
		{
			name:                 "fail when MSI credential creation fails",
			usesWorkloadIdentity: true,
			mockSetup: func(env *mock_env.MockInterface, _ *mock_azsecrets.MockClient, _ *mock_msidataplane.MockClientFactory) {
				env.EXPECT().Environment().Return(&azureclient.PublicCloud)
				env.EXPECT().ClusterMsiKeyVaultName().Return("test-msi-kv")
				env.EXPECT().NewMSITokenCredential().Return(nil, errors.New("credential creation failed"))
			},
			wantErr: "TerminalError: failed to create MSI credential",
		},
		{
			name:                 "fail when MSI dataplane client options fail",
			usesWorkloadIdentity: true,
			mockSetup: func(mockEnv *mock_env.MockInterface, mockKV *mock_azsecrets.MockClient, _ *mock_msidataplane.MockClientFactory) {
				mockEnv.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				mockEnv.EXPECT().ClusterMsiKeyVaultName().Return("test-msi-kv")
				mockEnv.EXPECT().NewMSITokenCredential().Return(&fakeTokenCredential{}, nil)
				mockEnv.EXPECT().FeatureIsSet(env.FeatureUseMockMsiRp).Return(false)
				mockEnv.EXPECT().MsiDataplaneClientOptions(gomock.Any()).Return(nil, errors.New("dataplane options failed"))
			},
			wantErr: "TerminalError: failed to get MSI dataplane client options",
		},
		{
			name:                 "fail when FP credential creation fails",
			usesWorkloadIdentity: true,
			mockSetup: func(mockEnv *mock_env.MockInterface, mockKV *mock_azsecrets.MockClient, _ *mock_msidataplane.MockClientFactory) {
				mockEnv.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				mockEnv.EXPECT().ClusterMsiKeyVaultName().Return("test-msi-kv")
				mockEnv.EXPECT().NewMSITokenCredential().Return(&fakeTokenCredential{}, nil)
				mockEnv.EXPECT().FeatureIsSet(env.FeatureUseMockMsiRp).Return(false)
				mockEnv.EXPECT().MsiDataplaneClientOptions(gomock.Any()).Return(&policy.ClientOptions{}, nil)
				mockEnv.EXPECT().TenantID().Return("test-tenant-id")
				mockEnv.EXPECT().FPNewClientCertificateCredential("test-tenant-id", []string{"*"}).Return(nil, errors.New("FP credential failed"))
			},
			wantErr: "TerminalError: failed to create FP credential for MSI dataplane",
		},
		{
			name:                 "fail when ClusterMsiResourceId fails in mock mode",
			usesWorkloadIdentity: true,
			clusterResourceID:    "/invalid",
			mockSetup: func(mockEnv *mock_env.MockInterface, mockKV *mock_azsecrets.MockClient, mockFactory *mock_msidataplane.MockClientFactory) {
				mockEnv.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				mockEnv.EXPECT().ClusterMsiKeyVaultName().Return("test-msi-kv")
				mockEnv.EXPECT().NewMSITokenCredential().Return(&fakeTokenCredential{}, nil)
				mockEnv.EXPECT().FeatureIsSet(env.FeatureUseMockMsiRp).Return(true)
			},
			wantErr: "TerminalError:",
		},
		// success paths are tested in pkg/cluster/clustermsi_test.go.
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controller := gomock.NewController(t)
			defer controller.Finish()

			var mockEnv *mock_env.MockInterface
			var mockKV *mock_azsecrets.MockClient
			var mockFactory *mock_msidataplane.MockClientFactory

			if tt.mockSetup != nil {
				mockEnv = mock_env.NewMockInterface(controller)
				mockKV = mock_azsecrets.NewMockClient(controller)
				mockFactory = mock_msidataplane.NewMockClientFactory(controller)
				tt.mockSetup(mockEnv, mockKV, mockFactory)
			}

			_, log := testlog.New()

			clusterResourceID := "/subscriptions/" + mockGuid + "/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster"
			if tt.clusterResourceID != "" {
				clusterResourceID = tt.clusterResourceID
			}

			oc := api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: "/subscriptions/" + mockGuid + "/resourceGroups/test-rg",
				},
			}

			if tt.usesWorkloadIdentity {
				oc.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{}
			}

			tc := testtasks.NewFakeTestContext(
				ctx, mockEnv, log, now,
				testtasks.WithOpenShiftClusterDocument(&api.OpenShiftClusterDocument{ID: mockGuid, OpenShiftCluster: &api.OpenShiftCluster{ID: clusterResourceID, Properties: oc}}),
			)

			err := EnsureClusterMsiCertificate(tc)

			if tt.wantSkip {
				g.Expect(err).ToNot(HaveOccurred())
				return
			}

			if tt.wantErr != "" {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring(tt.wantErr))
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

type fakeTokenCredential struct{}

func (f *fakeTokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "fake-token", ExpiresOn: time.Now().Add(time.Hour)}, nil
}
