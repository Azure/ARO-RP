package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
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
	mockTenantID := mockSubID
	method := http.MethodPost
	ctx := context.Background()

	type test struct {
		name                    string
		namespace               string
		podName                 string
		container               string
		command                 string
		useBody                 bool
		bodyData                interface{}
		noClusterDoc            bool
		kubeActionsFactory      func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error)
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
			wantResponse:            []byte("Executing in openshift-etcd/etcd-master-0/etcdctl...\nhello\nDone.\n\n"), // trailing \n appended by ReplyStream (adminreplies.go)
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
			name:      "exec writes stderr and Command failed when command exits non-zero",
			namespace: "openshift-etcd",
			podName:   "etcd-master-0",
			container: "etcdctl",
			command:   "bad-cmd",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeExecStream(gomock.Any(), tt.namespace, tt.podName, tt.container,
						[]string{"sh", "-c", tt.command}, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, _, stderr io.Writer) error {
						_, _ = fmt.Fprint(stderr, "error output\n")
						return errors.New("exit code 1")
					})
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponse:            []byte("Executing in openshift-etcd/etcd-master-0/etcdctl...\nstderr:\nerror output\nCommand failed: exit code 1\n\n"),
		},
		{
			name:      "stdout larger than 1 MiB is truncated with notice",
			namespace: "openshift-etcd",
			podName:   "etcd-master-0",
			container: "etcdctl",
			command:   "dd if=/dev/zero bs=2M count=1",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeExecStream(gomock.Any(), tt.namespace, tt.podName, tt.container,
						[]string{"sh", "-c", tt.command}, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, stdout, _ io.Writer) error {
						// Write 2 MiB to trigger the 1 MiB truncation limit.
						_, _ = fmt.Fprint(stdout, strings.Repeat("x", 2*1024*1024))
						return nil
					})
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponse: []byte("Executing in openshift-etcd/etcd-master-0/etcdctl...\n" +
				strings.Repeat("x", 1<<20) +
				"\n[stdout truncated at 1 MiB]\n" +
				"Done.\n\n"),
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
		{
			name:                    "cluster not found returns 404",
			namespace:               "openshift-etcd",
			podName:                 "etcd-master-0",
			container:               "etcdctl",
			command:                 "ls",
			noClusterDoc:            true,
			wantStatusCode:          http.StatusNotFound,
			wantResponseContentType: "application/json",
			wantError:               "404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.",
		},
		{
			name:                    "empty body returns 400",
			useBody:                 true,
			bodyData:                nil,
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidRequestContent: : The request body must not be empty.",
		},
		{
			name:                    "non-object JSON body returns 400",
			useBody:                 true,
			bodyData:                true, // marshals to JSON `true`; unmarshal into struct fails
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidRequestContent: : Failed to parse request body.",
		},
		{
			name:                    "invalid namespace format returns 400",
			namespace:               "invalid!namespace",
			podName:                 "etcd-master-0",
			container:               "etcdctl",
			command:                 "ls",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided namespace 'invalid!namespace' is invalid.",
		},
		{
			name:                    "invalid pod name format returns 400",
			namespace:               "openshift-etcd",
			podName:                 "bad pod!",
			container:               "etcdctl",
			command:                 "ls",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided pod name 'bad pod!' is invalid.",
		},
		{
			name:                    "invalid container name format returns 400",
			namespace:               "openshift-etcd",
			podName:                 "etcd-master-0",
			container:               "bad_container!",
			command:                 "ls",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided container name 'bad_container!' is invalid.",
		},
		{
			name:                    "command exceeding 4096 bytes returns 400",
			namespace:               "openshift-etcd",
			podName:                 "etcd-master-0",
			container:               "etcdctl",
			command:                 strings.Repeat("x", 4097),
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided command must not exceed 4096 bytes.",
		},
		{
			name:      "kubeActionsFactory error returns 500",
			namespace: "openshift-etcd",
			podName:   "etcd-master-0",
			container: "etcdctl",
			command:   "ls",
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

			var reqBody interface{}
			if tt.useBody {
				reqBody = tt.bodyData
			} else {
				reqBody = map[string]interface{}{
					"namespace": tt.namespace,
					"podName":   tt.podName,
					"container": tt.container,
					"command":   tt.command,
				}
			}
			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/admin%s/exec", resourceID),
				http.Header{"Content-Type": []string{"application/json"}},
				reqBody)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
			if tt.wantResponseContentType != resp.Header.Get("Content-Type") {
				t.Errorf("unexpected \"Content-Type\" response header value %q, wanted %q",
					resp.Header.Get("Content-Type"), tt.wantResponseContentType)
			}
		})
	}
}

// TestExecContainerStream_StderrSuppressedOnCancellation verifies stderr is suppressed on cancel.
func TestExecContainerStream_StderrSuppressedOnCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k.EXPECT().
		KubeExecStream(gomock.Any(), "openshift-etcd", "etcd-master-0", "etcdctl",
			[]string{"sh", "-c", "sleep 60"}, gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _, _, _ string, _ []string, _, stderr io.Writer) error {
			_, _ = stderr.Write([]byte("partial error output\n"))
			cancel()
			return ctx.Err()
		})

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	execContainerStream(ctx, logrus.NewEntry(logrus.New()), k, "openshift-etcd", "etcd-master-0", "etcdctl", []string{"sh", "-c"}, "sleep 60", wc)

	if !wc.closed {
		t.Error("expected Close to be called on the WriteCloser")
	}
	if strings.Contains(buf.String(), "stderr:") {
		t.Errorf("output %q must not contain 'stderr:' when context is cancelled", buf.String())
	}
}

// TestExecContainerStream_ContextCancellation verifies cancel causes a clean return.
func TestExecContainerStream_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k.EXPECT().
		KubeExecStream(gomock.Any(), "openshift-etcd", "etcd-master-0", "etcdctl",
			[]string{"sh", "-c", "sleep 60"}, gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _, _, _ string, _ []string, _, _ io.Writer) error {
			cancel()
			return ctx.Err()
		})

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	execContainerStream(ctx, logrus.NewEntry(logrus.New()), k, "openshift-etcd", "etcd-master-0", "etcdctl", []string{"sh", "-c"}, "sleep 60", wc)

	if !wc.closed {
		t.Error("expected Close to be called on the WriteCloser")
	}
}

// TestExecContainerStream_NilReturnAfterCancel verifies that when KubeExecStream returns nil
// but the context was already cancelled, "Done." is not written to the stream.
func TestExecContainerStream_NilReturnAfterCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	k := mock_adminactions.NewMockKubeActions(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k.EXPECT().
		KubeExecStream(gomock.Any(), "openshift-etcd", "etcd-master-0", "etcdctl",
			[]string{"sh", "-c", "sleep 60"}, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, _, _ io.Writer) error {
			cancel()
			return nil // nil error, but context is now cancelled
		})

	var buf bytes.Buffer
	wc := &testWriteCloser{Buffer: &buf}
	execContainerStream(ctx, logrus.NewEntry(logrus.New()), k, "openshift-etcd", "etcd-master-0", "etcdctl", []string{"sh", "-c"}, "sleep 60", wc)

	if !wc.closed {
		t.Error("expected Close to be called on the WriteCloser")
	}
	if strings.Contains(buf.String(), "Done.") {
		t.Errorf("output %q must not contain 'Done.' when context is cancelled after nil return", buf.String())
	}
}

func TestAdminPostExec_DBGroupError(t *testing.T) {
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
		fmt.Sprintf("https://server/admin%s/exec", resourceID),
		http.Header{"Content-Type": []string{"application/json"}},
		map[string]interface{}{
			"namespace": "openshift-etcd",
			"podName":   "etcd-master-0",
			"container": "etcdctl",
			"command":   "ls",
		})
	if err != nil {
		t.Fatal(err)
	}

	if err := validateResponse(resp, b, http.StatusInternalServerError,
		"500: InternalServerError: : Internal server error.", nil); err != nil {
		t.Error(err)
	}
}
