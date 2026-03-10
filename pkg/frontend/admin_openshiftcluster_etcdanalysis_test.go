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

	corev1 "k8s.io/api/core/v1"
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
		vmName                  string
		mocks                   func(*test, *mock_adminactions.MockKubeActions)
		wantStatusCode          int
		wantResponseContentType string
		wantError               string
		wantResponseContains    []string
	}

	for _, tt := range []*test{
		{
			name:   "successful analysis streams snapshot creation and job output",
			vmName: "master-0",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				// Job watcher is set up before the ordered sequence so it can be
				// referenced from within the DoAndReturn closures.
				fakeWatcher := watch.NewFake()

				gomock.InOrder(
					// Snapshot exec
					k.EXPECT().
						KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
							gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, stdout, _ io.Writer) error {
							_, _ = fmt.Fprint(stdout, "Snapshot saved.\n")
							return nil
						}),

					// SA creation: ServiceAccount, ClusterRole, ClusterRoleBinding, SCC (in order)
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),

					// Job watch is registered before the job is created
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
						}),

					// Job creation
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),

					k.EXPECT().KubeFollowPodLogs(gomock.Any(), namespaceEtcds, "analysis-pod", "analyzer", gomock.Any()).
						DoAndReturn(func(_ context.Context, _, _, _ string, w io.Writer) error {
							_, _ = fmt.Fprint(w, "Analysis report output.\n")
							return nil
						}),

					k.EXPECT().KubeGet(gomock.Any(), "Job.batch", namespaceEtcds, gomock.Any()).
						Return([]byte(`{"status":{"conditions":[{"type":"Complete","status":"True"}]}}`), nil),

					// Job cleanup (Job.batch, force=false, foreground propagation)
					k.EXPECT().KubeDelete(gomock.Any(), "Job.batch", namespaceEtcds, gomock.Any(), false, gomock.Any()).
						Return(nil),

					// SA cleanup: ServiceAccount, SecurityContextConstraints, ClusterRole, ClusterRoleBinding
					// (force=true, nil propagation - from createPrivilegedServiceAccount cleanup func)
					k.EXPECT().KubeDelete(gomock.Any(), "ServiceAccount", namespaceEtcds, gomock.Any(), true, nil).
						Return(nil),
					k.EXPECT().KubeDelete(gomock.Any(), "SecurityContextConstraints", "", gomock.Any(), true, nil).
						Return(nil),
					k.EXPECT().KubeDelete(gomock.Any(), "ClusterRole", "", gomock.Any(), true, nil).
						Return(nil),
					k.EXPECT().KubeDelete(gomock.Any(), "ClusterRoleBinding", "", gomock.Any(), true, nil).
						Return(nil),

					// Snapshot cleanup (always deferred, rm -f is a no-op on success)
					k.EXPECT().
						KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
							gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil),
				)
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
			name:   "snapshot exec failure streams error and does not start job",
			vmName: "master-0",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				// Snapshot exec fails
				k.EXPECT().
					KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
						gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("connection refused"))

				// Snapshot cleanup still runs (defer registered before the exec)
				k.EXPECT().
					KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
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
			// Item 3A (MA-5): SA cleanup (KubeDelete for ServiceAccount) returns an error.
			// The deferred cleanup func from createPrivilegedServiceAccount calls KubeDelete
			// for SA, SCC, ClusterRole, and CRB. When the SA delete fails, the combined error
			// is written to the stream as "SA cleanup failed: ...".
			name:   "SA cleanup failure streams error message",
			vmName: "master-0",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				gomock.InOrder(
					// Snapshot exec succeeds
					k.EXPECT().
						KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
							gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, stdout, _ io.Writer) error {
							_, _ = fmt.Fprint(stdout, "Snapshot saved.\n")
							return nil
						}),

					// SA creation: ServiceAccount, ClusterRole, ClusterRoleBinding, SCC
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),

					// Job watch registration
					k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").
						DoAndReturn(func(_ context.Context, _ *unstructured.Unstructured, _ string) (watch.Interface, error) {
							fakeWatcher := watch.NewFake()
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
						}),

					// Job creation
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),

					k.EXPECT().KubeFollowPodLogs(gomock.Any(), namespaceEtcds, "analysis-pod", "analyzer", gomock.Any()).
						DoAndReturn(func(_ context.Context, _, _, _ string, w io.Writer) error {
							_, _ = fmt.Fprint(w, "Analysis report output.\n")
							return nil
						}),

					k.EXPECT().KubeGet(gomock.Any(), "Job.batch", namespaceEtcds, gomock.Any()).
						Return([]byte(`{"status":{"conditions":[{"type":"Complete","status":"True"}]}}`), nil),

					// Job cleanup succeeds
					k.EXPECT().KubeDelete(gomock.Any(), "Job.batch", namespaceEtcds, gomock.Any(), false, gomock.Any()).
						Return(nil),

					// SA cleanup: ServiceAccount delete fails; remaining deletes succeed.
					// createPrivilegedServiceAccount cleanup calls in order:
					// ServiceAccount, SecurityContextConstraints, ClusterRole, ClusterRoleBinding.
					k.EXPECT().KubeDelete(gomock.Any(), "ServiceAccount", namespaceEtcds, gomock.Any(), true, nil).
						Return(errors.New("permission denied")),
					k.EXPECT().KubeDelete(gomock.Any(), "SecurityContextConstraints", "", gomock.Any(), true, nil).
						Return(nil),
					k.EXPECT().KubeDelete(gomock.Any(), "ClusterRole", "", gomock.Any(), true, nil).
						Return(nil),
					k.EXPECT().KubeDelete(gomock.Any(), "ClusterRoleBinding", "", gomock.Any(), true, nil).
						Return(nil),

					// Snapshot cleanup (deferred)
					k.EXPECT().
						KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
							gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil),
				)
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains: []string{
				"SA cleanup failed:",
			},
		},
		{
			// Item 3D (mi-3): SA setup succeeds but Job KubeCreateOrUpdate fails.
			// The response should contain the error text from the failed job creation.
			name:   "job creation failure after SA setup streams error",
			vmName: "master-0",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				gomock.InOrder(
					// Snapshot exec succeeds
					k.EXPECT().
						KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
							gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, stdout, _ io.Writer) error {
							_, _ = fmt.Fprint(stdout, "Snapshot saved.\n")
							return nil
						}),

					// SA creation: ServiceAccount, ClusterRole, ClusterRoleBinding, SCC - all succeed
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),

					// Job watch registration
					k.EXPECT().KubeWatch(gomock.Any(), gomock.Any(), "batch.kubernetes.io/job-name").
						Return(watch.NewFake(), nil),

					// Job creation fails
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).
						Return(errors.New("job creation error: quota exceeded")),

					// SA cleanup: all succeed (deferred cleanup still runs)
					k.EXPECT().KubeDelete(gomock.Any(), "ServiceAccount", namespaceEtcds, gomock.Any(), true, nil).
						Return(nil),
					k.EXPECT().KubeDelete(gomock.Any(), "SecurityContextConstraints", "", gomock.Any(), true, nil).
						Return(nil),
					k.EXPECT().KubeDelete(gomock.Any(), "ClusterRole", "", gomock.Any(), true, nil).
						Return(nil),
					k.EXPECT().KubeDelete(gomock.Any(), "ClusterRoleBinding", "", gomock.Any(), true, nil).
						Return(nil),

					// Snapshot cleanup (deferred)
					k.EXPECT().
						KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
							gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil),
				)
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponseContains: []string{
				"Snapshot created. Starting analysis job...\n",
				"job creation error: quota exceeded",
			},
		},
		{
			name:                    "missing vmName returns 400",
			vmName:                  "",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided vmName '' is invalid.",
		},
		{
			name:                    "invalid vmName returns 400",
			vmName:                  "bad node!",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided vmName 'bad node!' is invalid.",
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

			u := fmt.Sprintf("https://server/admin%s/etcdanalysis?vmName=%s", resourceID, url.QueryEscape(tt.vmName))
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

func TestBuildEtcdAnalysisJob(t *testing.T) {
	job := buildEtcdAnalysisJob("master-0", "etcd_analysis_123.snapshot", "etcd-analysis-privileged-abcdefgh")

	if !strings.HasPrefix(job.Name, "etcd-analysis-") {
		t.Errorf("job.Name %q does not have prefix 'etcd-analysis-'", job.Name)
	}
	if job.Namespace != namespaceEtcds {
		t.Errorf("job.Namespace = %q, want %q", job.Namespace, namespaceEtcds)
	}
	if job.Spec.BackoffLimit == nil || *job.Spec.BackoffLimit != 0 {
		t.Errorf("job.Spec.BackoffLimit = %v, want pointer to 0", job.Spec.BackoffLimit)
	}
	if job.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyNever {
		t.Errorf("RestartPolicy = %q, want Never", job.Spec.Template.Spec.RestartPolicy)
	}
	if job.Spec.Template.Spec.ServiceAccountName != "etcd-analysis-privileged-abcdefgh" {
		t.Errorf("ServiceAccountName = %q, want 'etcd-analysis-privileged-abcdefgh'", job.Spec.Template.Spec.ServiceAccountName)
	}
	if got := job.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"]; got != "master-0" {
		t.Errorf("NodeSelector[kubernetes.io/hostname] = %q, want 'master-0'", got)
	}
	if len(job.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("len(Containers) = %d, want 1", len(job.Spec.Template.Spec.Containers))
	}
	c := job.Spec.Template.Spec.Containers[0]
	if c.Name != "analyzer" {
		t.Errorf("container.Name = %q, want 'analyzer'", c.Name)
	}
	if c.Image != etcdAnalysisImage {
		t.Errorf("container.Image = %q, want %q", c.Image, etcdAnalysisImage)
	}
	wantCommand := []string{"/usr/local/bin/analyze-snapshot.sh"}
	if len(c.Command) != len(wantCommand) {
		t.Errorf("container.Command = %v, want %v", c.Command, wantCommand)
	} else {
		for i, v := range wantCommand {
			if c.Command[i] != v {
				t.Errorf("container.Command[%d] = %q, want %q", i, c.Command[i], v)
			}
		}
	}
	if len(c.Args) < 2 {
		t.Errorf("container.Args = %v, want [--delete /snapshot/<filename>]", c.Args)
	} else {
		if c.Args[0] != "--delete" {
			t.Errorf("container.Args[0] = %q, want %q", c.Args[0], "--delete")
		}
		if !strings.Contains(c.Args[1], "etcd_analysis_123.snapshot") {
			t.Errorf("container.Args[1] = %q, want it to contain snapshot filename %q",
				c.Args[1], "etcd_analysis_123.snapshot")
		}
	}
	if len(c.VolumeMounts) != 1 || c.VolumeMounts[0].MountPath != "/snapshot" {
		t.Errorf("VolumeMounts = %+v, want one mount at /snapshot", c.VolumeMounts)
	}
	if len(job.Spec.Template.Spec.Volumes) != 1 {
		t.Fatalf("len(Volumes) = %d, want 1", len(job.Spec.Template.Spec.Volumes))
	}
	v := job.Spec.Template.Spec.Volumes[0]
	if v.Name != "etcd-data" {
		t.Errorf("volume.Name = %q, want 'etcd-data'", v.Name)
	}
	if v.HostPath == nil || v.HostPath.Path != etcdDataDir {
		t.Errorf("HostPath.Path = %v, want %q", v.HostPath, etcdDataDir)
	}
}
