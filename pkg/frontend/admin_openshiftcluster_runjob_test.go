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

func TestAdminPostRunJob(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	method := http.MethodPost
	ctx := context.Background()

	minimalJobBody := map[string]interface{}{
		"kind":       "Job",
		"apiVersion": "batch/v1",
		"metadata":   map[string]interface{}{"name": "test-job"},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"restartPolicy": "Never",
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "worker",
							"image": "busybox",
						},
					},
				},
			},
		},
	}

	type test struct {
		name                    string
		body                    interface{}
		mocks                   func(*test, *mock_adminactions.MockKubeActions)
		wantStatusCode          int
		wantResponseContentType string
		wantError               string
		wantResponseContains    []string
	}

	for _, tt := range []*test{
		{
			name: "successful job run streams progress and logs",
			body: minimalJobBody,
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				fakeWatcher := watch.NewFake()
				k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").
					DoAndReturn(func(_ context.Context, _ *unstructured.Unstructured, _ string) (watch.Interface, error) {
						go func() {
							fakeWatcher.Add(&unstructured.Unstructured{
								Object: map[string]interface{}{
									"kind":       "Pod",
									"apiVersion": "v1",
									"metadata":   map[string]interface{}{"name": "test-pod"},
									"status":     map[string]interface{}{"phase": "Running"},
								},
							})
						}()
						return fakeWatcher, nil
					})

				k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

				k.EXPECT().KubeFollowPodLogs(gomock.Any(), runJobDefaultNamespace, "test-pod", "worker", gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, w io.Writer) error {
						_, _ = fmt.Fprint(w, "hello from job\n")
						return nil
					})

				k.EXPECT().KubeGet(gomock.Any(), "Job.batch", runJobDefaultNamespace, gomock.Any()).
					Return([]byte(`{"status":{"conditions":[{"type":"Complete","status":"True"}]}}`), nil)

				k.EXPECT().KubeDelete(gomock.Any(), "Job.batch", runJobDefaultNamespace, gomock.Any(), false, gomock.Any()).
					Return(nil)
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains: []string{
				"in " + runJobDefaultNamespace + "...\n",
				"Waiting for pod...\n",
				"Pod test-pod assigned, streaming logs...\n",
				"hello from job\n",
				"Job succeeded.\n",
				"Cleanup complete.\n",
			},
		},
		{
			name: "job result Failed=True streams Job failed",
			body: minimalJobBody,
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				fakeWatcher := watch.NewFake()
				k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").
					DoAndReturn(func(_ context.Context, _ *unstructured.Unstructured, _ string) (watch.Interface, error) {
						go func() {
							fakeWatcher.Add(&unstructured.Unstructured{
								Object: map[string]interface{}{
									"kind":       "Pod",
									"apiVersion": "v1",
									"metadata":   map[string]interface{}{"name": "test-pod"},
									"status":     map[string]interface{}{"phase": "Failed"},
								},
							})
						}()
						return fakeWatcher, nil
					})

				k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

				k.EXPECT().KubeFollowPodLogs(gomock.Any(), runJobDefaultNamespace, "test-pod", "worker", gomock.Any()).
					Return(nil)

				k.EXPECT().KubeGet(gomock.Any(), "Job.batch", runJobDefaultNamespace, gomock.Any()).
					Return([]byte(`{"status":{"conditions":[{"type":"Failed","status":"True"}]}}`), nil)

				k.EXPECT().KubeDelete(gomock.Any(), "Job.batch", runJobDefaultNamespace, gomock.Any(), false, gomock.Any()).
					Return(nil)
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains:    []string{"Job failed.\n", "Cleanup complete.\n"},
		},
		{
			name: "job result with no terminal condition streams Job result pending",
			body: minimalJobBody,
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				fakeWatcher := watch.NewFake()
				k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").
					DoAndReturn(func(_ context.Context, _ *unstructured.Unstructured, _ string) (watch.Interface, error) {
						go func() {
							fakeWatcher.Add(&unstructured.Unstructured{
								Object: map[string]interface{}{
									"kind":       "Pod",
									"apiVersion": "v1",
									"metadata":   map[string]interface{}{"name": "test-pod"},
									"status":     map[string]interface{}{"phase": "Succeeded"},
								},
							})
						}()
						return fakeWatcher, nil
					})

				k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

				k.EXPECT().KubeFollowPodLogs(gomock.Any(), runJobDefaultNamespace, "test-pod", "worker", gomock.Any()).
					Return(nil)

				k.EXPECT().KubeGet(gomock.Any(), "Job.batch", runJobDefaultNamespace, gomock.Any()).
					Return([]byte(`{"status":{"conditions":[]}}`), nil)

				k.EXPECT().KubeDelete(gomock.Any(), "Job.batch", runJobDefaultNamespace, gomock.Any(), false, gomock.Any()).
					Return(nil)
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains:    []string{"Job result: pending\n"},
		},
		{
			name: "job creation failure streams error without cleanup",
			body: minimalJobBody,
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").
					Return(watch.NewFake(), nil)

				k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(errors.New("quota exceeded"))
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains:    []string{"Failed to create job: quota exceeded\n"},
		},
		{
			name: "pod watch setup failure streams error without creating job",
			body: minimalJobBody,
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").
					Return(nil, errors.New("watch failed"))
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains:    []string{"Error setting up pod watch: watch failed\n"},
		},
		{
			name:                    "empty body returns 400",
			body:                    nil,
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidRequestContent: : The request body must not be empty.",
		},
		{
			name:                    "wrong kind returns 400",
			body:                    map[string]interface{}{"kind": "Pod"},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : Expected kind 'Job', got 'Pod'.",
		},
		{
			name: "missing name returns 400",
			body: map[string]interface{}{
				"kind":     "Job",
				"metadata": map[string]interface{}{"name": ""},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided Job manifest must have a non-empty metadata.name.",
		},
		{
			name: "invalid name returns 400",
			body: map[string]interface{}{
				"kind":     "Job",
				"metadata": map[string]interface{}{"name": "invalid name!"},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided Job metadata.name 'invalid name!' is invalid.",
		},
		{
			name: "customer namespace returns 403",
			body: map[string]interface{}{
				"kind":     "Job",
				"metadata": map[string]interface{}{"name": "test-job", "namespace": "customer-app"},
			},
			wantStatusCode:          http.StatusForbidden,
			wantResponseContentType: "application/json",
			wantError:               "403: Forbidden: : Access to the provided namespace 'customer-app' is forbidden.",
		},
		{
			name: "invalid namespace returns 400",
			body: map[string]interface{}{
				"kind":     "Job",
				"metadata": map[string]interface{}{"name": "test-job", "namespace": "bad namespace!"},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided Job metadata.namespace 'bad namespace!' is invalid.",
		},
		{
			name: "parallelism > 1 returns 400",
			body: map[string]interface{}{
				"kind":       "Job",
				"apiVersion": "batch/v1",
				"metadata":   map[string]interface{}{"name": "test-job"},
				"spec": map[string]interface{}{
					"parallelism": 2,
					"template":    minimalJobBody["spec"].(map[string]interface{})["template"],
				},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : Jobs with spec.parallelism > 1 are not implemented.",
		},
		{
			name: "completions > 1 returns 400",
			body: map[string]interface{}{
				"kind":       "Job",
				"apiVersion": "batch/v1",
				"metadata":   map[string]interface{}{"name": "test-job"},
				"spec": map[string]interface{}{
					"completions": 3,
					"template":    minimalJobBody["spec"].(map[string]interface{})["template"],
				},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : Jobs with spec.completions > 1 are not implemented.",
		},
		{
			name: "zero containers returns 400",
			body: map[string]interface{}{
				"kind":       "Job",
				"apiVersion": "batch/v1",
				"metadata":   map[string]interface{}{"name": "test-job"},
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"restartPolicy": "Never",
							"containers":    []interface{}{},
						},
					},
				},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The Job pod template must define exactly one container, got 0.",
		},
		{
			name: "multiple containers returns 400",
			body: map[string]interface{}{
				"kind":       "Job",
				"apiVersion": "batch/v1",
				"metadata":   map[string]interface{}{"name": "test-job"},
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"restartPolicy": "Never",
							"containers": []interface{}{
								map[string]interface{}{"name": "c1", "image": "busybox"},
								map[string]interface{}{"name": "c2", "image": "busybox"},
							},
						},
					},
				},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The Job pod template must define exactly one container, got 2.",
		},
		{
			name: "empty container name returns 400",
			body: map[string]interface{}{
				"kind":       "Job",
				"apiVersion": "batch/v1",
				"metadata":   map[string]interface{}{"name": "test-job"},
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"restartPolicy": "Never",
							"containers": []interface{}{
								map[string]interface{}{"name": "", "image": "busybox"},
							},
						},
					},
				},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The Job pod template must specify a non-empty container name.",
		},
		{
			name: "invalid container name returns 400",
			body: map[string]interface{}{
				"kind":       "Job",
				"apiVersion": "batch/v1",
				"metadata":   map[string]interface{}{"name": "test-job"},
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"restartPolicy": "Never",
							"containers": []interface{}{
								map[string]interface{}{"name": "bad name!", "image": "busybox"},
							},
						},
					},
				},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The Job pod template container name 'bad name!' is invalid.",
		},
		{
			name: "name longer than 57 chars is truncated to produce a 63-char final name",
			body: map[string]interface{}{
				"kind":       "Job",
				"apiVersion": "batch/v1",
				"metadata":   map[string]interface{}{"name": strings.Repeat("a", 70)},
				"spec":       minimalJobBody["spec"],
			},
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").
					Return(watch.NewFake(), nil)

				k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, obj *unstructured.Unstructured) error {
						if got := len(obj.GetName()); got != 63 {
							return fmt.Errorf("expected job name length 63 after truncation, got %d", got)
						}
						return fmt.Errorf("stop")
					})
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains:    []string{"Failed to create job"},
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

			var header http.Header
			if tt.body != nil {
				header = http.Header{"Content-Type": []string{"application/json"}}
			}
			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/admin%s/runjob", resourceID),
				header, tt.body)
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
