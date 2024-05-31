package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	ocmapi "github.com/Azure/ARO-RP/pkg/util/ocm/api"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

func createSecretBytesWithAuths(authsJson string) []byte {
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte(authsJson),
		},
	}
	secretBytes, _ := json.Marshal(secret)
	return secretBytes
}

func createConfigMapBytesWithConfigYaml(configYaml string) []byte {
	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		Data: map[string]string{
			"config.yaml": configYaml,
		},
	}
	configMapBytes, _ := json.Marshal(configMap)
	return configMapBytes
}

func TestOCMOperations(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	clusterInfo := &ocmapi.ClusterInfo{
		Id:                "testClusterID",
		Name:              "testClusterName",
		ExternalID:        "testExternalID",
		DisplayName:       "Test Cluster",
		CreationTimestamp: time.Now(),
		ActivityTimestamp: time.Now(),
		OpenshiftVersion:  "4.8.0",
		Version: ocmapi.ClusterVersion{
			Id:                 "4.8.0",
			ChannelGroup:       "stable",
			AvailableUpgrades:  []string{"4.9.0", "4.10.0"},
			EndOfLifeTimestamp: time.Now().AddDate(1, 0, 0), // One year from now
		},
		NodeDrainGracePeriod: ocmapi.NodeDrainGracePeriod{
			Value: 60,
			Unit:  "minutes",
		},
		UpgradePolicies: []ocmapi.UpgradePolicy{
			{
				Id: "testPolicyID",
				UpgradePolicyStatus: ocmapi.UpgradePolicyStatus{
					State:       "scheduled",
					Description: "Upgrade scheduled",
				},
			},
			{
				Id: "testPolicyID2",
				UpgradePolicyStatus: ocmapi.UpgradePolicyStatus{
					State:       "running",
					Description: "Upgrade is running",
				},
			},
		},
	}
	clusterInfoBytes, _ := json.Marshal(clusterInfo)

	cancelUpgradeReply := &ocmapi.CancelUpgradeResponse{
		Kind:        "CancelUpgradeResponse",
		Value:       "cancelled",
		Description: "Manually cancelled by SRE",
	}
	cancelUpgradeReplyBytes, _ := json.Marshal(cancelUpgradeReply)

	ctx := context.Background()

	testCases := []struct {
		name           string
		resourceID     string
		fixture        func(*testdatabase.Fixture)
		mocks          func(*mock_adminactions.MockOCMActions, *mock_adminactions.MockKubeActions)
		endpoint       string
		httpMethod     string
		wantStatusCode int
		wantResponse   []byte
		wantError      string
	}{
		{
			name:       "Get Cluster Info",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})

				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks: func(ocmActions *mock_adminactions.MockOCMActions, kubeActions *mock_adminactions.MockKubeActions) {
				kubeActions.EXPECT().KubeGet(gomock.Any(), "Secret", "openshift-config", "pull-secret").Return(createSecretBytesWithAuths(`{"auths":{"cloud.openshift.com":{"auth":"mock-token"}}}`), nil)
				kubeActions.EXPECT().KubeGet(gomock.Any(), "ConfigMap", "openshift-managed-upgrade-operator", "managed-upgrade-operator-config").Return(createConfigMapBytesWithConfigYaml(`configManager:
  source: LOCAL
  localConfigName: managed-upgrade-config`), nil)
				ocmActions.EXPECT().GetClusterInfoWithUpgradePolicies(gomock.Any()).Return(clusterInfo, nil)
			},
			endpoint:       "/getocmclusterinfowithupgradepolicies",
			httpMethod:     http.MethodGet,
			wantStatusCode: http.StatusOK,
			wantResponse:   clusterInfoBytes,
		},
		{
			name:       "Cancel Cluster Upgrade Policy",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})

				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks: func(ocmActions *mock_adminactions.MockOCMActions, kubeActions *mock_adminactions.MockKubeActions) {
				kubeActions.EXPECT().KubeGet(gomock.Any(), "Secret", "openshift-config", "pull-secret").Return(createSecretBytesWithAuths(`{"auths":{"cloud.openshift.com":{"auth":"mock-token"}}}`), nil)
				kubeActions.EXPECT().KubeGet(gomock.Any(), "ConfigMap", "openshift-managed-upgrade-operator", "managed-upgrade-operator-config").Return(createConfigMapBytesWithConfigYaml(`configManager:
  source: LOCAL
  localConfigName: managed-upgrade-config`), nil)
				ocmActions.EXPECT().CancelClusterUpgradePolicy(gomock.Any(), clusterInfo.UpgradePolicies[0].Id).Return(cancelUpgradeReply, nil)
			},
			endpoint:       fmt.Sprintf("/cancelocmupgradepolicy?policyID=%s", clusterInfo.UpgradePolicies[0].Id),
			httpMethod:     http.MethodPost,
			wantStatusCode: http.StatusOK,
			wantResponse:   cancelUpgradeReplyBytes,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			ocmActions := mock_adminactions.NewMockOCMActions(ti.controller)
			kubeActions := mock_adminactions.NewMockKubeActions(ti.controller)
			tc.mocks(ocmActions, kubeActions)

			err := ti.buildFixtures(tc.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return kubeActions, nil
			}, nil, func(clusterID, ocmBaseUrl, token string) adminactions.OCMActions {
				return ocmActions
			}, nil)

			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(tc.httpMethod,
				fmt.Sprintf("https://server/admin%s%s", tc.resourceID, tc.endpoint),
				nil, nil)
			if err != nil {
				t.Error(err)
			}

			// remove trailing newline added by reply method in frontend pkg.
			b = bytes.TrimSuffix(b, []byte("\n"))

			err = validateResponse(resp, b, tc.wantStatusCode, tc.wantError, tc.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
