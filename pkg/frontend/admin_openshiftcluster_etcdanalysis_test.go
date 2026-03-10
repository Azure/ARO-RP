package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
)

func TestAdminPostEtcdAnalysis(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	method := http.MethodPost
	ctx := context.Background()

	type test struct {
		name                    string
		nodeName                string
		mocks                   func(*test, *mock_adminactions.MockKubeActions)
		wantStatusCode          int
		wantResponseContentType string
		wantError               string
		wantResponseContains    []string
	}

	for _, tt := range []*test{
		{
			name:     "successful analysis streams snapshot creation and job output",
			nodeName: "master-0",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				// Snapshot exec
				k.EXPECT().
					KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdAnalysisContainer,
						gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, stdout, _ io.Writer) error {
						_, _ = fmt.Fprint(stdout, "Snapshot saved.\n")
						return nil
					})

				// Snapshot cleanup (always deferred, rm -f is a no-op on success)
				k.EXPECT().
					KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdAnalysisContainer,
						gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				// SA creation: ServiceAccount, ClusterRole, ClusterRoleBinding, SCC
				k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil).Times(4)

				// Job execution
				fakeWatcher := watch.NewFake()
				k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").
					DoAndReturn(func(_ context.Context, _ *unstructured.Unstructured, _ string) (watch.Interface, error) {
						go func() {
							fakeWatcher.Add(&unstructured.Unstructured{
								Object: map[string]interface{}{
									"kind":       "Pod",
									"apiVersion": "v1",
									"metadata":   map[string]interface{}{"name": "analysis-pod"},
									"status":     map[string]interface{}{"phase": "Succeeded"},
								},
							})
						}()
						return fakeWatcher, nil
					})

				// Job creation
				k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

				k.EXPECT().KubeFollowPodLogs(gomock.Any(), namespaceEtcds, "analysis-pod", "analyzer", gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, w io.Writer) error {
						_, _ = fmt.Fprint(w, "Analysis report output.\n")
						return nil
					})

				k.EXPECT().KubeGet(gomock.Any(), "Job.batch", namespaceEtcds, gomock.Any()).
					Return([]byte(`{"status":{"conditions":[{"type":"Complete","status":"True"}]}}`), nil)

				// Job cleanup then SA cleanup (SA, SCC, ClusterRole, CRB)
				k.EXPECT().KubeDelete(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).Times(5)
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains: []string{
				"Creating etcd snapshot on master-0...\n",
				"Snapshot saved.\n",
				"Snapshot created. Starting analysis job...\n",
				"Waiting for pod...\n",
				"Pod analysis-pod assigned, streaming logs...\n",
				"Analysis report output.\n",
				"Job succeeded.\n",
				"Cleanup complete.\n",
			},
		},
		{
			name:     "snapshot exec failure streams error and does not start job",
			nodeName: "master-0",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				// Snapshot exec fails
				k.EXPECT().
					KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdAnalysisContainer,
						gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("connection refused"))

				// Snapshot cleanup still runs (defer registered before the exec)
				k.EXPECT().
					KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdAnalysisContainer,
						gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains: []string{
				"Creating etcd snapshot on master-0...\n",
				"Snapshot failed: connection refused\n",
			},
		},
		{
			name:                    "missing nodeName returns 400",
			nodeName:                "",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided nodeName '' is invalid.",
		},
		{
			name:                    "invalid nodeName returns 400",
			nodeName:                "bad node!",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided nodeName 'bad node!' is invalid.",
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

			u := fmt.Sprintf("https://server/admin%s/etcdanalysis?nodeName=%s", resourceID, url.QueryEscape(tt.nodeName))
			resp, b, err := ti.request(method, u, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("unexpected status code %d, wanted %d: %s", resp.StatusCode, tt.wantStatusCode, string(b))
			}

			if tt.wantResponseContentType != resp.Header.Get("Content-Type") {
				t.Errorf("unexpected Content-Type %q, wanted %q",
					resp.Header.Get("Content-Type"), tt.wantResponseContentType)
			}

			if tt.wantError != "" {
				cloudErr := &api.CloudError{StatusCode: resp.StatusCode}
				if err := json.Unmarshal(b, cloudErr); err != nil {
					t.Fatalf("failed to unmarshal error response: %v", err)
				}
				if cloudErr.Error() != tt.wantError {
					t.Errorf("unexpected error %q, wanted %q", cloudErr.Error(), tt.wantError)
				}
			}

			for _, want := range tt.wantResponseContains {
				if !strings.Contains(string(b), want) {
					t.Errorf("response does not contain %q\nfull response: %s", want, string(b))
				}
			}
		})
	}
}
