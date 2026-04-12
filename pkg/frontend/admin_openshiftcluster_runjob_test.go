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
			name: "backoffLimit != 0 returns 400",
			body: map[string]interface{}{
				"kind":       "Job",
				"apiVersion": "batch/v1",
				"metadata":   map[string]interface{}{"name": "test-job"},
				"spec": map[string]interface{}{
					"backoffLimit": 3,
					"template":     minimalJobBody["spec"].(map[string]interface{})["template"],
				},
			},
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : Jobs with spec.backoffLimit != 0 are not supported; set it to 0 or omit it.",
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

// TestWaitForJobPod_ErrorBranches exercises the error paths in waitForJobPod
// that are not reachable through the full HTTP handler tests.
func TestWaitForJobPod_ErrorBranches(t *testing.T) {
	for _, tt := range []struct {
		name            string
		sendEvent       func(fw *watch.FakeWatcher)
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

			_, err := waitForJobPod(ctx, fw)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

// TestWaitForJobTerminal_MaxErrors verifies that waitForJobTerminal exits after
// maxConsecutiveErrors consecutive KubeGet failures and returns the error string.
func TestWaitForJobTerminal_MaxErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	// KubeGet always returns an error → hits maxConsecutiveErrors (10) and exits.
	k.EXPECT().KubeGet(gomock.Any(), kubeJobResource, "test-ns", "test-job").
		Return(nil, errors.New("apiserver down")).
		Times(10)

	result := waitForJobTerminal(context.Background(), logrus.NewEntry(logrus.New()), k, "test-ns", "test-job", 0)
	if !strings.Contains(result, "could not fetch job status") {
		t.Errorf("unexpected result %q; want it to contain 'could not fetch job status'", result)
	}
}

// TestRunJobStream_CleanupFailure verifies that a KubeDelete failure during
// cleanup writes "Cleanup failed:" to the stream.
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

// TestRunJobStream_CleanupNotFound verifies that a KubeDelete 404 during
// cleanup is treated as success (no "Cleanup failed:" in output).
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

// TestRunJobStream_ContextCancellation verifies that cancelling the context
// while runJobStream is streaming pod logs causes "Request cancelled." to be
// written to the output stream.
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
	if !strings.Contains(got, "Request cancelled.") {
		t.Errorf("output %q does not contain 'Request cancelled.'", got)
	}
}

// testWriteCloser is a simple WriteCloser backed by a bytes.Buffer for testing.
type testWriteCloser struct {
	*bytes.Buffer
	closed bool
}

func (w *testWriteCloser) Close() error {
	w.closed = true
	return nil
}
