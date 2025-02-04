package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestAdminEtcdCertificateRenew(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	mockRG := "resourceGroup"
	ctx := context.Background()

	type test struct {
		name                 string
		resourceID           string
		version              *configv1.ClusterVersion
		etcdoperator         *operatorv1.Etcd
		etcdoperatorRevisied *operatorv1.Etcd
		etcdCO               *configv1.ClusterOperator
		notBefore            time.Time
		notAfter             time.Time
		renewedNotBefore     time.Time
		renewedNotAfter      time.Time
		mocks                func(*test, *mock_adminactions.MockKubeActions)
		wantStatusCode       int
		wantResponse         []byte
		wantError            string
	}

	for _, tt := range []*test{
		{
			name:       "validate cluster version is <4.9",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			version: &configv1.ClusterVersion{
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.11.44",
						},
					},
				},
			},
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterVersion.config.openshift.io", "", "version").
					Return(encodeClusterVersion(t, tt.version), nil)
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      "403: Forbidden: : etcd certificate renewal is not needed for cluster running version 4.9+",
		},
		{
			name:       "validate etcd operator controller status",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			version: &configv1.ClusterVersion{
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.8.11",
						},
					},
				},
			},
			etcdoperator: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterVersion.config.openshift.io", "", "version").MaxTimes(1).
					Return(encodeClusterVersion(t, tt.version), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").
					Return(encodeEtcdOperatorController(t, tt.etcdoperator), nil)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : EtcdCertSignerControllerDegraded is in state True, quiting.",
		},
		{
			name:       "validate etcd cluster operator status",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			version: &configv1.ClusterVersion{
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.8.11",
						},
					},
				},
			},
			etcdoperator: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionFalse,
								},
							},
						},
					},
				},
			},
			etcdCO: &configv1.ClusterOperator{
				Status: configv1.ClusterOperatorStatus{
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:   configv1.OperatorDegraded,
							Status: configv1.ConditionTrue,
						},
					},
				},
			},
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterVersion.config.openshift.io", "", "version").MaxTimes(1).
					Return(encodeClusterVersion(t, tt.version), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").MaxTimes(1).
					Return(encodeEtcdOperatorController(t, tt.etcdoperator), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
					Return(encodeEtcdOperator(t, tt.etcdCO), nil)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : Etcd Operator is not in expected state, quiting.",
		},
		{
			name:       "validate etcd cluster operator Available state",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			version: &configv1.ClusterVersion{
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.8.11",
						},
					},
				},
			},
			etcdoperator: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionFalse,
								},
							},
						},
					},
				},
			},
			etcdCO: &configv1.ClusterOperator{
				Status: configv1.ClusterOperatorStatus{
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:   configv1.OperatorAvailable,
							Status: configv1.ConditionTrue,
							Reason: "UnExpected",
						},
					},
				},
			},
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterVersion.config.openshift.io", "", "version").MaxTimes(1).
					Return(encodeClusterVersion(t, tt.version), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").MaxTimes(1).
					Return(encodeEtcdOperatorController(t, tt.etcdoperator), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
					Return(encodeEtcdOperator(t, tt.etcdCO), nil)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : Etcd Operator Available state is not AsExpected, quiting.",
		},
		{
			name:       "validate if etcd certificates are expired",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			version: &configv1.ClusterVersion{
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.8.11",
						},
					},
				},
			},
			etcdoperator: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionFalse,
								},
							},
						},
					},
				},
			},
			etcdCO: &configv1.ClusterOperator{
				Status: configv1.ClusterOperatorStatus{
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:   configv1.OperatorDegraded,
							Status: configv1.ConditionFalse,
						},
					},
				},
			},
			notBefore: time.Now(),
			notAfter:  time.Now().Add(-10 * time.Minute),
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterVersion.config.openshift.io", "", "version").MaxTimes(1).
					Return(encodeClusterVersion(t, tt.version), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").MinTimes(1).
					Return(encodeEtcdOperatorController(t, tt.etcdoperator), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").MinTimes(1).
					Return(encodeEtcdOperator(t, tt.etcdCO), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "Secret", namespaceEtcds, gomock.Any()).MinTimes(1).
					Return(createCertSecret(t, tt.notBefore, tt.notAfter), nil)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : secret etcd-peer-cluster-aro-master-0 is already expired, quitting.",
		},
		{
			name:       "etcd certificates delete and successful renewal",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			version: &configv1.ClusterVersion{
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.8.11",
						},
					},
				},
			},
			etcdoperator: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionFalse,
								},
							},
						},
						LatestAvailableRevision: 1,
						NodeStatuses: []operatorv1.NodeStatus{
							{
								NodeName:        "master-0",
								CurrentRevision: 1,
							},
						},
					},
				},
			},
			etcdoperatorRevisied: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionFalse,
								},
							},
						},
						LatestAvailableRevision: 2,
						NodeStatuses: []operatorv1.NodeStatus{
							{
								NodeName:        "master-0",
								CurrentRevision: 2,
							},
						},
					},
				},
			},
			etcdCO: &configv1.ClusterOperator{
				Status: configv1.ClusterOperatorStatus{
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:   configv1.OperatorDegraded,
							Status: configv1.ConditionFalse,
						},
					},
				},
			},
			notBefore:        time.Now().AddDate(-2, -8, 0),
			notAfter:         time.Now().Add((1 * time.Hour)),
			renewedNotBefore: time.Now(),
			renewedNotAfter:  time.Now().AddDate(3, 0, 0),
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterVersion.config.openshift.io", "", "version").MaxTimes(1).
					Return(encodeClusterVersion(t, tt.version), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").MaxTimes(2).
					Return(encodeEtcdOperatorController(t, tt.etcdoperator), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").MinTimes(1).
					Return(encodeEtcdOperator(t, tt.etcdCO), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "Secret", namespaceEtcds, gomock.Any()).MaxTimes(18).
					Return(createCertSecret(t, tt.notBefore, tt.notAfter), nil)
				d := k.EXPECT().
					KubeDelete(gomock.Any(), "Secret", namespaceEtcds, gomock.Any(), false, nil).MinTimes(9).
					Return(nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "Secret", namespaceEtcds, gomock.Any()).After(d).MinTimes(1).
					Return(createCertSecret(t, tt.renewedNotBefore, tt.renewedNotAfter), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").MinTimes(1).After(d).
					Return(encodeEtcdOperatorController(t, tt.etcdoperatorRevisied), nil)
			},
			wantStatusCode: http.StatusOK,
			wantError:      "",
		},
		{
			name:       "validate if etcd certificates are expired",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			version: &configv1.ClusterVersion{
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.8.11",
						},
					},
				},
			},
			etcdoperator: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionFalse,
								},
							},
						},
					},
				},
			},
			etcdCO: &configv1.ClusterOperator{
				Status: configv1.ClusterOperatorStatus{
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:   configv1.OperatorDegraded,
							Status: configv1.ConditionFalse,
						},
					},
				},
			},
			notBefore: time.Now(),
			notAfter:  time.Now().Add(-10 * time.Minute),
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterVersion.config.openshift.io", "", "version").MaxTimes(1).
					Return(encodeClusterVersion(t, tt.version), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").MinTimes(1).
					Return(encodeEtcdOperatorController(t, tt.etcdoperator), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").MinTimes(1).
					Return(encodeEtcdOperator(t, tt.etcdCO), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "Secret", namespaceEtcds, gomock.Any()).MinTimes(1).
					Return(createCertSecret(t, tt.notBefore, tt.notAfter), nil)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : secret etcd-peer-cluster-aro-master-0 is already expired, quitting.",
		},
		{
			name:       "etcd certificates deleted but not renewed",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			version: &configv1.ClusterVersion{
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.8.11",
						},
					},
				},
			},
			etcdoperator: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionFalse,
								},
							},
						},
						LatestAvailableRevision: 1,
						NodeStatuses: []operatorv1.NodeStatus{
							{
								NodeName:        "master-0",
								CurrentRevision: 1,
							},
						},
					},
				},
			},
			etcdoperatorRevisied: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionFalse,
								},
							},
						},
						LatestAvailableRevision: 2,
						NodeStatuses: []operatorv1.NodeStatus{
							{
								NodeName:        "master-0",
								CurrentRevision: 2,
							},
						},
					},
				},
			},
			etcdCO: &configv1.ClusterOperator{
				Status: configv1.ClusterOperatorStatus{
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:   configv1.OperatorDegraded,
							Status: configv1.ConditionFalse,
						},
					},
				},
			},
			notBefore:        time.Now().AddDate(-2, -8, 0),
			notAfter:         time.Now().Add((1 * time.Hour)),
			renewedNotBefore: time.Now(),
			renewedNotAfter:  time.Now().AddDate(0, 0, 1),
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterVersion.config.openshift.io", "", "version").MaxTimes(1).
					Return(encodeClusterVersion(t, tt.version), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").MaxTimes(2).
					Return(encodeEtcdOperatorController(t, tt.etcdoperator), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").MinTimes(1).
					Return(encodeEtcdOperator(t, tt.etcdCO), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "Secret", namespaceEtcds, gomock.Any()).MaxTimes(18).
					Return(createCertSecret(t, tt.notBefore, tt.notAfter), nil)
				d := k.EXPECT().
					KubeDelete(gomock.Any(), "Secret", namespaceEtcds, gomock.Any(), false, nil).MinTimes(9).
					Return(nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "Secret", namespaceEtcds, gomock.Any()).After(d).MinTimes(1).
					Return(createCertSecret(t, tt.renewedNotBefore, tt.renewedNotAfter), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").MinTimes(1).After(d).
					Return(encodeEtcdOperatorController(t, tt.etcdoperatorRevisied), nil)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : etcd certificates renewal not successful, as at least one or all certificates are not renewed",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			k := mock_adminactions.NewMockKubeActions(ti.controller)
			tt.mocks(tt, k)

			f, err := NewFrontend(ctx,
				ti.audit,
				ti.log,
				ti.env,
				ti.dbGroup,
				api.APIs,
				&noop.Noop{},
				&noop.Noop{},
				nil,
				nil,
				nil,
				func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
					return k, nil
				},
				nil,
				nil,
				nil)
			if err != nil {
				t.Fatal(err)
			}

			ti.fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(tt.resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: api.OpenShiftClusterProperties{
						InfraID:       "cluster-aro",
						StorageSuffix: "xxx",
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", mockSubID, mockRG),
						},
					},
				},
			})
			ti.fixture.AddSubscriptionDocuments(&api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{
						TenantID: mockTenantID,
					},
				},
			})

			err = ti.buildFixtures(nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodPost,
				fmt.Sprintf("https://server/admin%s/etcdcertificaterenew", tt.resourceID),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestAdminEtcdCertificateRecovery(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	mockRG := "resourceGroup"
	ctx := context.Background()

	type test struct {
		name                  string
		resourceID            string
		version               *configv1.ClusterVersion
		etcdoperator          *operatorv1.Etcd
		etcdoperatorRevisied  *operatorv1.Etcd
		etcdoperatorRecovered *operatorv1.Etcd
		etcdCO                *configv1.ClusterOperator
		notBefore             time.Time
		notAfter              time.Time
		mocks                 func(*test, *mock_adminactions.MockKubeActions)
		wantStatusCode        int
		wantError             string
		timeout               int
	}

	for _, tt := range []*test{
		{
			name:       "etcd secrets recovery fails on timeout",
			timeout:    0,
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			version: &configv1.ClusterVersion{
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.8.11",
						},
					},
				},
			},
			etcdoperator: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionFalse,
								},
							},
						},
						LatestAvailableRevision: 1,
						NodeStatuses: []operatorv1.NodeStatus{
							{
								NodeName:        "master-0",
								CurrentRevision: 1,
							},
						},
					},
				},
			},
			etcdoperatorRevisied: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionFalse,
								},
							},
						},
						LatestAvailableRevision: 2,
						NodeStatuses: []operatorv1.NodeStatus{
							{
								NodeName:        "master-0",
								CurrentRevision: 1,
							},
						},
					},
				},
			},
			etcdoperatorRecovered: &operatorv1.Etcd{
				Status: operatorv1.EtcdStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   "EtcdCertSignerControllerDegraded",
									Status: operatorv1.ConditionFalse,
								},
							},
						},
						LatestAvailableRevision: 3,
						NodeStatuses: []operatorv1.NodeStatus{
							{
								NodeName:        "master-0",
								CurrentRevision: 3,
							},
						},
					},
				},
			},
			etcdCO: &configv1.ClusterOperator{
				Status: configv1.ClusterOperatorStatus{
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:   configv1.OperatorDegraded,
							Status: configv1.ConditionFalse,
						},
					},
				},
			},
			notBefore: time.Now().AddDate(-2, -8, 0),
			notAfter:  time.Now().Add((1 * time.Hour)),
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterVersion.config.openshift.io", "", "version").Times(1).
					Return(encodeClusterVersion(t, tt.version), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").MaxTimes(2).
					Return(encodeEtcdOperatorController(t, tt.etcdoperator), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").MinTimes(1).
					Return(encodeEtcdOperator(t, tt.etcdCO), nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "Secret", namespaceEtcds, gomock.Any()).MinTimes(9).
					Return(createCertSecret(t, tt.notBefore, tt.notAfter), nil)
				d := k.EXPECT().
					KubeDelete(gomock.Any(), "Secret", namespaceEtcds, gomock.Any(), false, nil).MinTimes(9).
					Return(nil)
				c := k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").MinTimes(1).After(d).
					Return(encodeEtcdOperatorController(t, tt.etcdoperatorRevisied), nil)
				r := k.EXPECT().
					KubeCreateOrUpdate(gomock.Any(), gomock.Any()).MinTimes(9).After(c).
					Return(nil)
				k.EXPECT().
					KubeGet(gomock.Any(), "etcd.operator.openshift.io", "", "cluster").MinTimes(1).After(r).
					Return(encodeEtcdOperatorController(t, tt.etcdoperatorRecovered), nil)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : etcd renewal failed, recovery performed to revert the changes.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			k := mock_adminactions.NewMockKubeActions(ti.controller)
			tt.mocks(tt, k)

			f, err := NewFrontend(ctx,
				ti.audit,
				ti.log,
				ti.env,
				ti.dbGroup,
				api.APIs,
				&noop.Noop{},
				&noop.Noop{},
				nil,
				nil,
				nil,
				func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
					return k, nil
				},
				nil,
				nil,
				nil)
			if err != nil {
				t.Fatal(err)
			}
			doc := &api.OpenShiftClusterDocument{
				Key: strings.ToLower(tt.resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: api.OpenShiftClusterProperties{
						InfraID:       "cluster-aro",
						StorageSuffix: "xxx",
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", mockSubID, mockRG),
						},
					},
				},
			}
			ti.fixture.AddOpenShiftClusterDocuments(doc)
			ti.fixture.AddSubscriptionDocuments(&api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{
						TenantID: mockTenantID,
					},
				},
			})

			err = ti.buildFixtures(nil)
			if err != nil {
				t.Fatal(err)
			}

			log := logrus.NewEntry(logrus.New())

			err = f._postAdminOpenShiftClusterEtcdCertificateRenew(ctx, strings.ToLower(tt.resourceID), log, time.Duration(tt.timeout)*time.Second)
			utilerror.AssertErrorMessage(t, err, tt.wantError)
		})
	}
}

func encodeClusterVersion(t *testing.T, version *configv1.ClusterVersion) []byte {
	buf := &bytes.Buffer{}
	err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(version)
	if err != nil {
		t.Fatalf("%s failed to encode version, %s", t.Name(), err.Error())
	}
	return buf.Bytes()
}

func encodeEtcdOperatorController(t *testing.T, etcd *operatorv1.Etcd) []byte {
	buf := &bytes.Buffer{}
	err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(etcd)
	if err != nil {
		t.Fatalf("%s failed to encode etcd operator, %s", t.Name(), err.Error())
	}
	return buf.Bytes()
}

func encodeEtcdOperator(t *testing.T, etcdCO *configv1.ClusterOperator) []byte {
	buf := &bytes.Buffer{}
	err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(etcdCO)
	if err != nil {
		t.Fatalf("%s failed to encode etcd operator, %s", t.Name(), err.Error())
	}
	return buf.Bytes()
}

func encodeSecret(t *testing.T, secret *corev1.Secret) []byte {
	buf := &bytes.Buffer{}
	err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(secret)
	if err != nil {
		t.Fatalf("%s failed to encode etcd secret, %s", t.Name(), err.Error())
	}
	return buf.Bytes()
}

func tweakTemplateFn(notBefore time.Time, notAfter time.Time) func(*x509.Certificate) {
	return func(template *x509.Certificate) {
		template.NotBefore = notBefore
		template.NotAfter = notAfter
	}
}

func createCertSecret(t *testing.T, notBefore time.Time, notAfter time.Time) []byte {
	secretname := "etcd-cert"
	_, cert, err := utiltls.GenerateTestKeyAndCertificate(secretname, nil, nil, false, false, tweakTemplateFn(notBefore, notAfter))
	if err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind: "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretname,
			Namespace: "openshift-etcd",
		},
		Data: map[string][]byte{
			corev1.TLSCertKey: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert[0].Raw}),
		},
		Type: corev1.SecretTypeTLS,
	}
	return encodeSecret(t, secret)
}
