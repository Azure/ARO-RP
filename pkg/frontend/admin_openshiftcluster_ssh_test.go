package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminOpenShiftClusterSSHNewElevated(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	resourcePath := strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName"))
	clusterResourceGroupID := "/subscriptions/" + mockSubID + "/resourceGroups/test-cluster"
	masterSubnetID := clusterResourceGroupID + "/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master"
	privateEndpointIP := "10.0.0.4"
	internalLoadBalancerIP := "10.0.0.5"
	sshPoolID := func(loadBalancerName, poolName string) string {
		return fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/loadBalancers/%s/backendAddressPools/%s", mockSubID, loadBalancerName, poolName)
	}
	nicInfo := func(nicID string, poolIDs []string) *adminactions.NetworkInterfaceSSHInfo {
		return &adminactions.NetworkInterfaceSSHInfo{
			ID:             nicID,
			BackendPoolIDs: poolIDs,
			SubnetIDs:      []string{masterSubnetID},
		}
	}
	expectRoute := func(a *mock_adminactions.MockAzureActions, loadBalancerName, poolID, nicID string, port int32, status *adminactions.SSHRouteStatus) {
		a.EXPECT().GetSSHRouteStatus(
			gomock.Any(),
			"test-cluster",
			clusterResourceGroupID,
			loadBalancerName,
			poolID,
			nicID,
			internalLoadBalancerIP,
			port,
		).Return(status, nil)
	}

	// One host key per test run. Real deployments load this from the portal
	// keyvault; the test just needs any RSA pubkey the KnownHosts helper
	// can render.
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	hostPub, err := cryptossh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	expectedKnownHostLine := knownhosts.Line([]string{"eastus.admin.aro.azure.com"}, hostPub)

	type test struct {
		name                string
		systemDataHeader    string
		body                interface{}
		contentType         string // "" => header omitted
		omitHostKey         bool
		omitClusterDoc      bool
		azureMock           func(*mock_adminactions.MockAzureActions)
		injectPortalError   error
		wantStatusCode      int
		wantError           string
		wantUsername        string
		wantMaster          int
		wantVMName          string
		wantPort            int
		architectureVersion *api.ArchitectureVersion
	}

	for _, tt := range []*test{
		{
			name:             "missing target returns 400",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{},
			contentType:      "application/json",
			wantStatusCode:   http.StatusBadRequest,
			wantError:        "400: InvalidParameter: : Either vmName or master is required.",
		},
		{
			name:             "malformed VM name returns 400",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master;bad"},
			contentType:      "application/json",
			wantStatusCode:   http.StatusBadRequest,
			wantError:        "400: InvalidParameter: : The provided vmName 'aro-infra-master;bad' is invalid.",
		},
		{
			name:             "master 0 issued by SRE",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(0)},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-0", mgmtcompute.InstanceView).Return(sshMasterVM("PowerState/running"), nil)
			},
			wantStatusCode: http.StatusOK,
			wantUsername:   "sre@redhat.com",
			wantMaster:     0,
		},
		{
			name:             "master 2 issued by SRE",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(2)},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-2", mgmtcompute.InstanceView).Return(sshMasterVM("PowerState/running"), nil)
			},
			wantStatusCode: http.StatusOK,
			wantUsername:   "sre@redhat.com",
			wantMaster:     2,
		},
		{
			name:             "falls back to createdBy when lastModifiedBy is empty",
			systemDataHeader: `{"createdBy":"creator@redhat.com"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(1)},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-1", mgmtcompute.InstanceView).Return(sshMasterVM("PowerState/running"), nil)
			},
			wantStatusCode: http.StatusOK,
			wantUsername:   "creator@redhat.com",
			wantMaster:     1,
		},
		{
			name:             "missing SystemData header returns 400",
			systemDataHeader: "",
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(0)},
			contentType:      "application/json",
			wantStatusCode:   http.StatusBadRequest,
			wantError:        "400: InvalidParameter: : The caller identity could not be determined from the request.",
		},
		{
			name:             "portal host key unavailable returns 503",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(0)},
			contentType:      "application/json",
			omitHostKey:      true,
			wantStatusCode:   http.StatusServiceUnavailable,
			wantError:        "503: InternalServerError: : Portal SSH host key is not available; the SSH endpoint is disabled.",
		},
		{
			name:             "wrong Content-Type returns 415",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(0)},
			contentType:      "text/plain",
			wantStatusCode:   http.StatusUnsupportedMediaType,
			wantError:        "415: UnsupportedMediaType: : The content media type 'text/plain' is not supported. Only 'application/json' is supported.",
		},
		{
			name:             "master out of range returns 400",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(3)},
			contentType:      "application/json",
			wantStatusCode:   http.StatusBadRequest,
			wantError:        "400: InvalidParameter: master: master must be 0, 1, or 2.",
		},
		{
			name:             "cosmos create failure surfaces 500",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(0)},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-0", mgmtcompute.InstanceView).Return(sshMasterVM("PowerState/running"), nil)
			},
			injectPortalError: errors.New("simulated cosmos write failure"),
			wantStatusCode:    http.StatusInternalServerError,
			wantError:         "500: InternalServerError: : simulated cosmos write failure",
		},
		{
			name:             "cluster not found returns 404 without minting a portal document",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(0)},
			contentType:      "application/json",
			omitClusterDoc:   true,
			wantStatusCode:   http.StatusNotFound,
			wantError:        "404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.",
		},
		{
			name:             "identity with shell metacharacters returns 400",
			systemDataHeader: `{"lastModifiedBy":"evil;rm -rf@redhat.com"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(0)},
			contentType:      "application/json",
			wantStatusCode:   http.StatusBadRequest,
			wantError:        "400: InvalidParameter: : The caller identity contains unsupported characters.",
		},
		{
			name:             "empty local-part identity returns 400",
			systemDataHeader: `{"lastModifiedBy":"@contoso"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(0)},
			contentType:      "application/json",
			wantStatusCode:   http.StatusBadRequest,
			wantError:        "400: InvalidParameter: : The caller identity contains unsupported characters.",
		},
		{
			name:             "deallocated master returns 400",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(1)},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-1", mgmtcompute.InstanceView).Return(sshMasterVM("PowerState/deallocated"), nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: master: master-1 is not running (PowerState/deallocated); power it on or choose a running master.",
		},
		{
			name:             "azure power-state lookup error is non-fatal",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: pointerutils.ToPtr(0)},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-0", mgmtcompute.InstanceView).Return(mgmtcompute.VirtualMachine{}, errors.New("azure boom"))
			},
			wantStatusCode: http.StatusOK,
			wantUsername:   "sre@redhat.com",
			wantMaster:     0,
		},
		{
			name:             "CPMS master resolves from NIC pool",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master-r4nd0-1"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				poolID := sshPoolID("aro-infra-internal", "ssh-1")
				nicID := "/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/aro-infra-master-r4nd0-1-nic"
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-r4nd0-1", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", nicID), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "aro-infra-master-r4nd0-1-nic").Return(nicInfo(nicID, []string{poolID}), nil)
				expectRoute(a, "aro-infra-internal", poolID, nicID, 2201, &adminactions.SSHRouteStatus{
					BackendPoolNetworkInterfaceIDs: []string{nicID},
					LoadBalancingRuleCount:         1,
				})
			},
			wantStatusCode: http.StatusOK,
			wantUsername:   "sre@redhat.com",
			wantMaster:     1,
			wantVMName:     "aro-infra-master-r4nd0-1",
			wantPort:       2201,
		},
		{
			name:             "bootstrap resolves to port 2199",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-bootstrap"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				poolID := sshPoolID("aro-infra-internal", "bootstrap-ssh")
				nicID := "/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/aro-infra-bootstrap-nic"
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-bootstrap", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", nicID), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "aro-infra-bootstrap-nic").Return(nicInfo(nicID, []string{poolID}), nil)
				expectRoute(a, "aro-infra-internal", poolID, nicID, 2199, &adminactions.SSHRouteStatus{
					BackendPoolNetworkInterfaceIDs: []string{nicID},
					LoadBalancingRuleCount:         1,
				})
			},
			wantStatusCode: http.StatusOK,
			wantUsername:   "sre@redhat.com",
			wantMaster:     -1,
			wantVMName:     "aro-infra-bootstrap",
			wantPort:       2199,
		},
		{
			name:                "V1 master resolves from V1 ILB pool",
			systemDataHeader:    `{"lastModifiedBy":"sre@redhat.com"}`,
			body:                &adminSSHRequest{VMName: "aro-infra-master-0"},
			contentType:         "application/json",
			architectureVersion: pointerutils.ToPtr(api.ArchitectureVersionV1),
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				poolID := sshPoolID("aro-infra-internal-lb", "ssh-0")
				nicID := "/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/master-nic"
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-0", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", nicID), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "master-nic").Return(nicInfo(nicID, []string{poolID}), nil)
				expectRoute(a, "aro-infra-internal-lb", poolID, nicID, 2200, &adminactions.SSHRouteStatus{
					BackendPoolNetworkInterfaceIDs: []string{nicID},
					LoadBalancingRuleCount:         1,
				})
			},
			wantStatusCode: http.StatusOK,
			wantUsername:   "sre@redhat.com",
			wantVMName:     "aro-infra-master-0",
			wantPort:       2200,
		},
		{
			name:             "exact VM must be running",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master-r4nd0-1"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-r4nd0-1", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/deallocated", "/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/nic"), nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: vmName: VM "aro-infra-master-r4nd0-1" is not running (PowerState/deallocated).`,
		},
		{
			name:             "master-like VM outside master subnet is rejected",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master-backdoor"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				nicID := "/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/backdoor-nic"
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-backdoor", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", nicID), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "backdoor-nic").Return(&adminactions.NetworkInterfaceSSHInfo{
					ID:        nicID,
					SubnetIDs: []string{clusterResourceGroupID + "/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker"},
				}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: vmName: VM "aro-infra-master-backdoor" is not attached to the cluster master subnet.`,
		},
		{
			name:             "exact VM not found returns 404",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master-missing"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-missing", mgmtcompute.InstanceView).
					Return(mgmtcompute.VirtualMachine{}, autorest.DetailedError{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: NotFound: vmName: Virtual machine "aro-infra-master-missing" was not found.`,
		},
		{
			name:             "exact VM backend failure returns 500",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master-r4nd0-1"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-r4nd0-1", mgmtcompute.InstanceView).
					Return(mgmtcompute.VirtualMachine{}, errors.New("azure VM service unavailable"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : azure VM service unavailable",
		},
		{
			name:             "NIC not found returns 404",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master-r4nd0-1"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-r4nd0-1", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", "/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/missing-nic"), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "missing-nic").
					Return(nil, &azcore.ResponseError{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: NotFound: vmName: Network interface "missing-nic" for VM "aro-infra-master-r4nd0-1" was not found.`,
		},
		{
			name:             "NIC backend failure returns 500",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master-r4nd0-1"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-r4nd0-1", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", "/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/master-nic"), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "master-nic").
					Return(nil, errors.New("azure NIC service unavailable"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : azure NIC service unavailable",
		},
		{
			name:             "master without SSH pool is rejected",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master-r4nd0-1"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-r4nd0-1", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", "/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/nic"), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "nic").Return(nicInfo("/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/nic", []string{"/loadBalancers/lb/backendAddressPools/api"}), nil)
			},
			wantStatusCode: http.StatusConflict,
			wantError:      `409: RequestNotAllowed: vmName: Exact SSH routing is unavailable for VM "aro-infra-master-r4nd0-1": its network interface is not attached to an SSH backend pool on load balancer "aro-infra-internal". Wait for control-plane SSH reconciliation to complete.`,
		},
		{
			name:             "same pool name on unrelated load balancer is rejected",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master-r4nd0-1"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-r4nd0-1", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", "/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/nic"), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "nic").Return(nicInfo("/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/nic", []string{sshPoolID("other", "ssh-1")}), nil)
			},
			wantStatusCode: http.StatusConflict,
			wantError:      `409: RequestNotAllowed: vmName: Exact SSH routing is unavailable for VM "aro-infra-master-r4nd0-1": its network interface is not attached to an SSH backend pool on load balancer "aro-infra-internal". Wait for control-plane SSH reconciliation to complete.`,
		},
		{
			name:             "duplicate matching pool IDs are rejected",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master-r4nd0-1"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				poolID := sshPoolID("aro-infra-internal", "ssh-1")
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-r4nd0-1", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", "/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/nic"), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "nic").Return(nicInfo("/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/nic", []string{poolID, poolID}), nil)
			},
			wantStatusCode: http.StatusConflict,
			wantError:      `409: RequestNotAllowed: vmName: Exact SSH routing is unavailable for VM "aro-infra-master-r4nd0-1": its network interface has 2 SSH backend pool mappings on load balancer "aro-infra-internal". Wait for control-plane SSH reconciliation to complete.`,
		},
		{
			name:             "shared master pool rejects exact routing",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-master-r4nd0-1"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				poolID := sshPoolID("aro-infra-internal", "ssh-1")
				nicID := "/subscriptions/sub/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/selected-nic"
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-master-r4nd0-1", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", nicID), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "selected-nic").Return(nicInfo(nicID, []string{poolID}), nil)
				expectRoute(a, "aro-infra-internal", poolID, nicID, 2201, &adminactions.SSHRouteStatus{
					BackendPoolNetworkInterfaceIDs: []string{nicID, "/subscriptions/" + mockSubID + "/resourceGroups/test-cluster/providers/Microsoft.Network/networkInterfaces/other-master"},
					LoadBalancingRuleCount:         1,
				})
			},
			wantStatusCode: http.StatusConflict,
			wantError:      `409: RequestNotAllowed: vmName: Exact SSH routing is unavailable for VM "aro-infra-master-r4nd0-1": backend pool has 2 network interfaces and does not uniquely target the selected VM. Wait for control-plane SSH reconciliation to complete.`,
		},
		{
			name:             "bootstrap without diagnostic pool is unavailable",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-bootstrap"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-bootstrap", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", "/networkInterfaces/bootstrap"), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "bootstrap").Return(nicInfo("/networkInterfaces/bootstrap", []string{}), nil)
			},
			wantStatusCode: http.StatusConflict,
			wantError:      `409: RequestNotAllowed: vmName: Bootstrap SSH is unavailable for VM "aro-infra-bootstrap": its network interface is not attached to an SSH backend pool on load balancer "aro-infra-internal". Run or re-run install-failure diagnostics to configure the bootstrap SSH route.`,
		},
		{
			name:             "bootstrap without LB rule is unavailable",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{VMName: "aro-infra-bootstrap"},
			contentType:      "application/json",
			azureMock: func(a *mock_adminactions.MockAzureActions) {
				poolID := sshPoolID("aro-infra-internal", "bootstrap-ssh")
				nicID := "/networkInterfaces/bootstrap"
				a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "aro-infra-bootstrap", mgmtcompute.InstanceView).
					Return(sshTargetVM("PowerState/running", nicID), nil)
				a.EXPECT().GetNetworkInterfaceSSHInfo(gomock.Any(), "test-cluster", "bootstrap").Return(nicInfo(nicID, []string{poolID}), nil)
				expectRoute(a, "aro-infra-internal", poolID, nicID, 2199, &adminactions.SSHRouteStatus{
					BackendPoolNetworkInterfaceIDs: []string{nicID},
					LoadBalancingRuleCount:         0,
				})
			},
			wantStatusCode: http.StatusConflict,
			wantError:      `409: RequestNotAllowed: vmName: Bootstrap SSH is unavailable for VM "aro-infra-bootstrap": load balancer "aro-infra-internal" has 0 matching SSH rules. Run or re-run install-failure diagnostics to configure the bootstrap SSH route.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)

			ti := newTestInfra(t).WithPortal().WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			azureActions := mock_adminactions.NewMockAzureActions(controller)
			if tt.azureMock != nil {
				tt.azureMock(azureActions)
			}

			if !tt.omitClusterDoc {
				architectureVersion := api.ArchitectureVersionV2
				if tt.architectureVersion != nil {
					architectureVersion = *tt.architectureVersion
				}
				ti.fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: resourcePath,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   resourcePath,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.OpenShiftClusterProperties{
							InfraID:             "aro-infra",
							ArchitectureVersion: architectureVersion,
							APIServerProfile:    api.APIServerProfile{IntIP: internalLoadBalancerIP},
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterResourceGroupID,
							},
							MasterProfile: api.MasterProfile{SubnetID: masterSubnetID},
							NetworkProfile: api.NetworkProfile{
								APIServerPrivateEndpointIP: privateEndpointIP,
							},
						},
					},
				})
				ti.fixture.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockSubID,
						},
					},
				})
				if err := ti.buildFixtures(nil); err != nil {
					t.Fatal(err)
				}
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil,
				func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error) {
					return azureActions, nil
				}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			if !tt.omitHostKey {
				f.portalSSHHostPubKey = hostPub
			}

			if tt.injectPortalError != nil {
				ti.portalClient.SetError(tt.injectPortalError)
			}

			go f.Run(ctx, nil, nil)

			header := http.Header{}
			if tt.systemDataHeader != "" {
				header.Set("X-Ms-Arm-Resource-System-Data", tt.systemDataHeader)
			}
			if tt.contentType != "" {
				header.Set("Content-Type", tt.contentType)
			}

			resp, b, err := ti.request(http.MethodPost,
				"https://server/admin"+resourcePath+"/ssh/newelevated",
				header, tt.body)
			if err != nil {
				t.Fatal(err)
			}

			if tt.injectPortalError != nil {
				ti.portalClient.SetError(nil)
			}

			if tt.wantError != "" {
				if err := validateResponse(resp, b, tt.wantStatusCode, tt.wantError, nil); err != nil {
					t.Error(err)
				}
				docs, listErr := ti.portalClient.ListAll(ctx, nil)
				if listErr != nil {
					t.Fatal(listErr)
				}
				if len(docs.PortalDocuments) != 0 {
					t.Errorf("expected 0 portal documents persisted, got %d", len(docs.PortalDocuments))
				}
				return
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Fatalf("unexpected status code %d, wanted %d: %s", resp.StatusCode, tt.wantStatusCode, string(b))
			}

			var got adminSSHResponse
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("response is not a valid adminSSHResponse: %v\nbody: %s", err, string(b))
			}
			if got.Password != firstPortalUUID {
				t.Errorf("response Password: got %q, want %q", got.Password, firstPortalUUID)
			}
			// Command must pin the injected host key; the LHS-of-@ username
			// is what the portal SSH proxy'll match on PasswordCallback.
			wantUserPrefix := strings.SplitN(tt.wantUsername, "@", 2)[0]
			if !strings.Contains(got.Command, expectedKnownHostLine) {
				t.Errorf("response Command missing KnownHosts pin\ncommand: %s\nwant substring: %s", got.Command, expectedKnownHostLine)
			}
			if !strings.Contains(got.Command, wantUserPrefix+"@eastus.admin.aro.azure.com") {
				t.Errorf("response Command missing %q@host suffix\ncommand: %s", wantUserPrefix, got.Command)
			}

			docs, err := ti.portalClient.ListAll(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(docs.PortalDocuments) != 1 {
				t.Fatalf("expected 1 portal document persisted, got %d", len(docs.PortalDocuments))
			}
			doc := docs.PortalDocuments[0]
			if doc.ID != firstPortalUUID {
				t.Errorf("portal doc ID: got %q, want %q", doc.ID, firstPortalUUID)
			}
			if doc.Portal == nil {
				t.Fatal("portal doc has nil Portal payload")
			}
			if doc.Portal.ID != resourcePath {
				t.Errorf("portal doc Portal.ID: got %q, want %q", doc.Portal.ID, resourcePath)
			}
			if doc.Portal.Username != tt.wantUsername {
				t.Errorf("portal doc Portal.Username: got %q, want %q", doc.Portal.Username, tt.wantUsername)
			}
			if doc.Portal.SSH == nil {
				t.Fatal("portal doc Portal.SSH is nil")
			}
			if doc.Portal.SSH.Master != tt.wantMaster {
				t.Errorf("portal doc SSH.Master: got %d, want %d", doc.Portal.SSH.Master, tt.wantMaster)
			}
			if doc.Portal.SSH.VMName != tt.wantVMName {
				t.Errorf("portal doc SSH.VMName: got %q, want %q", doc.Portal.SSH.VMName, tt.wantVMName)
			}
			if doc.Portal.SSH.Port != tt.wantPort {
				t.Errorf("portal doc SSH.Port: got %d, want %d", doc.Portal.SSH.Port, tt.wantPort)
			}
			if doc.Portal.SSH.Authenticated {
				t.Errorf("portal doc SSH.Authenticated: got true, want false")
			}
			if doc.Portal.Kubeconfig != nil {
				t.Errorf("portal doc Portal.Kubeconfig should be nil, got %+v", doc.Portal.Kubeconfig)
			}
			if doc.TTL != 60 {
				t.Errorf("portal doc TTL: got %d seconds, want 60 (1m)", doc.TTL)
			}
		})
	}
}

func sshTargetVM(powerCode, nicID string) mgmtcompute.VirtualMachine {
	vm := sshMasterVM(powerCode)
	interfaces := []mgmtcompute.NetworkInterfaceReference{{ID: pointerutils.ToPtr(nicID)}}
	vm.NetworkProfile = &mgmtcompute.NetworkProfile{NetworkInterfaces: &interfaces}
	return vm
}

func sshMasterVM(powerCode string) mgmtcompute.VirtualMachine {
	statuses := []mgmtcompute.InstanceViewStatus{
		{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
		{Code: pointerutils.ToPtr(powerCode)},
	}
	return mgmtcompute.VirtualMachine{
		VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
			InstanceView: &mgmtcompute.VirtualMachineInstanceView{
				Statuses: &statuses,
			},
		},
	}
}
