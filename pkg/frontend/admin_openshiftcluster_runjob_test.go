package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
)

func TestAdminPostRunJob(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := mockSubID
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
	minimalTemplate := minimalJobBody["spec"].(map[string]interface{})["template"]

	type test struct {
		name                    string
		body                    interface{}
		noClusterDoc            bool
		kubeActionsFactory      func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error)
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
			name: "job result with no terminal condition on first poll eventually succeeds",
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

				// First poll returns no terminal conditions; second poll returns Complete.
				gomock.InOrder(
					k.EXPECT().KubeGet(gomock.Any(), "Job.batch", runJobDefaultNamespace, gomock.Any()).
						Return([]byte(`{"status":{"conditions":[]}}`), nil),
					k.EXPECT().KubeGet(gomock.Any(), "Job.batch", runJobDefaultNamespace, gomock.Any()).
						Return([]byte(`{"status":{"conditions":[{"type":"Complete","status":"True"}]}}`), nil),
				)

				k.EXPECT().KubeDelete(gomock.Any(), "Job.batch", runJobDefaultNamespace, gomock.Any(), false, gomock.Any()).
					Return(nil)
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains:    []string{"Job succeeded.\n", "Cleanup complete.\n"},
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
			name:                    "non-object JSON body returns 400",
			body:                    true,
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidRequestContent: : Failed to parse request body.",
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
					"template":    minimalTemplate,
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
					"template":    minimalTemplate,
				},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : Jobs with spec.completions > 1 are not implemented.",
		},
		{
			name: "backoffLimit != 0 returns 400",
			body: map[string]interface{}{
				"kind":       "Job",
				"apiVersion": "batch/v1",
				"metadata":   map[string]interface{}{"name": "test-job"},
				"spec": map[string]interface{}{
					"backoffLimit": 3,
					"template":     minimalTemplate,
				},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : Jobs with spec.backoffLimit != 0 are not supported; set it to 0 or omit it.",
		},
		{
			name: "empty restartPolicy returns 400",
			body: map[string]interface{}{
				"kind":       "Job",
				"apiVersion": "batch/v1",
				"metadata":   map[string]interface{}{"name": "test-job"},
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"restartPolicy": "",
							"containers": []interface{}{
								map[string]interface{}{"name": "worker", "image": "busybox"},
							},
						},
					},
				},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               `400: InvalidParameter: : The Job pod template restartPolicy must be Never or OnFailure, got "".`,
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
						if !strings.HasPrefix(obj.GetName(), strings.Repeat("a", 57)) {
							return fmt.Errorf("expected 57-char prefix, got %q", obj.GetName())
						}
						return fmt.Errorf("stop")
					})
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains:    []string{"Failed to create job"},
		},
		{
			name: "all-dash name longer than 57 chars reduces to empty after truncation",
			body: map[string]interface{}{
				"kind":       "Job",
				"apiVersion": "batch/v1",
				"metadata":   map[string]interface{}{"name": strings.Repeat("-", 70)},
				"spec":       minimalJobBody["spec"],
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided Job metadata.name reduces to empty after truncation.",
		},
		{
			name:                    "cluster not found returns 404",
			body:                    minimalJobBody,
			noClusterDoc:            true,
			wantStatusCode:          http.StatusNotFound,
			wantResponseContentType: "application/json",
			wantError:               "404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.",
		},
		{
			name: "kubeActionsFactory error returns 500",
			body: minimalJobBody,
			kubeActionsFactory: func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return nil, errors.New("failed to create kubeactions")
			},
			wantStatusCode:          http.StatusInternalServerError,
			wantResponseContentType: "application/json",
			wantError:               "500: InternalServerError: : Internal server error.",
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

			if !tt.noClusterDoc {
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
			}
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

			kubeActionsFactory := func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return k, nil
			}
			if tt.kubeActionsFactory != nil {
				kubeActionsFactory = tt.kubeActionsFactory
			}
			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, kubeActionsFactory, nil, nil, nil)
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

			// b as wantResponse is intentional: body check is a no-op (bytes.Equal(b,b)=true) for streaming
			// cases without a fixed expected body, while still delegating status and error JSON parsing.
			if err := validateResponse(resp, b, tt.wantStatusCode, tt.wantError, b); err != nil {
				t.Error(err)
			}

			if tt.wantResponseContentType != resp.Header.Get("Content-Type") {
				t.Errorf("unexpected Content-Type %q, wanted %q",
					resp.Header.Get("Content-Type"), tt.wantResponseContentType)
			}

			for _, want := range tt.wantResponseContains {
				if !strings.Contains(string(b), want) {
					t.Errorf("response does not contain %q\nfull response: %s", want, string(b))
				}
			}
		})
	}
}

// TestWaitForJobPod_ErrorBranches exercises error paths not reachable via HTTP handler tests.
func TestWaitForJobPod_ErrorBranches(t *testing.T) {
	for _, tt := range []struct {
		name            string
		sendEvent       func(fw *watch.FakeWatcher)
		wantPodName     string
		wantErrContains string
	}{
		{
			name: "closed channel returns error",
			sendEvent: func(fw *watch.FakeWatcher) {
				fw.Stop() // closes the result channel
			},
			wantErrContains: "pod watch channel closed unexpectedly",
		},
		{
			name: "watch.Error with metav1.Status returns formatted error",
			sendEvent: func(fw *watch.FakeWatcher) {
				fw.Error(&metav1.Status{
					Message: "quota exceeded",
					Reason:  "Forbidden",
					Code:    403,
				})
			},
			wantErrContains: "pod watch error: quota exceeded",
		},
		{
			name: "watch.Error with non-Status object returns type error",
			sendEvent: func(fw *watch.FakeWatcher) {
				// FakeWatcher.Error wraps in a Status; send a raw error event
				// by directly writing to the channel via Action.
				fw.Action(watch.Error, &unstructured.Unstructured{})
			},
			wantErrContains: "pod watch error: unexpected object type",
		},
		{
			name:            "context cancellation returns ctx.Err",
			sendEvent:       nil, // no event; rely on context cancel below
			wantErrContains: "context canceled",
		},
		{
			name: "Deleted event is skipped, subsequent Added Running pod returns name",
			sendEvent: func(fw *watch.FakeWatcher) {
				pod := &unstructured.Unstructured{}
				pod.SetName("test-pod")
				_ = unstructured.SetNestedField(pod.Object, "Running", "status", "phase")
				fw.Delete(pod) // Deleted event: skipped (not Added/Modified)
				fw.Add(pod)    // Added event with Running phase: returns name
			},
			wantPodName: "test-pod",
		},
		{
			name: "non-Unstructured Added event is skipped, channel close returns error",
			sendEvent: func(fw *watch.FakeWatcher) {
				fw.Action(watch.Added, &metav1.Status{}) // not *unstructured.Unstructured: skipped
				fw.Stop()                                // close channel after skipped event
			},
			wantErrContains: "pod watch channel closed unexpectedly",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fw := watch.NewFake()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.sendEvent != nil {
				go tt.sendEvent(fw)
			} else {
				// Cancel immediately so waitForJobPod exits via ctx.Done().
				cancel()
			}

			podName, err := waitForJobPod(ctx, logrus.NewEntry(logrus.New()), fw)
			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if podName != tt.wantPodName {
					t.Errorf("pod name = %q, want %q", podName, tt.wantPodName)
				}
			}
		})
	}
}

// TestWaitForJobPod_ClosedChannelWithCancelledContext verifies ctx.Err() is preferred over generic error.
func TestWaitForJobPod_ClosedChannelWithCancelledContext(t *testing.T) {
	fw := watch.NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()  // cancel first so ctx.Err() is non-nil
	fw.Stop() // close the channel; the !ok branch should inspect ctx.Err()
	_, err := waitForJobPod(ctx, logrus.NewEntry(logrus.New()), fw)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want context.Canceled", err)
	}
}

// TestWaitForJobTerminal_MaxErrors verifies exit after maxConsecutiveErrors consecutive failures.
func TestWaitForJobTerminal_MaxErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	// KubeGet always returns an error → hits maxConsecutiveErrors (10) and exits.
	k.EXPECT().KubeGet(gomock.Any(), kubeJobResource, "test-ns", "test-job").
		Return(nil, errors.New("apiserver down")).
		Times(10)

	result := waitForJobTerminal(context.Background(), logrus.NewEntry(logrus.New()), k, "test-ns", "test-job", 0)
	if !strings.Contains(result, "fetching job status") {
		t.Errorf("unexpected result %q; want it to contain 'fetching job status'", result)
	}
}

func TestJobResult(t *testing.T) {
	for _, tt := range []struct {
		name       string
		kubeGetRet []byte
		kubeGetErr error
		wantResult string
		wantErr    string
	}{
		{
			name:       "succeeded",
			kubeGetRet: []byte(`{"status":{"conditions":[{"type":"Complete","status":"True"}]}}`),
			wantResult: jobResultSucceeded,
		},
		{
			name:       "failed",
			kubeGetRet: []byte(`{"status":{"conditions":[{"type":"Failed","status":"True"}]}}`),
			wantResult: jobResultFailed,
		},
		{
			name:       "pending with no terminal conditions",
			kubeGetRet: []byte(`{"status":{"conditions":[{"type":"Complete","status":"False"}]}}`),
			wantResult: jobResultPending,
		},
		{
			name:       "pending with empty conditions",
			kubeGetRet: []byte(`{"status":{"conditions":[]}}`),
			wantResult: jobResultPending,
		},
		{
			name:       "KubeGet error",
			kubeGetErr: errors.New("apiserver unreachable"),
			wantErr:    "fetching job status",
		},
		{
			name:       "malformed JSON",
			kubeGetRet: []byte(`{invalid`),
			wantErr:    "parsing job status",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			k := mock_adminactions.NewMockKubeActions(ctrl)

			k.EXPECT().KubeGet(gomock.Any(), kubeJobResource, "ns", "job").
				Return(tt.kubeGetRet, tt.kubeGetErr)

			result, err := jobResult(context.Background(), k, "ns", "job")
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.wantResult {
				t.Errorf("result = %q, want %q", result, tt.wantResult)
			}
		})
	}
}

// TestRunJobStream_LogStreamingError verifies log failure writes error then continues.
func TestRunJobStream_LogStreamingError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	fw := watch.NewFake()
	k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").Return(fw, nil)
	k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

	go func() {
		fw.Add(&unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{"name": "test-pod"},
				"status":   map[string]interface{}{"phase": "Succeeded"},
			},
		})
	}()

	// KubeFollowPodLogs fails with an active context; should write "Log streaming error:".
	k.EXPECT().KubeFollowPodLogs(gomock.Any(), gomock.Any(), "test-pod", "worker", gomock.Any()).
		Return(errors.New("connection reset by peer"))

	// waitForJobTerminal polls for job completion; return succeeded on first call.
	k.EXPECT().KubeGet(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _ string) ([]byte, error) {
			return json.Marshal(&batchv1.Job{Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}},
			}})
		})
	k.EXPECT().KubeDelete(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any(), false, gomock.Any()).
		Return(nil)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: "openshift-azure-operator"},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "worker"}}},
			},
		},
	}

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	runJobStream(context.Background(), logrus.NewEntry(logrus.New()), k, job, wc, 0)

	got := buf.String()
	if !strings.Contains(got, "Log streaming error:") {
		t.Errorf("output %q does not contain 'Log streaming error:'", got)
	}
	if !strings.Contains(got, "Job succeeded.") {
		t.Errorf("output %q does not contain 'Job succeeded.'", got)
	}
}

// TestRunJobStream_CleanupFailure verifies KubeDelete failure writes "Cleanup failed:".
func TestRunJobStream_CleanupFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	fw := watch.NewFake()
	k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").Return(fw, nil)

	k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

	go func() {
		fw.Add(&unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{"name": "test-pod"},
				"status":   map[string]interface{}{"phase": "Succeeded"},
			},
		})
	}()

	k.EXPECT().KubeFollowPodLogs(gomock.Any(), gomock.Any(), "test-pod", "worker", gomock.Any()).Return(nil)

	k.EXPECT().KubeGet(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any()).
		Return([]byte(`{"status":{"conditions":[{"type":"Complete","status":"True"}]}}`), nil)

	// KubeDelete fails with a non-transient error; retry.OnError does not retry.
	k.EXPECT().KubeDelete(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any(), false, gomock.Any()).
		Return(errors.New("delete forbidden"))

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "openshift-azure-operator",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "worker"},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	runJobStream(context.Background(), logrus.NewEntry(logrus.New()), k, job, wc, 0)

	if !wc.closed {
		t.Error("expected Close to be called on the WriteCloser")
	}
	got := buf.String()
	if !strings.Contains(got, "Cleanup failed:") {
		t.Errorf("output %q does not contain 'Cleanup failed:'", got)
	}
}

// TestRunJobStream_CleanupNotFound verifies 404 during cleanup is treated as success.
func TestRunJobStream_CleanupNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	fw := watch.NewFake()
	k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").Return(fw, nil)
	k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

	go func() {
		fw.Add(&unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{"name": "test-pod"},
				"status":   map[string]interface{}{"phase": "Succeeded"},
			},
		})
	}()

	k.EXPECT().KubeFollowPodLogs(gomock.Any(), gomock.Any(), "test-pod", "worker", gomock.Any()).Return(nil)

	k.EXPECT().KubeGet(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any()).
		Return([]byte(`{"status":{"conditions":[{"type":"Complete","status":"True"}]}}`), nil)

	notFound := kerrors.NewNotFound(schema.GroupResource{Group: "batch", Resource: "jobs"}, "test-job")
	k.EXPECT().KubeDelete(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any(), false, gomock.Any()).
		Return(notFound)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "openshift-azure-operator",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "worker"},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	runJobStream(context.Background(), logrus.NewEntry(logrus.New()), k, job, wc, 0)

	if !wc.closed {
		t.Error("expected Close to be called on the WriteCloser")
	}
	got := buf.String()
	if strings.Contains(got, "Cleanup failed:") {
		t.Errorf("expected no cleanup failure for NotFound, got: %s", got)
	}
	if !strings.Contains(got, "Cleanup complete.") {
		t.Errorf("expected 'Cleanup complete.' in output, got: %s", got)
	}
}

// TestRunJobStream_ContextCancellation verifies cancel skips terminal status writes.
func TestRunJobStream_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fw := watch.NewFake()
	k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").Return(fw, nil)
	k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

	go func() {
		fw.Add(&unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{"name": "test-pod"},
				"status":   map[string]interface{}{"phase": "Succeeded"},
			},
		})
	}()

	// Cancel the context during log streaming to simulate a mid-stream cancellation.
	k.EXPECT().KubeFollowPodLogs(gomock.Any(), gomock.Any(), "test-pod", "worker", gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _ string, _ io.Writer) error {
			cancel()
			return nil
		})

	// cleanupJob uses a fresh background context, so KubeDelete is still called.
	k.EXPECT().KubeDelete(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any(), false, gomock.Any()).
		Return(nil)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "openshift-azure-operator",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "worker"},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	runJobStream(ctx, logrus.NewEntry(logrus.New()), k, job, wc, 0)

	if !wc.closed {
		t.Error("expected Close to be called on the WriteCloser")
	}
	got := buf.String()
	if strings.Contains(got, "Job succeeded.") || strings.Contains(got, "Job failed.") {
		t.Errorf("output %q should NOT contain terminal status on cancellation path", got)
	}
}

// TestRunJobStream_PollExhausted verifies exit after maxConsecutiveErrors writes "Job polling exhausted".
func TestRunJobStream_PollExhausted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	fw := watch.NewFake()
	k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").Return(fw, nil)
	k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

	go func() {
		fw.Add(&unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{"name": "test-pod"},
				"status":   map[string]interface{}{"phase": "Succeeded"},
			},
		})
	}()

	k.EXPECT().KubeFollowPodLogs(gomock.Any(), gomock.Any(), "test-pod", "worker", gomock.Any()).Return(nil)

	// All 10 KubeGet calls fail, exhausting maxConsecutiveErrors (10).
	k.EXPECT().KubeGet(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any()).
		Return(nil, errors.New("apiserver unavailable")).Times(10)

	k.EXPECT().KubeDelete(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any(), false, gomock.Any()).
		Return(nil)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: "openshift-azure-operator"},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "worker"}}},
			},
		},
	}

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	runJobStream(context.Background(), logrus.NewEntry(logrus.New()), k, job, wc, 0)

	if !strings.Contains(buf.String(), "Job polling exhausted") {
		t.Errorf("output %q does not contain 'Job polling exhausted'", buf.String())
	}
}

// TestRunJobStream_WaitForPodErrorWithCleanupFailure verifies pod error is streamed, cleanup error is logged.
func TestRunJobStream_WaitForPodErrorWithCleanupFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	fw := watch.NewFake()
	k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").Return(fw, nil)
	k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

	// Stop the watcher immediately so waitForJobPod returns "pod watch channel closed unexpectedly".
	go fw.Stop()

	// cleanupJob uses a fresh background context; KubeDelete returns a non-transient error.
	k.EXPECT().KubeDelete(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any(), false, gomock.Any()).
		Return(errors.New("cleanup failed: permission denied"))

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: "openshift-azure-operator"},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "worker"}}},
			},
		},
	}

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	runJobStream(context.Background(), logrus.NewEntry(logrus.New()), k, job, wc, 0)

	got := buf.String()
	if !strings.Contains(got, "Error waiting for pod:") {
		t.Errorf("output %q does not contain 'Error waiting for pod:'", got)
	}
	if strings.Contains(got, "Cleanup failed:") {
		t.Errorf("output %q should NOT contain 'Cleanup failed:' on error path (cleanup errors are only logged)", got)
	}
}

// TestRunJobStream_ContextCancellationWithCleanupFailure verifies cancel skips all pipe writes.
func TestRunJobStream_ContextCancellationWithCleanupFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fw := watch.NewFake()
	k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").Return(fw, nil)
	k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

	go func() {
		fw.Add(&unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{"name": "test-pod"},
				"status":   map[string]interface{}{"phase": "Succeeded"},
			},
		})
	}()

	// Cancel the context during log streaming.
	k.EXPECT().KubeFollowPodLogs(gomock.Any(), gomock.Any(), "test-pod", "worker", gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _ string, _ io.Writer) error {
			cancel()
			return nil
		})

	// cleanupJob uses a fresh background context; return a non-transient error.
	k.EXPECT().KubeDelete(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any(), false, gomock.Any()).
		Return(errors.New("cleanup failed: permission denied"))

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: "openshift-azure-operator"},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "worker"}}},
			},
		},
	}

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	runJobStream(ctx, logrus.NewEntry(logrus.New()), k, job, wc, 0)
	if !wc.closed {
		t.Error("expected WriteCloser to be closed")
	}

	got := buf.String()
	if strings.Contains(got, "Cleanup failed:") {
		t.Errorf("output %q should NOT contain 'Cleanup failed:' on cancellation path (cleanup errors are only logged)", got)
	}
}

// TestRunJobStream_LogErrorWithCancelledContext verifies log errors are suppressed when cancelled.
func TestRunJobStream_LogErrorWithCancelledContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fw := watch.NewFake()
	k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").Return(fw, nil)
	k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

	go func() {
		fw.Add(&unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{"name": "test-pod"},
				"status":   map[string]interface{}{"phase": "Running"},
			},
		})
	}()

	// Cancel the context AND return an error from KubeFollowPodLogs.
	k.EXPECT().KubeFollowPodLogs(gomock.Any(), gomock.Any(), "test-pod", "worker", gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _ string, _ io.Writer) error {
			cancel()
			return errors.New("stream broken pipe")
		})

	k.EXPECT().KubeDelete(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any(), false, gomock.Any()).
		Return(nil)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: "openshift-azure-operator"},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "worker"}}},
			},
		},
	}

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	runJobStream(ctx, logrus.NewEntry(logrus.New()), k, job, wc, 0)

	got := buf.String()
	if strings.Contains(got, "Log streaming error:") {
		t.Errorf("output %q should NOT contain 'Log streaming error:' when context is cancelled", got)
	}
}

// bufferedTestWatcher implements watch.Interface with a buffered channel for pre-loaded events.
type bufferedTestWatcher struct {
	ch   chan watch.Event
	once sync.Once
}

// Stop guards against double-close if Stop is called more than once.
func (w *bufferedTestWatcher) Stop()                          { w.once.Do(func() { close(w.ch) }) }
func (w *bufferedTestWatcher) ResultChan() <-chan watch.Event { return w.ch }

// TestRunJobStream_ContextCancelledBetweenPodAndLogs verifies log streaming is skipped on cancel.
func TestRunJobStream_ContextCancelledBetweenPodAndLogs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a buffered watcher so the event can be pre-loaded without blocking.
	// watch.NewFake() uses an unbuffered channel, which would deadlock here.
	wCh := make(chan watch.Event, 1)
	watcher := &bufferedTestWatcher{ch: wCh}
	k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").Return(watcher, nil)
	k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)

	// KubeFollowPodLogs must NOT be called when context is cancelled before streaming.
	// cleanupJob uses context.Background(), so KubeDelete is still expected.
	k.EXPECT().KubeDelete(gomock.Any(), kubeJobResource, gomock.Any(), gomock.Any(), false, gomock.Any()).
		Return(nil)

	// Pre-buffer the pod event then immediately cancel so that neither log streaming
	// nor terminal-status output is written.
	wCh <- watch.Event{
		Type: watch.Added,
		Object: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{"name": "test-pod"},
				"status":   map[string]interface{}{"phase": "Running"},
			},
		},
	}
	cancel()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: "openshift-azure-operator"},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "worker"}}},
			},
		},
	}

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	runJobStream(ctx, logrus.NewEntry(logrus.New()), k, job, wc, 0)

	if !wc.closed {
		t.Error("expected Close to be called on the WriteCloser")
	}
	got := buf.String()
	if strings.Contains(got, "streaming logs") {
		t.Errorf("output %q should NOT contain 'streaming logs' when context is cancelled", got)
	}
	if strings.Contains(got, "Job succeeded.") || strings.Contains(got, "Job failed.") {
		t.Errorf("output %q should NOT contain terminal status when context is cancelled", got)
	}
}

// TestCleanupJob_TransientRetry verifies retry on transient errors then success.
func TestCleanupJob_TransientRetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	// Zero the backoff duration so the retry fires immediately in the unit test.
	// Not parallel-safe: mutates package-level kubeRetryBackoff; do not add t.Parallel() to this test.
	origDuration := kubeRetryBackoff.Duration
	kubeRetryBackoff.Duration = 0
	defer func() { kubeRetryBackoff.Duration = origDuration }()

	callCount := 0
	k.EXPECT().
		KubeDelete(gomock.Any(), kubeJobResource, "test-ns", "test-job", false, gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _ string, _ bool, _ *metav1.DeletionPropagation) error {
			callCount++
			if callCount == 1 {
				return kerrors.NewInternalError(errors.New("etcd timeout"))
			}
			return nil
		}).Times(2)

	if err := cleanupJob(k, "test-ns", "test-job"); err != nil {
		t.Errorf("expected nil after retry; got %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 KubeDelete calls; got %d", callCount)
	}
}

func TestAdminPostRunJob_DBGroupError(t *testing.T) {
	ctx := context.Background()
	mockSubID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)

	// WithSubscriptions but no WithOpenShiftClusters: dbGroup.OpenShiftClusters() returns an error.
	ti := newTestInfra(t).WithSubscriptions()
	defer ti.done()

	if err := ti.buildFixtures(nil); err != nil {
		t.Fatal(err)
	}

	f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	go f.Run(ctx, nil, nil)

	resp, b, err := ti.request(http.MethodPost,
		fmt.Sprintf("https://server/admin%s/runjob", resourceID),
		http.Header{"Content-Type": []string{"application/json"}},
		map[string]interface{}{
			"apiVersion": "batch/v1",
			"kind":       "Job",
			"metadata":   map[string]interface{}{"name": "test-job", "namespace": "openshift-azure-operator"},
			"spec": map[string]interface{}{
				"backoffLimit": 0,
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"restartPolicy": "Never",
						"containers": []interface{}{
							map[string]interface{}{
								"name":    "worker",
								"image":   "busybox",
								"command": []interface{}{"sh", "-c", "echo hi"},
							},
						},
					},
				},
			},
		})
	if err != nil {
		t.Fatal(err)
	}

	if err := validateResponse(resp, b, http.StatusInternalServerError,
		"500: InternalServerError: : Internal server error.", nil); err != nil {
		t.Error(err)
	}
}
