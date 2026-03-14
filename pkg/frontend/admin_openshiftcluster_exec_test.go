package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
)

func TestAdminPostExec(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	method := http.MethodPost
	ctx := context.Background()

	type test struct {
		name                    string
		namespace               string
		podName                 string
		container               string
		command                 string
		mocks                   func(*test, *mock_adminactions.MockKubeActions)
		wantStatusCode          int
		wantResponse            []byte
		wantResponseContentType string
		wantError               string
	}

	for _, tt := range []*test{
		{
			name:      "successful exec streams stdout then Done",
			namespace: "openshift-etcd",
			podName:   "etcd-master-0",
			container: "etcdctl",
			command:   "echo hello",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeExecStream(gomock.Any(), tt.namespace, tt.podName, tt.container,
						[]string{"sh", "-c", tt.command}, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, stdout, _ io.Writer) error {
						_, _ = fmt.Fprint(stdout, "hello\n")
						return nil
					})
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			// ReplyStream appends one '\n' after io.Copy.
			wantResponse: []byte("Executing in openshift-etcd/etcd-master-0/etcdctl...\nhello\nDone.\n\n"),
		},
		{
			name:      "exec writes stderr section when command produces stderr",
			namespace: "openshift-etcd",
			podName:   "etcd-master-0",
			container: "etcdctl",
			command:   "echo err >&2",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeExecStream(gomock.Any(), tt.namespace, tt.podName, tt.container,
						[]string{"sh", "-c", tt.command}, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, _, stderr io.Writer) error {
						_, _ = fmt.Fprint(stderr, "err\n")
						return nil
					})
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponse:            []byte("Executing in openshift-etcd/etcd-master-0/etcdctl...\nstderr:\nerr\nDone.\n\n"),
		},
		{
			name:      "exec failure appends Command failed line",
			namespace: "openshift-etcd",
			podName:   "etcd-master-0",
			container: "etcdctl",
			command:   "false",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeExecStream(gomock.Any(), tt.namespace, tt.podName, tt.container,
						[]string{"sh", "-c", tt.command}, gomock.Any(), gomock.Any()).
					Return(errors.New("exit code 1"))
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponse:            []byte("Executing in openshift-etcd/etcd-master-0/etcdctl...\nCommand failed: exit code 1\n\n"),
		},
		{
			name:                    "missing namespace returns 400",
			namespace:               "",
			podName:                 "etcd-master-0",
			container:               "etcdctl",
			command:                 "ls",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided namespace '' is invalid.",
		},
		{
			name:                    "customer namespace returns 403",
			namespace:               "customer-namespace",
			podName:                 "etcd-master-0",
			container:               "etcdctl",
			command:                 "ls",
			wantStatusCode:          http.StatusForbidden,
			wantResponseContentType: "application/json",
			wantError:               "403: Forbidden: : Access to the provided namespace 'customer-namespace' is forbidden.",
		},
		{
			name:                    "missing pod name returns 400",
			namespace:               "openshift-etcd",
			podName:                 "",
			container:               "etcdctl",
			command:                 "ls",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided pod name '' is invalid.",
		},
		{
			name:                    "missing container returns 400",
			namespace:               "openshift-etcd",
			podName:                 "etcd-master-0",
			container:               "",
			command:                 "ls",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided container name '' is invalid.",
		},
		{
			name:                    "missing command returns 400",
			namespace:               "openshift-etcd",
			podName:                 "etcd-master-0",
			container:               "etcdctl",
			command:                 "",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided command must not be empty.",
		},
	} {
		t.Run(fmt.Sprintf("%s: %s", method, tt.name), func(t *testing.T) {
			resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			k := mock_adminactions.NewMockKubeActions(ti.controller)
			if tt.mocks != nil {
				tt.mocks(tt, k)
			}

			ti.fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: api.OpenShiftClusterProperties{
						NetworkProfile: api.NetworkProfile{
							APIServerPrivateEndpointIP: "0.0.0.0",
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

			err := ti.buildFixtures(nil)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return k, nil
			}, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			body := map[string]interface{}{
				"namespace": tt.namespace,
				"podName":   tt.podName,
				"container": tt.container,
				"command":   tt.command,
			}
			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/admin%s/exec", resourceID),
				http.Header{"Content-Type": []string{"application/json"}},
				body)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
			if tt.wantResponseContentType != resp.Header.Get("Content-Type") {
				t.Error(fmt.Errorf("unexpected \"Content-Type\" response header value %q, wanted %q",
					resp.Header.Get("Content-Type"), tt.wantResponseContentType))
			}
		})
	}
}
