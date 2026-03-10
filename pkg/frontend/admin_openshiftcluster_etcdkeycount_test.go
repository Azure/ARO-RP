package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestAdminPostEtcdKeyCount(t *testing.T) {
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
		wantResponse            []byte
	}

	for _, tt := range []*test{
		{
			name:   "successful key count streams output",
			vmName: "master-0",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
						[]string{"sh", "-c", etcdKeyCountScript}, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, stdout, _ io.Writer) error {
						_, _ = fmt.Fprint(stdout, "42 default\n18 kube-system\n")
						return nil
					})
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponse:            []byte("Executing in openshift-etcd/etcd-master-0/etcdctl...\n42 default\n18 kube-system\nDone.\n"),
		},
		{
			name:   "exec failure streams Command failed",
			vmName: "master-0",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
						[]string{"sh", "-c", etcdKeyCountScript}, gomock.Any(), gomock.Any()).
					Return(errors.New("connection refused"))
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponse:            []byte("Executing in openshift-etcd/etcd-master-0/etcdctl...\nCommand failed: connection refused\n"),
		},
		{
			name:   "stderr output is prefixed with stderr header",
			vmName: "master-0",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
						[]string{"sh", "-c", etcdKeyCountScript}, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, _, stderr io.Writer) error {
						_, _ = fmt.Fprint(stderr, "etcdctl: connection refused\n")
						return nil
					})
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponse:            []byte("Executing in openshift-etcd/etcd-master-0/etcdctl...\nstderr:\netcdctl: connection refused\nDone.\n"),
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

			u := fmt.Sprintf("https://server/admin%s/etcdkeycount?vmName=%s", resourceID, url.QueryEscape(tt.vmName))
			resp, b, err := ti.request(method, u, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
			if tt.wantResponseContentType != resp.Header.Get("Content-Type") {
				t.Errorf("unexpected Content-Type %q, wanted %q",
					resp.Header.Get("Content-Type"), tt.wantResponseContentType)
			}
		})
	}
}

func TestEtcdKeyCountScriptSyntax(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available: skipping script syntax check")
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	scriptPath := filepath.Join(filepath.Dir(thisFile), "scripts", "etcdkeycount.sh")

	cmd := exec.Command(bashPath, "-n", scriptPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("bash -n %s failed: %v\n%s", scriptPath, err, out)
	}
}
