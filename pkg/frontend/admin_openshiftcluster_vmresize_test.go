package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminVMResize(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	addClusterDoc := func(f *testdatabase.Fixture) {
		f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
			Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
			OpenShiftCluster: &api.OpenShiftCluster{
				ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
					},
				},
			},
		})
	}

	addSubscriptionDoc := func(f *testdatabase.Fixture) {
		f.AddSubscriptionDocuments(&api.SubscriptionDocument{
			ID: mockSubID,
			Subscription: &api.Subscription{
				State: api.SubscriptionStateRegistered,
				Properties: &api.SubscriptionProperties{
					TenantID: mockTenantID,
				},
			},
		})
	}

	type test struct {
		name               string
		resourceID         string
		vmName             string
		vmSize             string
		fixture            func(f *testdatabase.Fixture)
		azureActionsMocks  func(*test, *mock_adminactions.MockAzureActions)
		kubeActionsMocks   func(*test, *mock_adminactions.MockKubeActions)
		wantStatusCode     int
		wantResponse       []byte
		wantError          string
		kubeActionsFactory func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error)
	}

	for _, tt := range []*test{
		{
			name:       "basic coverage",
			vmName:     "aro-fake-node-master-0",
			vmSize:     "Standard_D8s_v3",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				addClusterDoc(f)
				addSubscriptionDoc(f)
			},
			azureActionsMocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().ResourceGroupHasVM(gomock.Any(), tt.vmName).Return(true, nil)
				a.EXPECT().VMResize(gomock.Any(), tt.vmName, tt.vmSize).Return(nil)
			},
			kubeActionsMocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeGet(gomock.Any(), "machine", "openshift-machine-api", tt.vmName).
					Return(encodeMachine(t, mockMachine(tt.vmName, true, true)), nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:       "cluster not found",
			vmName:     "aro-fake-node-master-0",
			vmSize:     "Standard_D8s_v3",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				addSubscriptionDoc(f)
			},
			azureActionsMocks: func(tt *test, a *mock_adminactions.MockAzureActions) {},
			kubeActionsMocks:  func(tt *test, k *mock_adminactions.MockKubeActions) {},
			wantStatusCode:    http.StatusNotFound,
			wantError:         `404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.`,
		},
		{
			name:       "subscription doc not found",
			vmName:     "aro-fake-node-master-0",
			vmSize:     "Standard_D8s_v3",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				addClusterDoc(f)
			},
			azureActionsMocks: func(tt *test, a *mock_adminactions.MockAzureActions) {},
			kubeActionsMocks:  func(tt *test, k *mock_adminactions.MockKubeActions) {},
			wantStatusCode:    http.StatusBadRequest,
			wantError:         fmt.Sprintf(`400: InvalidSubscriptionState: : Request is not allowed in unregistered subscription '%s'.`, mockSubID),
		},
		{
			name:       "master node not found",
			vmName:     "aro-fake-node-master-0",
			vmSize:     "Standard_D8s_v3",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				addClusterDoc(f)
				addSubscriptionDoc(f)
			},
			azureActionsMocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().ResourceGroupHasVM(gomock.Any(), tt.vmName).Return(false, nil)
			},
			kubeActionsMocks: func(tt *test, k *mock_adminactions.MockKubeActions) {},
			wantStatusCode:   http.StatusNotFound,
			wantError:        `404: NotFound: : "The VirtualMachine 'aro-fake-node-master-0' under resource group 'resourcegroup' was not found."`,
		},
		{
			name:       "not a control plane machine",
			vmName:     "aro-fake-node-0",
			vmSize:     "Standard_D8s_v3",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				addClusterDoc(f)
				addSubscriptionDoc(f)
			},
			azureActionsMocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().ResourceGroupHasVM(gomock.Any(), tt.vmName).Return(true, nil)
			},
			kubeActionsMocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeGet(gomock.Any(), "machine", "openshift-machine-api", tt.vmName).
					Return(encodeMachine(t, mockMachine(tt.vmName, false, true)), nil)
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      `403: Forbidden: : "The vmName 'aro-fake-node-0' provided cannot be resized. It is not a control plane machine."`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions().WithOpenShiftClusters()
			defer ti.done()

			a := mock_adminactions.NewMockAzureActions(ti.controller)
			tt.azureActionsMocks(tt, a)

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			k := mock_adminactions.NewMockKubeActions(ti.controller)
			tt.kubeActionsMocks(tt, k)

			kubeActionsFactory := func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return k, nil
			}
			if tt.kubeActionsFactory != nil {
				kubeActionsFactory = tt.kubeActionsFactory
			}

			f, err := NewFrontend(ctx,
				ti.auditLog,
				ti.log,
				ti.otelAudit,
				ti.env,
				ti.dbGroup,
				api.APIs,
				&noop.Noop{},
				&noop.Noop{},
				nil,
				nil,
				nil,
				nil,
				kubeActionsFactory,
				func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error) {
					return a, nil
				},
				nil,
				nil)

			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodPost,
				fmt.Sprintf("https://server/admin%s/resize?vmName=%s&vmSize=%s", tt.resourceID, tt.vmName, tt.vmSize),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func encodeMachine(t *testing.T, machine *machinev1beta1.Machine) []byte {
	buf := &bytes.Buffer{}
	err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(machine)
	if err != nil {
		t.Fatalf("%s failed to encode machine, %s", t.Name(), err.Error())
	}
	return buf.Bytes()
}

func mockMachine(name string, isMaster bool, hasRole bool) *machinev1beta1.Machine {
	labels := map[string]string{}
	if hasRole {
		labels = map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"}
		if isMaster {
			labels = map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"}
		}
	}

	return &machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "openshift-machine-api",
			Labels:    labels,
		},
		Spec: machinev1beta1.MachineSpec{
			ProviderSpec: machinev1beta1.ProviderSpec{
				Value: &kruntime.RawExtension{
					Raw: []byte(`{
"apiVersion": "machine.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
}`)},
			},
		},
	}
}
