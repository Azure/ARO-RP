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
	mockTenantID := mockSubID
	method := http.MethodPost
	ctx := context.Background()

	type test struct {
		name                    string
		vmName                  string
		noClusterDoc            bool
		kubeActionsFactory      func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error)
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
						[]string{"bash", "-c", etcdKeyCountScript}, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, stdout, _ io.Writer) error {
						_, _ = fmt.Fprint(stdout, "42 default\n18 kube-system\n")
						return nil
					})
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponse:            []byte(fmt.Sprintf("Executing in %s/etcd-master-0/%s...\n42 default\n18 kube-system\nDone.\n\n", namespaceEtcds, etcdContainerName)),
		},
		{
			name:   "exec failure streams Command failed",
			vmName: "master-0",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
						[]string{"bash", "-c", etcdKeyCountScript}, gomock.Any(), gomock.Any()).
					Return(errors.New("connection refused"))
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponse:            []byte(fmt.Sprintf("Executing in %s/etcd-master-0/%s...\nCommand failed: connection refused\n\n", namespaceEtcds, etcdContainerName)),
		},
		{
			name:   "stderr output is prefixed with stderr header",
			vmName: "master-0",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeExecStream(gomock.Any(), namespaceEtcds, "etcd-master-0", etcdContainerName,
						[]string{"bash", "-c", etcdKeyCountScript}, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, _ []string, _, stderr io.Writer) error {
						_, _ = fmt.Fprint(stderr, "etcdctl: connection refused\n")
						return nil
					})
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponse:            []byte(fmt.Sprintf("Executing in %s/etcd-master-0/%s...\nstderr:\netcdctl: connection refused\nDone.\n\n", namespaceEtcds, etcdContainerName)),
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
		{
			name:                    "cluster not found returns 404",
			vmName:                  "master-0",
			noClusterDoc:            true,
			wantStatusCode:          http.StatusNotFound,
			wantResponseContentType: "application/json",
			wantError:               "404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.",
		},
		{
			name:   "kubeActionsFactory error returns 500",
			vmName: "master-0",
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

func TestAdminPostEtcdKeyCount_DBGroupError(t *testing.T) {
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
		fmt.Sprintf("https://server/admin%s/etcdkeycount?vmName=master-0", resourceID),
		nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := validateResponse(resp, b, http.StatusInternalServerError,
		"500: InternalServerError: : Internal server error.", nil); err != nil {
		t.Error(err)
	}
}

func TestEtcdKeyCountScriptSyntax(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available: skipping script syntax check")
	}

	cmd := exec.Command(bashPath, "-n", "-c", etcdKeyCountScript)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("bash -n -c <embedded script> failed: %v\n%s", err, out)
	}
}
