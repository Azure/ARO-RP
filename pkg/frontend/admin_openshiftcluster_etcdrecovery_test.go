package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	kschema "k8s.io/apimachinery/pkg/runtime/schema"
	ktesting "k8s.io/client-go/testing"

	operatorv1client "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"
	operatorv1fake "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
)

func fakeRecoveryDoc(privateEndpoint bool, resourceID, resourceName string) *api.OpenShiftClusterDocument {
	netProfile := api.NetworkProfile{}
	if privateEndpoint {
		netProfile = api.NetworkProfile{
			APIServerPrivateEndpointIP: "0.0.0.0",
		}
	}
	doc := &api.OpenShiftClusterDocument{
		Key: strings.ToLower(resourceID),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID:   resourceID,
			Name: resourceName,
			Type: "Microsoft.RedHatOpenShift/openshiftClusters",
			Properties: api.OpenShiftClusterProperties{
				NetworkProfile: netProfile,
				AdminKubeconfig: api.SecureBytes(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://server
name: cluster
`),
				KubeadminPassword: api.SecureString("p"),
				InfraID:           "zfsbk",
			},
		},
	}

	return doc
}

func TestAdminEtcdRecovery(t *testing.T) {
	const (
		resourceName = "cluster"
		mockSubID    = "00000000-0000-0000-0000-000000000000"
		mockTenantID = mockSubID
		method       = http.MethodPost
	)
	ctx := context.Background()

	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", mockSubID, resourceName)
	gvk := kschema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "Etcd",
	}
	type test struct {
		name                    string
		mocks                   func(ctx context.Context, ti *testInfra, k *mock_adminactions.MockKubeActions, log *logrus.Entry, env env.Interface, doc *api.OpenShiftClusterDocument, pods *corev1.PodList, etcdcli operatorv1client.EtcdInterface)
		wantStatusCode          int
		wantResponse            []byte
		wantResponseContentType string
		wantError               string
		doc                     *api.OpenShiftClusterDocument
		kubeActionsFactory      func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error)
	}
	for _, tt := range []*test{
		{
			name:                    "fail: parse group kind resource",
			wantStatusCode:          http.StatusInternalServerError,
			wantResponseContentType: "application/json",
			wantError:               "500: InternalServerError: : failed to parse resource",
			doc:                     fakeRecoveryDoc(true, resourceID, resourceName),
			mocks: func(ctx context.Context, ti *testInfra, k *mock_adminactions.MockKubeActions, log *logrus.Entry, env env.Interface, doc *api.OpenShiftClusterDocument, pods *corev1.PodList, etcdcli operatorv1client.EtcdInterface) {
				k.EXPECT().ResolveGVR("Etcd", "").Times(1).Return(gvk, errors.New("failed to parse resource"))
			},
		},
		{
			name:                    "fail: validate kubernetes objects",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided resource is invalid.",
			doc:                     fakeRecoveryDoc(true, resourceID, resourceName),
			mocks: func(ctx context.Context, ti *testInfra, k *mock_adminactions.MockKubeActions, log *logrus.Entry, env env.Interface, doc *api.OpenShiftClusterDocument, pods *corev1.PodList, etcdcli operatorv1client.EtcdInterface) {
				k.EXPECT().ResolveGVR("Etcd", "").Times(1).Return(kschema.GroupVersionResource{}, nil)
			},
		},
		{
			name:                    "fail: privateEndpointIP cannot be empty",
			wantStatusCode:          http.StatusInternalServerError,
			wantResponseContentType: "application/json",
			wantError:               "500: InternalServerError: : privateEndpointIP is empty",
			doc:                     fakeRecoveryDoc(false, resourceID, resourceName),
			mocks: func(ctx context.Context, ti *testInfra, k *mock_adminactions.MockKubeActions, log *logrus.Entry, env env.Interface, doc *api.OpenShiftClusterDocument, pods *corev1.PodList, etcdcli operatorv1client.EtcdInterface) {
				k.EXPECT().ResolveGVR("Etcd", "").Times(1).Return(gvk, nil)
			},
		},
		{
			name:                    "fail: kubeActionsFactory error",
			wantStatusCode:          http.StatusInternalServerError,
			wantResponseContentType: "application/json",
			wantError:               "500: InternalServerError: : failed to create kubeactions",
			doc:                     fakeRecoveryDoc(true, resourceID, resourceName),
			kubeActionsFactory: func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return nil, errors.New("failed to create kubeactions")
			},
		},
	} {
		t.Run(fmt.Sprintf("%s: %s", method, tt.name), func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			ti.fixture.AddOpenShiftClusterDocuments(tt.doc)
			ti.fixture.AddSubscriptionDocuments(&api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{
						TenantID: mockTenantID,
					},
				},
			})

			err := ti.buildFixtures(nil)
			if err != nil {
				t.Fatal(err)
			}

			k := mock_adminactions.NewMockKubeActions(ti.controller)
			if tt.mocks != nil {
				tt.mocks(ctx, ti, k, ti.log, ti.env, tt.doc, newEtcdPods(t, tt.doc, false, false, false), &operatorv1fake.FakeEtcds{
					Fake: &operatorv1fake.FakeOperatorV1{
						Fake: &ktesting.Fake{},
					},
				})
			}

			kubeActionsFactory := func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return k, nil
			}
			if tt.kubeActionsFactory != nil {
				kubeActionsFactory = tt.kubeActionsFactory
			}
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
				kubeActionsFactory,
				nil,
				nil,
				ti.enricher)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/admin%s/etcdrecovery?api-version=admin", resourceID),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
			if tt.wantResponseContentType != resp.Header.Get("Content-Type") {
				t.Error(fmt.Errorf("unexpected \"Content-Type\" response header value \"%s\", wanted \"%s\"", resp.Header.Get("Content-Type"), tt.wantResponseContentType))
			}
		})
	}
}
